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
import re
import sys
import time
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.scenario_assert import check_text_patterns, looks_like_read_only_inspection_command  # noqa: E402

SCENARIOS_DIR = ROOT / "scenarios" / "chat"
DEFAULT_CHANNEL = "chat-scenarios"
DEFAULT_AGENT = "Assistant"
DEFAULT_FROM = "ChatScenario"


def _language_for_path(path: str) -> str:
    ext = Path(path).suffix.lower()
    return {
        ".go": "go",
        ".ts": "typescript",
        ".tsx": "typescriptreact",
        ".js": "javascript",
        ".jsx": "javascriptreact",
        ".py": "python",
        ".md": "markdown",
    }.get(ext, "text")


def _load_open_files(workspace_root: Path, rel_paths: list, *, active: str | None = None) -> list[dict]:
    active_rel = (active or (rel_paths[0] if rel_paths else "")).strip()
    out: list[dict] = []
    for rel in rel_paths:
        rel = str(rel).strip()
        if not rel:
            continue
        full = (workspace_root / rel).resolve()
        content = ""
        if full.is_file():
            try:
                content = full.read_text(encoding="utf-8", errors="replace")[:10000]
            except OSError:
                content = ""
        out.append(
            {
                "path": str(full),
                "language": _language_for_path(rel),
                "content": content,
                "is_active": rel == active_rel or (not active_rel and len(out) == 0),
            }
        )
    return out


def enrich_send_metadata(meta: dict | None, scenario: dict) -> dict | None:
    """Mirror desktop sends: context_scope without workspace_context cannot trigger visibility replies."""
    if not meta:
        return None
    out = dict(meta)
    scope = str(out.get("context_scope") or "").strip().lower()
    if scope in ("", "none") or out.get("workspace_context"):
        return out
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    root = os.environ.get("NEURAL_JUNKIE_SCENARIO_REPO", str(ROOT)).strip()
    fixture = (ws_cfg.get("fixture") or scenario.get("workspace_fixture") or "").strip()
    if fixture:
        root = str((ROOT / "scenarios" / "fixtures" / fixture).resolve())
    open_rel = ws_cfg.get("open_files") or []
    open_files = (
        _load_open_files(Path(root), open_rel, active=(open_rel[0] if open_rel else None))
        if open_rel
        else []
    )
    out["workspace_context"] = {
        "workspace_name": Path(root).name,
        "workspace_path": root,
        "file_tree": ws_cfg.get("file_tree") or "desktop/\ninternal/\ncmd/\n",
        "open_files": open_files,
    }
    return out


class ChatScenarioContext:
    def __init__(
        self,
        base: str,
        scenario: dict,
        verbose: bool = False,
        *,
        require_debug: bool = False,
    ) -> None:
        self.base = base.rstrip("/")
        self.scenario = scenario
        self.verbose = verbose
        self.require_debug = require_debug
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
        data = json.load(f)
    if isinstance(data, dict) and not data.get("name"):
        data["name"] = name
    return data


def scenario_tags(scenario: dict) -> list[str]:
    raw = scenario.get("tags")
    if not isinstance(raw, list):
        return []
    return [str(t).strip().lower() for t in raw if str(t).strip()]


def list_scenarios(tags: list[str] | None = None) -> list[str]:
    if not SCENARIOS_DIR.is_dir():
        return []
    names = sorted(p.stem for p in SCENARIOS_DIR.glob("*.json"))
    if not tags:
        return names
    want = {t.strip().lower() for t in tags if t.strip()}
    out = []
    for name in names:
        try:
            sc = load_scenario(name)
        except (OSError, json.JSONDecodeError):
            continue
        have = set(scenario_tags(sc))
        if want <= have:
            out.append(name)
    return out


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
    meta = enrich_send_metadata(step.get("metadata"), ctx.scenario)
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


def _last_agent_chat_message(ctx: ChatScenarioContext, from_agent: str) -> dict | None:
    want = from_agent.strip().lstrip("@")
    msgs = hub.chat_agent_messages(hub.list_messages(ctx.base, ctx.channel, 200))
    agent_msgs = [m for m in msgs if (m.get("from") or {}).get("name") == want]
    return agent_msgs[-1] if agent_msgs else None


def _is_generation_error_reply(msg: dict | None) -> bool:
    if not msg:
        return False
    body = (msg.get("content") or "").lower()
    if "encountered an error while generating" in body or "provider_error" in body:
        return True
    if "workspace root not set" in body:
        return True
    if "implementation session finished" in body and "error" in body:
        return True
    return False


