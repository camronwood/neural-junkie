#!/usr/bin/env python3
"""
Run general chat quality scenarios against a live hub with real agents.

Examples:
  ./scripts/chat-scenarios.py --list
  ./scripts/chat-scenarios.py --scenario greeting-chat-mode
  ./scripts/chat-scenarios.py --all
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
from lib.scenario_assert import check_text_patterns  # noqa: E402

SCENARIOS_DIR = ROOT / "scenarios" / "chat"
DEFAULT_CHANNEL = "chat-scenarios"
DEFAULT_AGENT = "Assistant"
DEFAULT_FROM = "ChatScenario"


class ChatScenarioContext:
    def __init__(self, base: str, scenario: dict, verbose: bool = False) -> None:
        self.base = base.rstrip("/")
        self.scenario = scenario
        self.verbose = verbose
        self.channel = (scenario.get("channel") or DEFAULT_CHANNEL).strip()
        self.target_agent = (scenario.get("target_agent") or DEFAULT_AGENT).strip().lstrip("@")
        self.mention = (scenario.get("mention") or "").strip()
        self.baseline_agent_count = 0
        self.start_agent_count = 0

    def log(self, msg: str) -> None:
        if self.verbose:
            print(msg)

    def format_send_content(self, content: str) -> str:
        content = content.strip()
        if self.mention and self.mention not in content:
            return f"{self.mention} {content}"
        return content


def load_scenario(name: str) -> dict:
    path = SCENARIOS_DIR / f"{name}.json"
    if not path.is_file():
        raise FileNotFoundError(f"scenario not found: {path}")
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def list_scenarios() -> list[str]:
    if not SCENARIOS_DIR.is_dir():
        return []
    return sorted(p.stem for p in SCENARIOS_DIR.glob("*.json"))


def dump_transcript(ctx: ChatScenarioContext, tail: int = 12) -> None:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    print("  --- transcript (last messages) ---", file=sys.stderr)
    for m in msgs[-tail:]:
        who = (m.get("from") or {}).get("name", "?")
        typ = m.get("type", "?")
        body = (m.get("content") or "").replace("\n", " ")[:200]
        print(f"    [{typ}] {who}: {body}", file=sys.stderr)


def step_send(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    content = ctx.format_send_content(step.get("content") or "")
    if not content:
        return False, "send: empty content"
    meta = step.get("metadata")
    from_name = (step.get("from") or DEFAULT_FROM).strip()
    # Baseline before send so instant replies (e.g. closure) still satisfy wait_reply.
    ctx.baseline_agent_count = hub.count_chat_agent_messages(
        hub.list_messages(ctx.base, ctx.channel, 200),
        ctx.target_agent,
    )
    code, _data = hub.send_message(
        ctx.base, ctx.channel, content, metadata=meta, from_name=from_name, max_retries=5
    )
    if code != 200:
        return False, f"send failed HTTP {code}"
    return True, content[:80] + ("…" if len(content) > 80 else "")


def step_wait_reply(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    from_agent = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    timeout = hub.parse_timeout(step.get("timeout", 90))
    baseline = int(step.get("baseline", ctx.baseline_agent_count))
    max_new = int(step.get("max_new", 1))
    return hub.wait_chat_reply(
        ctx.base,
        ctx.channel,
        from_agent=from_agent,
        baseline_count=baseline,
        timeout=timeout,
        max_new=max_new,
    )


def step_assert_messages(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    pool = hub.chat_agent_messages(msgs)
    if step.get("last_reply_only"):
        from_agent = (step.get("from") or ctx.target_agent).strip().lstrip("@")
        agent_msgs = [m for m in pool if (m.get("from") or {}).get("name") == from_agent]
        if not agent_msgs:
            return False, f"no messages from {from_agent}"
        pool = [agent_msgs[-1]]

    for pattern in step.get("none_match") or []:
        hits = hub.messages_matching(pool, pattern)
        if hits:
            who = (hits[0].get("from") or {}).get("name")
            return False, f"none_match violated {pattern!r} ({who})"

    for pattern in step.get("any_match") or []:
        if not hub.messages_matching(pool, pattern):
            agents = sorted({(m.get("from") or {}).get("name", "?") for m in pool})
            return False, f"any_match not found: {pattern!r} (agents: {agents or 'none'})"

    if step.get("max_chars"):
        limit = int(step["max_chars"])
        for m in pool[-1:]:
            body = m.get("content") or ""
            if len(body) > limit:
                return False, f"reply exceeds max_chars {limit} (got {len(body)})"

    return True, "message assertions ok"


def step_assert_reply_count(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    from_agent = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    total = hub.count_chat_agent_messages(msgs, from_agent)
    since = total - ctx.start_agent_count
    want = int(step.get("count", step.get("since_start", 1)))
    if since != want:
        return False, f"reply count since start: got {since} want {want} (total={total})"
    return True, f"reply count since start={since}"


def step_assert_debug_context(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    sample = (step.get("message") or "").strip()
    if not sample:
        return False, "assert_debug_context: message required"
    mode = (step.get("conversation_mode") or "").strip() or None
    data = hub.fetch_debug_context(ctx.base, ctx.channel, sample, conversation_mode=mode)
    if not data:
        if step.get("optional"):
            return True, "skipped (NEURAL_JUNKIE_DEBUG=1 not set on hub)"
        return False, "debug API unavailable (set NEURAL_JUNKIE_DEBUG=1 on hub)"
    if want := step.get("resolved_intent"):
        got = (data.get("resolved_intent") or "").strip()
        if got != want:
            return False, f"resolved_intent: got {got!r} want {want!r}"
    if want := step.get("conversation_mode"):
        got = (data.get("conversation_mode") or "").strip()
        if got != want:
            return False, f"conversation_mode: got {got!r} want {want!r}"
    return True, "debug context ok"


def run_step(ctx: ChatScenarioContext, step: dict, label: str) -> bool:
    action = (step.get("action") or step.get("type") or "").strip()
    handlers = {
        "send": step_send,
        "wait_reply": step_wait_reply,
        "assert_messages": step_assert_messages,
        "assert_reply_count": step_assert_reply_count,
        "assert_debug_context": step_assert_debug_context,
    }
    fn = handlers.get(action)
    if not fn:
        print(f"  FAIL [{label}]: unknown action {action!r}", file=sys.stderr)
        return False
    ok, detail = fn(ctx, step)
    mark = "✓" if ok else "✗"
    print(f"  {mark} [{label}] {action}: {detail}")
    if not ok:
        dump_transcript(ctx)
    return ok


def ensure_scenario_channel(ctx: ChatScenarioContext) -> bool:
    ch_type = (ctx.scenario.get("channel_type") or "public").strip().lower()
    if ch_type == "dm":
        user = (ctx.scenario.get("dm_user") or DEFAULT_FROM).strip()
        agent = ctx.target_agent
        name = hub.ensure_dm_channel(ctx.base, user, agent)
        if not name:
            print(f"  FAIL: could not create DM for {user} + {agent}", file=sys.stderr)
            return False
        ctx.channel = name
        return True
    desc = ctx.scenario.get("description") or "General chat quality scenarios"
    required = ctx.scenario.get("required_agents") or [ctx.target_agent]
    ok, failed = hub.ensure_channel_with_agents(ctx.base, ctx.channel, required, desc)
    if not ok:
        if failed:
            print(f"  FAIL: could not join agents to {ctx.channel!r}: {', '.join(failed)}", file=sys.stderr)
        else:
            print(f"  FAIL: could not ensure channel {ctx.channel!r}", file=sys.stderr)
        return False
    return True


def run_scenario(
    base: str,
    name: str,
    *,
    verbose: bool = False,
    keep: bool = False,
) -> bool:
    scenario = load_scenario(name)
    ctx = ChatScenarioContext(base, scenario, verbose)

    print(f"\n=== scenario: {name} ===")
    print(f"  hub={base}")

    health = hub.check_health(base)
    if not health:
        print("  FAIL: hub not healthy", file=sys.stderr)
        return False

    required = scenario.get("required_agents") or [ctx.target_agent]
    ok_agents, missing = hub.verify_agents_online(base, required)
    if not ok_agents:
        print(f"  FAIL: required agents offline: {', '.join(missing)}", file=sys.stderr)
        return False

    if not ensure_scenario_channel(ctx):
        return False
    print(f"  channel={ctx.channel} agent={ctx.target_agent}")

    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    ctx.start_agent_count = hub.count_chat_agent_messages(msgs, ctx.target_agent)

    all_ok = True
    steps = scenario.get("setup") or []
    steps = list(steps) + list(scenario.get("steps") or scenario.get("turns") or [])
    for i, step in enumerate(steps, 1):
        if not run_step(ctx, step, f"{i}"):
            all_ok = False
            break

    cleanup = scenario.get("cleanup", "clear")
    if cleanup == "clear" and not keep:
        if hub.clear_channel_history(ctx.base, ctx.channel):
            print("  ✓ cleanup: cleared channel history")
        else:
            print("  ⚠ cleanup: clear-history failed (local hub only)", file=sys.stderr)
    elif keep:
        print("  --keep: channel history preserved")

    if all_ok:
        print(f"=== PASS: {name} ===\n")
    else:
        print(f"=== FAIL: {name} ===\n", file=sys.stderr)
    return all_ok


def main() -> int:
    p = argparse.ArgumentParser(description="Run general chat scenarios (live hub)")
    p.add_argument("--list", action="store_true", help="List scenario names")
    p.add_argument("--scenario", help="Scenario name (without .json)")
    p.add_argument("--all", action="store_true", help="Run all scenarios")
    p.add_argument("--hub", default=hub.DEFAULT_HUB)
    p.add_argument("--verbose", "-v", action="store_true")
    p.add_argument("--keep", action="store_true", help="Do not clear channel history after run")
    args = p.parse_args()

    if args.list:
        for n in list_scenarios():
            print(n)
        return 0

    base = args.hub.rstrip("/")
    if args.all:
        names = list_scenarios()
        if not names:
            print("No scenarios found", file=sys.stderr)
            return 1
        failed = []
        for i, n in enumerate(names):
            if i > 0:
                time.sleep(3.0)
            if not run_scenario(base, n, verbose=args.verbose, keep=args.keep):
                failed.append(n)
        return 1 if failed else 0

    if not args.scenario:
        p.error("specify --scenario <name> or --all")
    ok = run_scenario(base, args.scenario, verbose=args.verbose, keep=args.keep)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
