#!/usr/bin/env python3
"""
Run personal learning scenarios against a live hub.

Examples:
  ./scripts/learning-scenarios.py --list
  ./scripts/learning-scenarios.py --scenario learning-save-and-list
  make learning-scenario SCENARIO=learning-save-and-list
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import time
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402

SCENARIOS_DIR = ROOT / "scenarios" / "learning"
DEFAULT_HUB = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")


def load_scenario(name: str) -> dict:
    path = SCENARIOS_DIR / f"{name}.json"
    if not path.is_file():
        raise SystemExit(f"scenario not found: {path}")
    return json.loads(path.read_text())


def list_scenarios() -> list[str]:
    return sorted(p.stem for p in SCENARIOS_DIR.glob("*.json"))


def merge_settings(base: str, patch: dict) -> None:
    code, cfg = hub.hub_request(base, "GET", "/api/settings")
    if code != 200 or not isinstance(cfg, dict):
        raise RuntimeError(f"GET /api/settings failed: {code} {cfg}")
    merged = {**cfg, **patch}
    if "features" in patch:
        merged["features"] = {**(cfg.get("features") or {}), **patch["features"]}
    code, out = hub.hub_request(base, "PUT", "/api/settings", merged)
    if code != 200:
        raise RuntimeError(f"PUT /api/settings failed: {code} {out}")


def enable_pack(base: str, pack_id: str) -> None:
    code, out = hub.hub_request(
        base,
        "PUT",
        f"/api/packs/{pack_id}",
        {"enabled": True},
    )
    if code != 200:
        raise RuntimeError(f"enable pack {pack_id}: {code} {out}")


def run_step(base: str, scenario: dict, step: dict, ctx: dict) -> None:
    stype = step.get("type")
    if stype == "setup_pack":
        enable_pack(base, step.get("pack_id", "specialist-tuning"))
        setup = scenario.get("setup") or {}
        if setup.get("enable_personal_learning"):
            merge_settings(base, {"features": {"personal_learning_enabled": True}})
        return

    if stype == "post_learning":
        body = {
            "agent_id": step["agent_id"],
            "agent_name": step.get("agent_name", ""),
            "agent_type": step.get("agent_type", ""),
            "content": step["content"],
            "category": step.get("category", "preference"),
        }
        code, out = hub.hub_request(base, "POST", "/api/learnings", body)
        if code != 200:
            raise RuntimeError(f"post_learning failed: {code} {out}")
        ctx.setdefault("learnings", []).append(out)
        return

    if stype == "assert_learnings":
        agent_id = step.get("agent_id", "")
        q = f"?agent_id={agent_id}" if agent_id else ""
        code, rows = hub.hub_request(base, "GET", f"/api/learnings{q}")
        if code != 200 or not isinstance(rows, list):
            raise RuntimeError(f"assert_learnings GET failed: {code} {rows}")
        min_count = int(step.get("min_count", 1))
        if len(rows) < min_count:
            raise RuntimeError(f"expected >= {min_count} learnings, got {len(rows)}")
        match = step.get("any_match")
        if match and not any(match in (r.get("content") or "") for r in rows):
            raise RuntimeError(f"no learning matched {match!r}")
        return

    if stype == "forget_learning":
        agent_id = step["agent_id"]
        code, rows = hub.hub_request(base, "GET", f"/api/learnings?agent_id={agent_id}")
        if code != 200 or not isinstance(rows, list):
            raise RuntimeError(f"forget_learning list failed: {code}")
        target = None
        content_match = step.get("content_match")
        for row in rows:
            if content_match and content_match in (row.get("content") or ""):
                target = row
                break
        if not target:
            raise RuntimeError("forget_learning: no matching row")
        lid = target.get("id")
        code, out = hub.hub_request(base, "DELETE", f"/api/learnings/{lid}")
        if code != 200:
            raise RuntimeError(f"DELETE learning failed: {code} {out}")
        return

    if stype == "send":
        channel = scenario.get("channel") or step.get("channel") or "general"
        content = step["content"]
        code, before = hub.hub_request(
            base,
            "GET",
            f"/api/messages?channel={channel}&limit=50",
        )
        baseline = len(before) if code == 200 and isinstance(before, list) else 0
        code, out = hub.hub_request(
            base,
            "POST",
            "/api/send",
            {
                "channel": channel,
                "content": content,
                "type": "chat",
                "from": {"name": "ScenarioUser", "type": "human"},
            },
        )
        if code != 200:
            raise RuntimeError(f"send failed: {code} {out}")
        ctx["last_channel"] = channel
        ctx["baseline_message_count"] = baseline
        ctx["send_started_at"] = time.time()
        return

    if stype == "wait_reply":
        channel = ctx.get("last_channel") or scenario.get("channel") or "general"
        timeout = float(step.get("timeout_sec", 20))
        baseline = int(ctx.get("baseline_message_count", 0))
        want_proposal = bool(step.get("learning_proposal"))
        deadline = time.time() + timeout
        while time.time() < deadline:
            code, msgs = hub.hub_request(
                base,
                "GET",
                f"/api/messages?channel={channel}&limit=50",
            )
            if code == 200 and isinstance(msgs, list):
                ctx["messages"] = msgs
                if want_proposal:
                    for msg in reversed(msgs):
                        action = (msg.get("metadata") or {}).get("client_action") or {}
                        if action.get("type") == "learning_proposal":
                            return
                elif len(msgs) > baseline:
                    return
            time.sleep(0.5)
        raise RuntimeError("wait_reply timed out")

    if stype == "assert_learning_proposal":
        msgs = ctx.get("messages") or []
        want_agent = step.get("agent_name")
        for msg in reversed(msgs):
            meta = msg.get("metadata") or {}
            action = meta.get("client_action") or {}
            if action.get("type") != "learning_proposal":
                continue
            if want_agent and action.get("agent_name") != want_agent:
                continue
            return
        raise RuntimeError("assert_learning_proposal: no proposal metadata on recent messages")

    if stype == "assert_expert_context":
        agent_id = step["agent_id"]
        code, body = hub.hub_request(
            base,
            "GET",
            f"/api/lora/train/expert-context?agent_id={agent_id}",
        )
        if code != 200 or not isinstance(body, dict):
            raise RuntimeError(f"expert-context failed: {code} {body}")
        if "ready" in step and bool(body.get("ready")) != bool(step["ready"]):
            raise RuntimeError(f"ready mismatch: {body.get('ready')} vs {step['ready']}")
        pattern = step.get("tag_pattern")
        if pattern:
            import re

            tag = body.get("suggested_ollama_tag") or ""
            if not re.search(pattern, tag):
                raise RuntimeError(f"tag {tag!r} did not match {pattern!r}")
        return

    raise RuntimeError(f"unknown step type: {stype}")


def run_scenario(base: str, name: str, verbose: bool = False) -> None:
    scenario = load_scenario(name)
    if not hub.check_health(base):
        raise SystemExit(f"hub not reachable at {base}")
    ctx: dict = {}
    if verbose:
        print(f"=== {name} ===")
    for i, step in enumerate(scenario.get("steps") or []):
        if verbose:
            print(f"  step {i + 1}: {step.get('type')}")
        run_step(base, scenario, step, ctx)
    if verbose:
        print(f"OK {name}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Personal learning scenario runner")
    parser.add_argument("--hub", default=DEFAULT_HUB)
    parser.add_argument("--list", action="store_true")
    parser.add_argument("--all", action="store_true")
    parser.add_argument("--scenario")
    parser.add_argument("--verbose", action="store_true")
    args = parser.parse_args()

    if args.list:
        for name in list_scenarios():
            print(name)
        return

    names = list_scenarios() if args.all else ([args.scenario] if args.scenario else [])
    if not names:
        parser.error("pass --scenario NAME or --all")

    failed = 0
    for name in names:
        try:
            run_scenario(args.hub, name, verbose=args.verbose)
            print(f"PASS {name}")
        except Exception as e:
            failed += 1
            print(f"FAIL {name}: {e}", file=sys.stderr)
    if failed:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
