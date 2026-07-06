#!/usr/bin/env python3
"""
Run runbook scenarios against a live hub (deterministic action steps, no LLM).

Examples:
  ./scripts/runbook-scenarios.py --list
  ./scripts/runbook-scenarios.py --scenario health-check-branch
  make runbook-scenario SCENARIO=health-check-branch
"""
from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SCENARIOS_DIR = ROOT / "scenarios" / "runbook"


def api(base: str, method: str, path: str, body: dict | None = None) -> dict:
    url = base.rstrip("/") + path
    data = None
    headers = {"Content-Type": "application/json"}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw.strip() else {}
    except urllib.error.HTTPError as e:
        detail = e.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} -> {e.code}: {detail}") from e


def resolve_agents(base: str, names: list[str]) -> list[str]:
    agents = api(base, "GET", "/api/agents").get("agents") or []
    by_name = {a.get("name", "").lower(): a.get("id") for a in agents}
    ids: list[str] = []
    for n in names:
        key = n.lstrip("@").lower()
        if key in by_name and by_name[key]:
            ids.append(by_name[key])
    if not ids:
        raise RuntimeError(f"Could not resolve agents {names}")
    return ids


def run_scenario(base: str, scenario: dict, verbose: bool = False) -> None:
    channel = scenario.get("channel") or "runbook-scenarios"
    collab_id = ""
    saved_definition_id = ""
    agent_ids = resolve_agents(base, scenario.get("agents") or ["@Assistant"])

    for step in scenario.get("steps") or []:
        action = step.get("action")
        if verbose:
            print(f"  step: {action}")

        if action == "save_definition":
            fixture = step.get("fixture") or step.get("path")
            if not fixture:
                raise RuntimeError("save_definition: fixture path required")
            fixture_path = ROOT / fixture if not Path(fixture).is_absolute() else Path(fixture)
            def_body = json.loads(fixture_path.read_text(encoding="utf-8"))
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

        if action == "assert_task_status":
            snap = api(base, "GET", f"/api/runbooks/{collab_id}")
            tasks = snap.get("tasks") or snap.get("collaboration", {}).get("tasks") or []
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
