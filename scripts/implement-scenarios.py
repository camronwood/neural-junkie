#!/usr/bin/env python3
"""Run implementation session scenarios against a live hub."""
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
from lib.scenario_assert import check_text_patterns  # noqa: E402

SCENARIOS_DIR = ROOT / "scenarios" / "implement"
DEFAULT_CHANNEL = "implement-scenarios"
DEFAULT_FROM = "ImplementScenario"


def load_scenario(name: str) -> dict:
    path = SCENARIOS_DIR / f"{name}.json"
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def scenario_repo_root(scenario: dict) -> str:
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    root = os.environ.get("NEURAL_JUNKIE_SCENARIO_REPO", str(ROOT)).strip()
    fixture = (ws_cfg.get("fixture") or "").strip()
    if fixture:
        root = str((ROOT / "scenarios" / "fixtures" / fixture).resolve())
    return root


def enrich_send_metadata(meta: dict | None, scenario: dict) -> dict | None:
    if not meta:
        return None
    out = dict(meta)
    if out.get("workspace_context"):
        return out
    root = scenario_repo_root(scenario)
    out["workspace_context"] = {
        "workspace_name": Path(root).name,
        "workspace_path": root,
        "file_tree": "",
        "open_files": [],
    }
    return out


class ImplementContext:
    def __init__(self, base: str, scenario: dict) -> None:
        self.base = base.rstrip("/")
        self.scenario = scenario
        self.channel = (scenario.get("channel") or DEFAULT_CHANNEL).strip()
        self.target_agent = (scenario.get("target_agent") or "BackendEngineer").strip().lstrip("@")


def step_send(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    content = (step.get("content") or "").strip()
    meta = enrich_send_metadata(step.get("metadata"), ctx.scenario)
    code, _ = hub.send_message(ctx.base, ctx.channel, content, metadata=meta, from_name=DEFAULT_FROM)
    return (True, "sent") if code == 200 else (False, f"send failed ({code})")


def step_wait_reply(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    timeout = step.get("timeout", "180s")
    secs = 180
    if isinstance(timeout, str) and timeout.endswith("s"):
        try:
            secs = int(timeout[:-1])
        except ValueError:
            pass
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    deadline = time.time() + secs
    while time.time() < deadline:
        msgs = hub.list_messages(ctx.base, ctx.channel, 30)
        for m in reversed(msgs):
            if m.get("from", {}).get("name") == from_name:
                return True, f"reply from {from_name}"
        time.sleep(2)
    return False, f"timeout waiting for {from_name}"


def step_assert_messages(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    msgs = hub.list_messages(ctx.base, ctx.channel, 40)
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    candidates = [m for m in msgs if m.get("from", {}).get("name") == from_name]
    if not candidates:
        return False, f"no messages from {from_name}"
    text = candidates[-1].get("content") or ""
    if step.get("last_reply_only"):
        pass
    if any_match := step.get("any_match"):
        ok, detail = check_text_patterns(text, any_match=any_match)
        if not ok:
            return False, detail
    if none_match := step.get("none_match"):
        ok, detail = check_text_patterns(text, none_match=none_match)
        if not ok:
            return False, detail
    return True, "message assertions ok"


def step_assert_file_exists(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    root = scenario_repo_root(ctx.scenario)
    rel = (step.get("path") or "").strip()
    if not rel:
        return False, "path required"
    full = Path(root) / rel
    if not full.is_file():
        return False, f"missing {full}"
    if want := step.get("contains"):
        if want not in full.read_text(encoding="utf-8", errors="replace"):
            return False, f"{rel} missing {want!r}"
    return True, rel


def step_assert_no_file_change(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    msgs = hub.list_messages(ctx.base, ctx.channel, 50)
    for m in reversed(msgs):
        if m.get("from", {}).get("name") != ctx.target_agent:
            continue
        body = (m.get("content") or "").lower()
        if "[file_change]" in body:
            return False, "file_change in reply"
        meta = m.get("metadata") or {}
        if meta.get("implementation_files_changed"):
            return False, "files changed"
        return True, "no file changes"
    return False, "no agent reply"


HANDLERS = {
    "send": step_send,
    "wait_reply": step_wait_reply,
    "assert_messages": step_assert_messages,
    "assert_file_exists": step_assert_file_exists,
    "assert_no_file_change": step_assert_no_file_change,
}


def ensure_channel(ctx: ImplementContext) -> bool:
    ch_type = (ctx.scenario.get("channel_type") or "public").strip().lower()
    if ch_type == "dm":
        user = (ctx.scenario.get("dm_user") or DEFAULT_FROM).strip()
        name = hub.ensure_dm_channel(ctx.base, user, ctx.target_agent)
        if not name:
            return False
        ctx.channel = name
        return True
    required = ctx.scenario.get("required_agents") or [ctx.target_agent]
    ok, _ = hub.ensure_channel_with_agents(
        ctx.base, ctx.channel, required, ctx.scenario.get("description") or "implement scenarios"
    )
    return ok


def run_scenario(base: str, name: str, *, keep: bool = False) -> bool:
    scenario = load_scenario(name)
    ctx = ImplementContext(base, scenario)
    print(f"\n=== implement: {name} ===")
    if not hub.check_health(base):
        print("  FAIL: hub unhealthy", file=sys.stderr)
        return False
    required = scenario.get("required_agents") or [ctx.target_agent]
    ok, missing = hub.verify_agents_online(base, required)
    if not ok:
        print(f"  FAIL: offline agents: {missing}", file=sys.stderr)
        return False
    if not ensure_channel(ctx):
        return False
    all_ok = True
    for i, step in enumerate(scenario.get("steps") or [], 1):
        action = (step.get("action") or "").strip()
        fn = HANDLERS.get(action)
        if not fn:
            print(f"  FAIL unknown action {action}", file=sys.stderr)
            all_ok = False
            break
        ok, detail = fn(ctx, step)
        print(f"  {'✓' if ok else '✗'} [{i}] {action}: {detail}")
        if not ok:
            all_ok = False
            break
    if scenario.get("cleanup", "clear") == "clear" and not keep:
        hub.clear_channel_history(ctx.base, ctx.channel)
    print(f"=== {'PASS' if all_ok else 'FAIL'}: {name} ===\n")
    return all_ok


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--list", action="store_true")
    p.add_argument("--scenario")
    p.add_argument("--all", action="store_true")
    p.add_argument("--hub", default=hub.DEFAULT_HUB)
    p.add_argument("--keep", action="store_true")
    args = p.parse_args()
    if args.list:
        for f in sorted(SCENARIOS_DIR.glob("*.json")):
            print(f.stem)
        return 0
    names = [f.stem for f in sorted(SCENARIOS_DIR.glob("*.json"))] if args.all else ([args.scenario] if args.scenario else [])
    if not names:
        p.print_help()
        return 1
    failed = sum(1 for n in names if not run_scenario(args.hub, n, keep=args.keep))
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