def step_wait_reply(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    from_agent = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    timeout = hub.parse_timeout(step.get("timeout", 90))
    baseline = int(step.get("baseline", ctx.baseline_agent_count))
    max_new = int(step.get("max_new", 1))
    retries = max(0, int(step.get("retries", 0)))
    resend_on_error = bool(step.get("resend_on_generation_error"))

    for attempt in range(retries + 1):
        ok, detail = hub.wait_chat_reply(
            ctx.base,
            ctx.channel,
            from_agent=from_agent,
            baseline_count=baseline,
            timeout=timeout,
            max_new=max_new,
            detect_failures=True,
        )
        if not ok:
            if attempt < retries and (
                "failure" in detail.lower() or "timeout" in detail.lower()
            ):
                if resend_on_error and step.get("resend_content"):
                    ctx.log(f"  wait_reply: {detail}; re-sending user message")
                    hub.send_message(
                        ctx.base,
                        ctx.channel,
                        str(step.get("resend_content")),
                        metadata=enrich_send_metadata(step.get("resend_metadata"), ctx.scenario),
                        from_name=(step.get("resend_from") or DEFAULT_FROM).strip(),
                        max_retries=5,
                    )
                    baseline = hub.count_chat_agent_messages(
                        hub.list_messages(ctx.base, ctx.channel, 200),
                        from_agent,
                    )
                    time.sleep(3)
                    continue
                ctx.log(f"  wait_reply attempt {attempt + 1}: {detail}; retrying")
                time.sleep(3)
                continue
            return ok, detail
        last = _last_agent_chat_message(ctx, from_agent)
        if _is_generation_error_reply(last):
            if attempt < retries and resend_on_error and step.get("resend_content"):
                ctx.log("  wait_reply got generation_error; re-sending user message")
                hub.send_message(
                    ctx.base,
                    ctx.channel,
                    str(step.get("resend_content")),
                    metadata=enrich_send_metadata(step.get("resend_metadata"), ctx.scenario),
                    from_name=(step.get("resend_from") or DEFAULT_FROM).strip(),
                    max_retries=5,
                )
                baseline = hub.count_chat_agent_messages(
                    hub.list_messages(ctx.base, ctx.channel, 200),
                    from_agent,
                )
                time.sleep(2)
                continue
            if attempt < retries:
                ctx.log("  wait_reply got generation_error; retrying poll")
                time.sleep(3)
                continue
            return False, "agent returned generation_error reply"
        suffix = f" (after retry {attempt})" if attempt else ""
        return True, detail + suffix
    return False, "wait_reply exhausted retries"


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
    baseline = int(step.get("baseline", ctx.baseline_agent_count if step.get("since_baseline") else ctx.start_agent_count))
    since = total - baseline
    want = int(step.get("count", step.get("since_start", 1)))
    label = "since baseline" if step.get("since_baseline") else "since start"
    if since != want:
        return False, f"reply count {label}: got {since} want {want} (total={total})"
    return True, f"reply count {label}={since}"


def step_channel_interject(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    held_by = (step.get("held_by") or DEFAULT_FROM).strip()
    retries = max(0, int(step.get("retries", 0)))
    for attempt in range(retries + 1):
        ok, detail = hub.channel_interject(ctx.base, ctx.channel, held_by=held_by)
        if ok:
            ctx.baseline_agent_count = hub.count_chat_agent_messages(
                hub.list_messages(ctx.base, ctx.channel, 200),
                ctx.target_agent,
            )
            suffix = f" (retry {attempt})" if attempt else ""
            return True, detail + suffix
        if attempt < retries:
            ctx.log(f"  interject attempt {attempt + 1} failed; retrying: {detail}")
            time.sleep(2)
    return False, detail


def step_wait_no_reply(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    from_agent = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    duration = hub.parse_timeout(step.get("duration", step.get("timeout", 8)))
    baseline = int(step.get("baseline", ctx.baseline_agent_count))
    retries = max(0, int(step.get("retries", 0)))
    reinterject = bool(step.get("reinterject_on_retry"))

    for attempt in range(retries + 1):
        ok, detail = hub.wait_no_new_chat_replies(
            ctx.base,
            ctx.channel,
            from_agent=from_agent,
            baseline_count=baseline,
            duration=duration,
        )
        if ok:
            suffix = f" (after retry {attempt})" if attempt else ""
            return True, detail + suffix
        if attempt < retries:
            ctx.log(f"  wait_no_reply attempt {attempt + 1} failed; retrying\n  {detail}")
            if reinterject:
                hub.channel_interject(ctx.base, ctx.channel, held_by=DEFAULT_FROM)
            time.sleep(2)
            continue
    return False, detail


def step_assert_suggested_commands(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    pool = hub.chat_agent_messages(msgs)
    from_agent = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    if step.get("last_reply_only"):
        agent_msgs = [m for m in pool if (m.get("from") or {}).get("name") == from_agent]
        if not agent_msgs:
            return False, f"no messages from {from_agent}"
        pool = [agent_msgs[-1]]

    want_safe = step.get("require_safe")
    want_unsafe = step.get("require_unsafe")
    cmd_pattern = (step.get("command_match") or "").strip()
    found = False
    for m in pool:
        meta = m.get("metadata") or {}
        raw_cmds = meta.get("suggested_commands") or []
        if not isinstance(raw_cmds, list):
            continue
        for item in raw_cmds:
            if not isinstance(item, dict):
                continue
            cmd = (item.get("command") or "").strip()
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


def step_assert_debug_context(ctx: ChatScenarioContext, step: dict) -> tuple[bool, str]:
    sample = (step.get("message") or "").strip()
    if not sample:
        return False, "assert_debug_context: message required"
    mode = (step.get("conversation_mode") or "").strip() or None
    scope_q = (step.get("context_scope") or (step.get("metadata") or {}).get("context_scope") or "")
    scope_q = str(scope_q).strip() or None
    data = hub.fetch_debug_context(
        ctx.base, ctx.channel, sample, conversation_mode=mode, context_scope=scope_q
    )
    if not data:
        if ctx.require_debug:
            return False, "debug API unavailable (hub needs NEURAL_JUNKIE_DEBUG=1; use make server-regression)"
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
    if want := step.get("context_scope"):
        got = (data.get("context_scope") or "").strip()
        if got != str(want):
            return False, f"context_scope: got {got!r} want {want!r}"
    return True, "debug context ok"


def run_step(ctx: ChatScenarioContext, step: dict, label: str) -> bool:
    action = (step.get("action") or step.get("type") or "").strip()
    handlers = {
        "send": step_send,
        "wait_reply": step_wait_reply,
        "channel_interject": step_channel_interject,
        "wait_no_reply": step_wait_no_reply,
        "assert_messages": step_assert_messages,
        "assert_reply_count": step_assert_reply_count,
        "assert_suggested_commands": step_assert_suggested_commands,
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
    require_debug: bool = False,
) -> bool:
    scenario = load_scenario(name)
    ctx = ChatScenarioContext(base, scenario, verbose, require_debug=require_debug)

    print(f"\n=== scenario: {name} ===")
    print(f"  hub={base}")

    health = hub.check_health(base)
    if not health:
        print("  FAIL: hub not healthy", file=sys.stderr)
        return False

    required = scenario.get("required_agents") or [ctx.target_agent]
    ok_agents, missing = hub.verify_agents_online(base, required)
    if not ok_agents:
        if scenario.get("optional"):
            print(f"=== SKIP (optional): {name} — offline: {', '.join(missing)} ===\n")
            return True
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
    p.add_argument(
        "--tag",
        action="append",
        default=[],
        help="Filter scenarios (repeatable). Scenario must include all listed tags.",
    )
    p.add_argument(
        "--require-debug",
        action="store_true",
        help="Fail assert_debug_context when hub debug API is off (even if step is optional)",
    )
    args = p.parse_args()

    if args.list:
        for n in list_scenarios(args.tag or None):
            sc = load_scenario(n)
            tags = ",".join(scenario_tags(sc)) or "-"
            opt = " (optional)" if sc.get("optional") else ""
            print(f"{n}\t{tags}{opt}")
        return 0

    base = args.hub.rstrip("/")
    if args.all:
        names = list_scenarios(args.tag or None)
        if not names:
            print("No scenarios found", file=sys.stderr)
            return 1
        failed = []
        skipped = []
        for i, n in enumerate(names):
            if i > 0:
                time.sleep(3.0)
            scenario = load_scenario(n)
            required = scenario.get("required_agents") or [scenario.get("target_agent") or DEFAULT_AGENT]
            ok_agents, missing = hub.verify_agents_online(base, required)
            if not ok_agents and scenario.get("optional"):
                print(f"=== SKIP (optional): {n} — offline: {', '.join(missing)} ===\n")
                skipped.append(n)
                continue
            if not run_scenario(
                base,
                n,
                verbose=args.verbose,
                keep=args.keep,
                require_debug=args.require_debug,
            ):
                failed.append(n)
        if skipped:
            print(f"Skipped optional: {', '.join(skipped)}", file=sys.stderr)
        return 1 if failed else 0

    if not args.scenario:
        p.error("specify --scenario <name> or --all")
    ok = run_scenario(
        base,
        args.scenario,
        verbose=args.verbose,
        keep=args.keep,
        require_debug=args.require_debug,
    )
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
