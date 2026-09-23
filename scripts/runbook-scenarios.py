#!/usr/bin/env python3
"""
Run runbook scenarios against a live hub (deterministic action steps, no LLM).

Examples:
  ./scripts/runbook-scenarios.py --list
  ./scripts/runbook-scenarios.py --scenario health-check-branch
  make runbook-scenario SCENARIO=health-check-branch
  RUNBOOK_LIVE_EMAIL=1 make runbook-scenario SCENARIO=notify-email-smtp
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SCENARIOS_DIR = ROOT / "scenarios" / "runbook"


def api(base: str, method: str, path: str, body: dict | None = None) -> dict | list:
    url = base.rstrip("/") + path
    data = None
    headers = {"Content-Type": "application/json"}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            raw = resp.read().decode("utf-8")
            if not raw.strip():
                return {}
            return json.loads(raw)
    except urllib.error.HTTPError as e:
        detail = e.read().decode("utf-8", errors="replace")
        # 204 No Content is success for workspace ack.
        if e.code == 204:
            return {}
        raise RuntimeError(f"{method} {path} -> {e.code}: {detail}") from e


def resolve_agents(base: str, names: list[str]) -> list[str]:
    raw = api(base, "GET", "/api/agents")
    if isinstance(raw, list):
        agents = raw
    elif isinstance(raw, dict):
        agents = raw.get("agents") or []
    else:
        agents = []
    by_name = {a.get("name", "").lower(): a.get("id") for a in agents if isinstance(a, dict)}
    ids: list[str] = []
    for n in names:
        key = n.lstrip("@").lower()
        if key in by_name and by_name[key]:
            ids.append(by_name[key])
    if not ids:
        raise RuntimeError(f"Could not resolve agents {names}; known={sorted(by_name)}")
    return ids


def collab_tasks(snap: dict) -> list[dict]:
    return snap.get("tasks") or snap.get("collaboration", {}).get("tasks") or []


def find_task(tasks: list[dict], title_contains: str = "", task_id: str = "") -> dict | None:
    needle = (title_contains or "").lower()
    for t in tasks:
        if task_id and t.get("id") == task_id:
            return t
        if needle and needle in (t.get("title") or "").lower():
            return t
    return None


def expand_env_placeholders(obj):
    """Replace {{ENV:NAME}} strings with os.environ values (empty if unset)."""
    if isinstance(obj, dict):
        return {k: expand_env_placeholders(v) for k, v in obj.items()}
    if isinstance(obj, list):
        return [expand_env_placeholders(v) for v in obj]
    if isinstance(obj, str) and obj.startswith("{{ENV:") and obj.endswith("}}"):
        key = obj[len("{{ENV:") : -2]
        return os.environ.get(key, "")
    return obj


def run_scenario(base: str, scenario: dict, verbose: bool = False) -> None:
    channel = scenario.get("channel") or "runbook-scenarios"
    collab_id = ""
    saved_definition_id = ""
    saved_connector_id = ""
    agent_ids = resolve_agents(base, scenario.get("agents") or ["@Assistant"])

    for step in scenario.get("steps") or []:
        action = step.get("action")
        if verbose:
            print(f"  step: {action}")

        if action == "skip_unless_env":
            key = step.get("env") or ""
            if not key or os.environ.get(key, "").strip() in ("", "0", "false", "False"):
                print(f"SKIP: set {key}=1 to run this scenario")
                return
            continue

        if action == "create_connector":
            body = expand_env_placeholders({
                "type": step.get("type") or "sms",
                "label": step.get("label") or "scenario-connector",
                "config": step.get("config") or {},
                "secret": step.get("secret") or "",
            })
            if body.get("type") == "email":
                host = (body.get("config") or {}).get("host") or ""
                secret = body.get("secret") or ""
                if not host or not secret:
                    raise RuntimeError(
                        "create_connector email: set RUNBOOK_SMTP_HOST and RUNBOOK_SMTP_PASS "
                        "(and typically RUNBOOK_SMTP_USER / RUNBOOK_SMTP_FROM / RUNBOOK_SMTP_PORT)"
                    )
            created = api(base, "POST", "/api/connectors", body)
            saved_connector_id = created.get("id") or ""
            if not saved_connector_id:
                raise RuntimeError("create_connector: missing id")
            continue

        if action == "save_definition":
            fixture = step.get("fixture") or step.get("path")
            if not fixture:
                raise RuntimeError("save_definition: fixture path required")
            fixture_path = ROOT / fixture if not Path(fixture).is_absolute() else Path(fixture)
            def_body = json.loads(fixture_path.read_text(encoding="utf-8"))
            # Patch connector_id placeholders after create_connector.
            if saved_connector_id:
                raw = json.dumps(def_body).replace("{{connector_id}}", saved_connector_id)
                def_body = json.loads(raw)
            saved = api(base, "POST", "/api/runbook-definitions", def_body)
            saved_definition_id = saved.get("id") or def_body.get("id") or ""
            if not saved_definition_id:
                raise RuntimeError("save_definition: missing id in response")
            continue

        if action == "instantiate_definition":
            body = {
                "channel": channel,
                "created_by": "runbook-scenario",
                "agent_ids": agent_ids,
                "inputs": step.get("inputs") or {},
            }
            def_id = step.get("definition_id") or saved_definition_id
            if not def_id:
                raise RuntimeError("instantiate_definition: definition_id required")
            out = api(base, "POST", f"/api/runbook-definitions/{def_id}/instantiate", body)
            collab_id = out.get("collaboration_id") or ""
            if not collab_id:
                raise RuntimeError("instantiate_definition: missing collaboration_id")
            continue

        if action == "submit_runbook":
            api(base, "POST", f"/api/runbooks/{collab_id}/submit", {})
            continue

        if action == "start_runbook":
            body = {"inputs": step.get("inputs") or {}}
            api(base, "POST", f"/api/runbooks/{collab_id}/start", body)
            # Dispatch is gated on workspace ack for some runbooks; always ack in scenarios.
            try:
                api(base, "POST", "/api/collaboration-workspace-ack", {"collaboration_id": collab_id})
            except RuntimeError as e:
                # 204 No Content may come back as empty; urllib may still succeed via api().
                if "204" not in str(e) and "No Content" not in str(e):
                    # Some hubs return empty body with 204 — api() already handles empty as {}.
                    pass
            continue

        if action == "ack_workspace":
            api(base, "POST", "/api/collaboration-workspace-ack", {"collaboration_id": collab_id})
            continue

        if action == "wait_phase":
            want = step.get("phase", "executing")
            timeout = float(step.get("timeout_sec", 60))
            deadline = time.time() + timeout
            while time.time() < deadline:
                snap = api(base, "GET", f"/api/runbooks/{collab_id}")
                phase = snap.get("phase") or snap.get("collaboration", {}).get("phase")
                if phase == want:
                    break
                time.sleep(0.5)
            else:
                raise RuntimeError(f"wait_phase: timed out waiting for {want}")
            continue

        if action == "wait_task":
            needle = step.get("task_title_contains") or ""
            task_id = step.get("task_id") or ""
            want_status = step.get("status")
            want_awaiting = step.get("awaiting_approval")
            timeout = float(step.get("timeout_sec", 60))
            deadline = time.time() + timeout
            last = None
            while time.time() < deadline:
                snap = api(base, "GET", f"/api/runbooks/{collab_id}")
                tasks = collab_tasks(snap if isinstance(snap, dict) else {})
                t = find_task(tasks, needle, task_id)
                last = t
                if t is None:
                    time.sleep(0.5)
                    continue
                ok = True
                if want_status is not None and t.get("status") != want_status:
                    ok = False
                if want_awaiting is not None and bool(t.get("awaiting_approval")) != bool(want_awaiting):
                    ok = False
                if ok:
                    break
                time.sleep(0.5)
            else:
                raise RuntimeError(f"wait_task: timed out; last={last!r}")
            continue

        if action == "approve_task":
            snap = api(base, "GET", f"/api/runbooks/{collab_id}")
            tasks = collab_tasks(snap if isinstance(snap, dict) else {})
            t = find_task(tasks, step.get("task_title_contains") or "", step.get("task_id") or "")
            if t is None or not t.get("id"):
                raise RuntimeError("approve_task: task not found")
            api(base, "POST", f"/api/collaborations/{collab_id}/tasks/{t['id']}/approve", {})
            continue

        if action == "assert_task_status":
            snap = api(base, "GET", f"/api/runbooks/{collab_id}")
            tasks = collab_tasks(snap if isinstance(snap, dict) else {})
            needle = (step.get("task_title_contains") or "").lower()
            want = step.get("status")
            matched = [
                t for t in tasks
                if needle in (t.get("title") or "").lower() and t.get("status") == want
            ]
            if not matched:
                raise RuntimeError(f"assert_task_status: no task matching {needle!r} with status {want}")
            continue

        raise RuntimeError(f"unknown step action: {action}")

    # Free the concurrent-collab slot so scenarios can be chained.
    # Slash commands go through POST /api/send (not /api/messages, which is history GET).
    # Post on the collaboration's own channel (collab-<id>).
    if collab_id:
        try:
            snap = api(base, "GET", f"/api/runbooks/{collab_id}")
            cancel_ch = ""
            if isinstance(snap, dict):
                cancel_ch = (
                    snap.get("channel")
                    or (snap.get("collaboration") or {}).get("channel")
                    or ""
                )
            if not cancel_ch:
                cancel_ch = f"collab-{collab_id}"
            api(
                base,
                "POST",
                "/api/send",
                {
                    "channel": cancel_ch,
                    "content": f"/cancel-plan {collab_id[:8]}",
                    "type": "question",
                    "from": {"name": "runbook-scenario", "type": "human"},
                },
            )
            # Brief settle so the concurrent-collab counter drops before the next scenario.
            time.sleep(0.4)
        except Exception as e:
            if verbose:
                print(f"  cleanup cancel-plan note: {e}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Run runbook scenario JSON against live hub")
    parser.add_argument("--scenario", help="Scenario name (without .json)")
    parser.add_argument("--list", action="store_true", help="List available scenarios")
    parser.add_argument("--hub", default="http://127.0.0.1:18765", help="Hub base URL")
    parser.add_argument("--verbose", action="store_true")
    args = parser.parse_args()

    if args.list:
        for p in sorted(SCENARIOS_DIR.glob("*.json")):
            print(p.stem)
        return 0

    if not args.scenario:
        parser.error("--scenario required (or use --list)")

    path = SCENARIOS_DIR / f"{args.scenario}.json"
    if not path.is_file():
        print(f"Scenario not found: {path}", file=sys.stderr)
        return 1

    scenario = json.loads(path.read_text(encoding="utf-8"))
    base = scenario.get("hub_url") or args.hub
    print(f"Running runbook scenario {args.scenario} against {base}")
    run_scenario(base, scenario, verbose=args.verbose)
    print("OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
