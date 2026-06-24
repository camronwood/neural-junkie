#!/usr/bin/env python3
"""Run implementation session scenarios against a live hub."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import time
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.hub_regression import ensure_hub_with_recovery  # noqa: E402
from lib.release_prep_env import release_prep_env  # noqa: E402
from lib.scenario_assert import (  # noqa: E402
    check_file_deliverable,
    check_text_patterns,
    expand_deliverable_steps,
    looks_like_read_only_inspection_command,
    merge_deliverable_step,
    scenario_question,
)
from lib.fixture_baseline import reset_all_fixture_baselines, reset_fixture_baseline  # noqa: E402
from lib.workspace_context import enrich_send_metadata  # noqa: E402

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


class ImplementContext:
    def __init__(self, base: str, scenario: dict) -> None:
        self.base = base.rstrip("/")
        self.scenario = scenario
        self.channel = (scenario.get("channel") or DEFAULT_CHANNEL).strip()
        self.target_agent = (scenario.get("target_agent") or "BackendEngineer").strip().lstrip("@")
        self.baseline_agent_count: dict[str, int] = {}
        self.file_snapshots: dict[str, str] = {}


def _chat_baseline(ctx: ImplementContext, agent: str) -> int:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    return hub.count_chat_agent_messages(hub.chat_agent_messages(msgs), agent)


def _file_hash(root: Path, rel: str) -> str:
    path = (root / rel).resolve()
    if not path.is_file():
        return ""
    try:
        data = path.read_bytes()
    except OSError:
        return ""
    return hashlib.sha256(data).hexdigest()


def snapshot_files(ctx: ImplementContext, rel_paths: list[str]) -> None:
    root = Path(scenario_repo_root(ctx.scenario))
    for rel in rel_paths:
        rel = str(rel).strip()
        if rel:
            ctx.file_snapshots[rel] = _file_hash(root, rel)


def step_send(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    ctx.baseline_agent_count[from_name] = _chat_baseline(ctx, from_name)
    content = (step.get("content") or "").strip()
    meta = enrich_send_metadata(step.get("metadata"), ctx.scenario, content=content)
    code, _ = hub.send_message(ctx.base, ctx.channel, content, metadata=meta, from_name=DEFAULT_FROM)
    return (True, "sent") if code == 200 else (False, f"send failed ({code})")


def step_wait_reply(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    timeout = step.get("timeout", "420s")
    secs = 420
    if isinstance(timeout, str) and timeout.endswith("s"):
        try:
            secs = int(timeout[:-1])
        except ValueError:
            pass
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    until_any = step.get("until_any_match")
    baseline = int(step.get("baseline", ctx.baseline_agent_count.get(from_name, _chat_baseline(ctx, from_name))))
    deadline = time.time() + secs
    while time.time() < deadline:
        msgs = hub.list_messages(ctx.base, ctx.channel, 200)
        pool = hub.chat_agent_messages(msgs)
        candidates = [m for m in pool if m.get("from", {}).get("name") == from_name]
        for msg in candidates[baseline:]:
            text = msg.get("content") or ""
            if until_any:
                ok, detail = check_text_patterns(text, any_match=until_any)
                if ok:
                    return True, f"reply from {from_name} ({detail})"
            else:
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


def step_assert_deliverable(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    root = scenario_repo_root(ctx.scenario)
    spec = merge_deliverable_step(ctx.scenario, step)
    rel = (spec.get("path") or "").strip()
    return check_file_deliverable(
        root=root,
        rel=rel,
        spec=spec,
        question=scenario_question(ctx.scenario),
        hub_base=ctx.base,
    )


def step_assert_file_exists(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    return step_assert_deliverable(ctx, step)


def step_assert_file_absent(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    return step_assert_deliverable(ctx, {**step, "action": "assert_file_absent"})


def _last_agent_message(ctx: ImplementContext, from_name: str) -> dict | None:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    pool = hub.chat_agent_messages(msgs)
    candidates = [m for m in pool if m.get("from", {}).get("name") == from_name]
    return candidates[-1] if candidates else None


def step_assert_no_file_change(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    msg = _last_agent_message(ctx, from_name)
    if not msg:
        return False, "no agent reply"
    body = (msg.get("content") or "").lower()
    if "[file_change]" in body:
        return False, "file_change in reply"
    meta = msg.get("metadata") or {}
    if meta.get("implementation_files_changed"):
        return False, "files changed"
    outcome = meta.get("implementation_session_outcome")
    if isinstance(outcome, dict):
        if outcome.get("outcome") not in (None, "", "no_changes"):
            return False, f"implementation outcome {outcome.get('outcome')!r}"
        if outcome.get("files_changed"):
            return False, "outcome lists files_changed"
    return True, "no file changes"


def step_assert_suggested_commands(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    msg = _last_agent_message(ctx, from_name)
    if not msg:
        return False, f"no messages from {from_name}"
    meta = msg.get("metadata") or {}
    raw_cmds = meta.get("suggested_commands") or []
    if not isinstance(raw_cmds, list):
        raw_cmds = []
    want_safe = step.get("require_safe")
    want_unsafe = step.get("require_unsafe")
    cmd_pattern = (step.get("command_match") or "").strip()
    none_match = step.get("none_match") or []
    found = False
    for item in raw_cmds:
        if not isinstance(item, dict):
            continue
        cmd = (item.get("command") or "").strip()
        for pattern in none_match:
            if re.search(pattern, cmd, re.I):
                return False, f"forbidden command {cmd!r} matched {pattern!r}"
        if cmd_pattern and not re.search(cmd_pattern, cmd, re.I):
            continue
        found = True
        is_safe = bool(item.get("is_safe"))
        if want_safe is True and not is_safe:
            return False, f"expected safe command, got unsafe: {cmd!r}"
        if want_unsafe is True and is_safe:
            return False, f"expected unsafe command, got safe: {cmd!r}"
        if step.get("deny_readonly_unsafe"):
            if looks_like_read_only_inspection_command(cmd) and not is_safe:
                return False, f"read-only command marked unsafe: {cmd!r}"
    if not found:
        if step.get("optional"):
            return True, "skipped (no matching suggested_commands)"
        return False, f"no suggested_commands matched (pattern={cmd_pattern!r})"
    return True, "suggested command assertions ok"


def _metadata_get(meta: dict, dotted: str):
    cur: object = meta
    for part in dotted.split("."):
        if not isinstance(cur, dict):
            return None
        cur = cur.get(part)
    return cur


def step_assert_message_metadata(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    msg = _last_agent_message(ctx, from_name)
    if not msg:
        if step.get("optional"):
            return True, "skipped (no agent reply)"
        return False, f"no messages from {from_name}"
    meta = msg.get("metadata") or {}
    for key, expected in (step.get("equals") or {}).items():
        got = _metadata_get(meta, str(key))
        if got != expected:
            if step.get("optional"):
                return True, f"skipped optional metadata mismatch on {key!r}"
            return False, f"metadata {key!r}: got {got!r} want {expected!r}"
    for key, pattern in (step.get("match") or {}).items():
        got = _metadata_get(meta, str(key))
        text = "" if got is None else str(got)
        if not re.search(str(pattern), text, re.I):
            if step.get("optional"):
                return True, f"skipped optional metadata pattern on {key!r}"
            return False, f"metadata {key!r} did not match {pattern!r} (got {text!r})"
    for key in step.get("require_keys") or []:
        if _metadata_get(meta, str(key)) is None:
            if step.get("optional"):
                return True, f"skipped optional missing metadata key {key!r}"
            return False, f"missing metadata key {key!r}"
    return True, "metadata assertions ok"


def step_assert_files_unchanged(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    paths = step.get("paths") or []
    if not paths:
        return False, "assert_files_unchanged: paths required"
    root = Path(scenario_repo_root(ctx.scenario))
    for rel in paths:
        rel = str(rel).strip()
        if not rel:
            continue
        before = ctx.file_snapshots.get(rel)
        if before is None:
            before = _file_hash(root, rel)
            ctx.file_snapshots[rel] = before
        after = _file_hash(root, rel)
        if before != after:
            return False, f"file changed: {rel}"
    return True, f"{len(paths)} file(s) unchanged"


HANDLERS = {
    "send": step_send,
    "wait_reply": step_wait_reply,
    "assert_messages": step_assert_messages,
    "assert_deliverable": step_assert_deliverable,
    "assert_file_exists": step_assert_file_exists,
    "assert_file_absent": step_assert_file_absent,
    "assert_no_file_change": step_assert_no_file_change,
    "assert_suggested_commands": step_assert_suggested_commands,
    "assert_message_metadata": step_assert_message_metadata,
    "assert_files_unchanged": step_assert_files_unchanged,
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


def ensure_hub_ready(base: str, context: str) -> bool:
    if ensure_hub_with_recovery(ROOT, base, context=context, env=release_prep_env(ROOT)):
        return True
    print("  FAIL: hub unhealthy (recovery exhausted after 3 attempts)", file=sys.stderr)
    return False


def run_scenario(base: str, name: str, *, keep: bool = False) -> bool:
    scenario = load_scenario(name)
    ctx = ImplementContext(base, scenario)
    print(f"\n=== implement: {name} ===")
    if not ensure_hub_ready(base, f"implement:{name}"):
        return False
    required = scenario.get("required_agents") or [ctx.target_agent]
    ok, missing = hub.verify_agents_online(base, required)
    if not ok:
        print(f"  FAIL: offline agents: {missing}", file=sys.stderr)
        return False
    if not ensure_channel(ctx):
        return False
    if scenario.get("cleanup", "clear") == "clear" and not keep:
        hub.clear_channel_history(ctx.base, ctx.channel)
    reset_fixture_baseline(scenario, root=ROOT)
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    unchanged = ws_cfg.get("unchanged_files") or scenario.get("unchanged_files") or []
    if unchanged:
        snapshot_files(ctx, list(unchanged))
    # In-process agents discover new channels on a ~1s tick; allow subscribe before send.
    time.sleep(3.0)
    all_ok = True
    try:
        combined_steps = (
            list(scenario.get("setup") or [])
            + list(scenario.get("steps") or [])
            + expand_deliverable_steps(scenario)
        )
        for i, step in enumerate(combined_steps, 1):
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
    finally:
        reset_fixture_baseline(scenario, root=ROOT)
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
    if args.all:
        reset_all_fixture_baselines(root=ROOT)
    failed = sum(1 for n in names if not run_scenario(args.hub, n, keep=args.keep))
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
