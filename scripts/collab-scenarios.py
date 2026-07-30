#!/usr/bin/env python3
"""
Run collaboration scenarios against a live hub with real in-process agents.

Examples:
  ./scripts/collab-scenarios.py --list
  ./scripts/collab-scenarios.py --scenario planning-two-agent
  ./scripts/collab-scenarios.py --all
  NJ_SCENARIO_PROFILE=realistic ./scripts/collab-scenarios.py --scenario planning-two-agent
"""
from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.hub_config import env_or_automation_bool  # noqa: E402
from lib.hub_regression import ensure_hub_with_recovery  # noqa: E402
from lib.regression_boot import maybe_boot_regression  # noqa: E402
from lib.release_prep_env import release_prep_env  # noqa: E402
from lib.scenario_assert import (  # noqa: E402
    check_file_deliverable,
    check_text_patterns,
    expand_deliverable_steps,
    looks_like_stack_tool_command,
    merge_deliverable_step,
    scenario_question,
)
from lib.workspace_context import build_workspace_context_for_path  # noqa: E402

SCENARIOS_DIR = ROOT / "scenarios" / "collab"
DEFAULT_CHANNEL = "collab-scenarios"
BLOCKER_CHANNEL = "collab-scenario-blocker"


class ScenarioContext:
    def __init__(self, base: str, scenario: dict, verbose: bool = False) -> None:
        self.base = base.rstrip("/")
        self.scenario = scenario
        self.verbose = verbose
        self.channel = scenario.get("channel") or DEFAULT_CHANNEL
        self.collab_id = ""
        self.collab_channel = ""
        self.workspace_root = ""
        self.extra_collabs: list[dict] = []
        self.last_failure = ""
        self.last_music_path = ""

    def log(self, msg: str) -> None:
        if self.verbose:
            print(msg)

    def resolve_workspace_root(self) -> str:
        return resolve_workspace_repo(self.scenario)

    def build_send_metadata(self) -> dict | None:
        path = self.workspace_root
        if not path:
            return None
        ws_cfg = scenario_workspace(self.scenario) or {}
        outline = ws_cfg.get("outline", True)
        ctx = build_workspace_context_for_path(path, self.scenario)
        # Outline sends never include open-file bodies — avoids session/desktop bleed.
        if outline or ws_cfg.get("workspace_flag"):
            ctx["open_files"] = []
        meta: dict[str, Any] = {"workspace_context": ctx}
        if outline or ws_cfg.get("workspace_flag"):
            meta["context_scope"] = "outline"
        meta["collab_source_mode"] = "path"
        meta["collab_source_path"] = path
        return meta


def scenario_workspace(scenario: dict) -> dict | None:
    ws = scenario.get("workspace")
    if isinstance(ws, dict):
        return ws
    return None


def resolve_workspace_repo(scenario: dict) -> str:
    """Resolve workspace path for --repo and workspace_context (shared fallback for path_env)."""
    ws = scenario_workspace(scenario)
    if not ws:
        return ""
    path_env = (ws.get("path_env") or "").strip()
    if path_env:
        raw = os.environ.get(path_env, "").strip()
        if not raw and path_env == "NEURAL_JUNKIE_SCENARIO_REPO":
            fallback = ROOT / "scenarios" / "fixtures" / "minimal-repo"
            if fallback.is_dir():
                return str(fallback.resolve())
        if not raw:
            return ""
        p = Path(raw)
        return str(p.resolve()) if p.is_dir() else ""
    if ws.get("path"):
        p = Path(ws["path"])
        if not p.is_absolute():
            p = ROOT / p
        return str(p.resolve())
    fixture = (ws.get("fixture") or "").strip()
    if fixture:
        p = ROOT / "scenarios" / "fixtures" / fixture
        return str(p.resolve())
    return ""


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


def scenario_requires_gemini(scenario: dict) -> bool:
    for agent in scenario.get("required_agents") or []:
        if str(agent).strip().lower() == "gemini":
            return True
    agents_line = str(scenario.get("agents") or "")
    return "@gemini" in agents_line.lower()


def scenario_requires_claude(scenario: dict) -> bool:
    for agent in scenario.get("required_agents") or []:
        if str(agent).strip().lower() == "claude":
            return True
    agents_line = str(scenario.get("agents") or "")
    return "@claude" in agents_line.lower()


_gemini_probe_ok: bool | None = None
_gemini_probe_detail: str = ""

_claude_probe_ok: bool | None = None
_claude_probe_detail: str = ""


def prepare_gemini_for_collab(*, scenario_names: list[str], require: bool = False) -> bool:
    """Probe API keys × models once before Gemini collab scenarios."""
    global _gemini_probe_ok, _gemini_probe_detail
    if _gemini_probe_ok is not None:
        return _gemini_probe_ok
    if os.environ.get("NJ_SKIP_GEMINI_PROBE", "").strip().lower() in ("1", "true", "yes"):
        _gemini_probe_ok = True
        return True

    needs_gemini = False
    for name in scenario_names:
        try:
            if scenario_requires_gemini(load_scenario(name)):
                needs_gemini = True
                break
        except OSError:
            continue
    if not needs_gemini:
        _gemini_probe_ok = True
        return True

    from lib.gemini_judge_auth import ensure_gemini_for_testing  # noqa: E402
    from lib.release_prep_env import explicit_gemini_judge_model  # noqa: E402

    print("\n>>> Gemini preflight (keys × models)...", flush=True)
    sel = ensure_gemini_for_testing(
        root=ROOT,
        timeout_s=45.0,
        explicit_model=explicit_gemini_judge_model(ROOT),
        retry_quota=True,
        collab=True,
    )
    _gemini_probe_ok = sel.ok
    _gemini_probe_detail = sel.detail
    if sel.ok:
        print(f"OK: {sel.detail}", flush=True)
        return True

    msg = f"Gemini not available for collab testing: {sel.detail}"
    if require:
        print(f"FAIL: {msg}", file=sys.stderr)
        return False
    print(f"WARN: {msg}", file=sys.stderr)
    return False


def prepare_claude_for_collab(*, scenario_names: list[str], require: bool = False) -> bool:
    """Probe Claude Code CLI once before @Claude collab scenarios."""
    global _claude_probe_ok, _claude_probe_detail
    if _claude_probe_ok is not None:
        return _claude_probe_ok
    if os.environ.get("NJ_SKIP_CLAUDE_PROBE", "").strip().lower() in ("1", "true", "yes"):
        _claude_probe_ok = True
        return True

    needs_claude = False
    for name in scenario_names:
        try:
            if scenario_requires_claude(load_scenario(name)):
                needs_claude = True
                break
        except OSError:
            continue
    if not needs_claude:
        _claude_probe_ok = True
        return True

    # Ollama-routed Claude does not need cloud CLI auth.
    route = (os.environ.get("NJ_REGRESSION_CLAUDE_ROUTE") or "").strip().lower()
    mode = (os.environ.get("NJ_REGRESSION_CLAUDE_CLOUD") or "auto").strip().lower()
    if route == "ollama" or mode in ("0", "false", "no", "ollama", "local"):
        _claude_probe_ok = True
        _claude_probe_detail = "Claude on Ollama (cloud probe skipped)"
        return True

    from lib.claude_judge_auth import ensure_claude_for_testing  # noqa: E402

    print("\n>>> Claude preflight...", flush=True)
    # Claude CLI can occasionally flake on first probe (e.g. transient auth/IPC hiccup).
    # Retry once so required @Claude scenarios don't fail at sweep start.
    sel = ensure_claude_for_testing(timeout_s=45.0)
    if not sel.ok:
        time.sleep(2.0)
        sel = ensure_claude_for_testing(timeout_s=45.0)
    _claude_probe_ok = sel.ok
    _claude_probe_detail = sel.detail
    if sel.ok:
        print(f"OK: {sel.detail}", flush=True)
        return True

    # auto mode: boot may already have fallen back to Ollama; don't fail the sweep.
    if mode in ("", "auto") and not require:
        print(f"WARN: Claude cloud auth unavailable ({sel.detail}); continuing (expect Ollama Claude)", flush=True)
        _claude_probe_ok = True
        return True

    msg = f"Claude not available for collab testing: {sel.detail}"
    if require:
        print(f"FAIL: {msg}", file=sys.stderr)
        return False
    print(f"WARN: {msg}", file=sys.stderr)
    return False


def _strip_flag(flags: list[str], name: str) -> list[str]:
    out: list[str] = []
    skip_next = False
    for f in flags:
        if skip_next:
            skip_next = False
            continue
        if f == name:
            skip_next = True
            continue
        out.append(f)
    return out


def apply_flag_overrides(flags: list[str]) -> list[str]:
    """Apply NJ_SCENARIO_ROUNDS / NJ_SCENARIO_MESSAGES for matrix sweeps."""
    flags = list(flags)
    env_rounds = os.environ.get("NJ_SCENARIO_ROUNDS", "").strip()
    env_messages = os.environ.get("NJ_SCENARIO_MESSAGES", "").strip()
    if env_rounds:
        flags = _strip_flag(flags, "--rounds")
        flags = ["--rounds", env_rounds, *flags]
    if env_messages:
        flags = _strip_flag(flags, "--messages")
        flags = ["--messages", env_messages, *flags]
    return flags


def _shell_quote_arg(value: str) -> str:
    if any(ch.isspace() for ch in value):
        return "'" + value.replace("'", "'\"'\"'") + "'"
    return value


def build_collaborate_command(scenario: dict, agents: str) -> str:
    collab = scenario.get("collaborate") or {}
    flags = collab.get("flags") or []
    if isinstance(flags, str):
        flags = flags.split()
    flags = apply_flag_overrides(list(flags))
    goal = (collab.get("goal") or "").strip()
    ws = scenario_workspace(scenario)
    if ws and ws.get("workspace_flag"):
        if "--workspace" not in flags:
            flags = ["--workspace", *flags]
    repo = resolve_workspace_repo(scenario) if ws else ""
    if repo and "--repo" not in flags:
        flags = ["--repo", _shell_quote_arg(repo), *flags]
    # Ensure every collaborate roster agent is @mentioned in the command tail
    # (hub requires >=2 mentions; goals often repeat only the first agent).
    roster = hub.collaborate_agent_names(scenario, (agents or "").strip())
    goal_mentions = set(hub.parse_agent_mentions(goal))
    missing = [name for name in roster if name not in goal_mentions]
    if missing:
        prefix = " ".join(f"@{name}" for name in missing)
        goal = f"{prefix} {goal}".strip() if goal else prefix
    parts = ["/collaborate", *flags, goal]
    return " ".join(p for p in parts if p)


def start_collaboration(
    ctx: ScenarioContext,
    channel: str,
    scenario: dict,
    agents: str,
) -> tuple[str, str] | None:
    ctx.workspace_root = ctx.resolve_workspace_root()
    content = build_collaborate_command(scenario, agents)
    meta = ctx.build_send_metadata()
    code, data = hub.send_message(ctx.base, channel, content, metadata=meta)
    if code != 200 or not isinstance(data, dict):
        print(f"  FAIL: POST /api/send ({code}): {data}", file=sys.stderr)
        return None
    cid = data.get("collaboration_id") or ""
    ch = data.get("collaboration_channel") or ""
    if not cid or not ch:
        err = hub.last_system_error(ctx.base, channel)
        print(f"  FAIL: no collaboration redirect: {data}", file=sys.stderr)
        if err:
            print(f"  hub: {err}", file=sys.stderr)
        return None
    return cid, ch


def dump_transcript(ctx: ScenarioContext, tail: int = 8) -> None:
    if not ctx.collab_channel:
        return
    msgs = hub.list_messages(ctx.base, ctx.collab_channel, 200)
    agent_msgs = hub.agent_messages(msgs)
    print("\n  --- transcript (agent messages) ---", file=sys.stderr)
    for m in agent_msgs[-tail:]:
        who = (m.get("from") or {}).get("name", "?")
        typ = m.get("type", "?")
        body = (m.get("content") or "").replace("\n", " ")[:160]
        print(f"    [{typ}] {who}: {body}", file=sys.stderr)
    print("  --- end ---\n", file=sys.stderr)


def step_wait_phase(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    phase = step.get("phase") or ""
    timeout = hub.parse_timeout(step.get("timeout", 90))
    if not ctx.collab_id:
        return False, "no active collaboration"
    if hub.wait_phase(ctx.base, ctx.collab_channel, ctx.collab_id, phase, timeout):
        return True, f"phase={phase}"
    return False, f"timeout waiting for phase {phase!r} (last={hub.collab_phase(ctx.base, ctx.collab_channel, ctx.collab_id)!r})"


def step_wait_planning_recap(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    timeout = hub.parse_timeout(step.get("timeout", 90))
    if hub.wait_planning_recap(ctx.base, ctx.collab_channel, ctx.collab_id, timeout):
        c = hub.fetch_collab(ctx.base, ctx.collab_channel, ctx.collab_id)
        st = (c or {}).get("planning_recap_status", "")
        return True, f"planning_recap_status={st}"
    return False, "planning recap still pending"


def _discussion_requirements_met(
    counts: dict[str, int],
    total: int,
    *,
    min_total: int,
    min_per: int,
    required_agents: list[str],
) -> bool:
    if total < min_total:
        return False
    if not required_agents:
        if min_per <= 0:
            return True
        return bool(counts) and all(n >= min_per for n in counts.values())
    return all(counts.get(name, 0) >= max(min_per, 1) for name in required_agents)


def _nudge_discussion_agents(ctx: ScenarioContext, names: list) -> None:
    meta: dict[str, object] | None = None
    if ctx.collab_id:
        meta = {
            "collaboration_id": ctx.collab_id,
            "collaboration_phase": "planning",
            # Keep nudges as collaboration_discussion so image tools stay suppressed.
            "type": "collaboration_discussion",
        }
    for raw in names:
        name = str(raw).strip().lstrip("@")
        if not name:
            continue
        msg = (
            f"@{name} — please add your planning perspective with concrete "
            f"`- Task N: @Agent - Write collabs/<id>/file.ext …` lines."
        )
        nudge_meta = dict(meta) if meta else {}
        nudge_meta["mentions"] = [name]
        hub.send_message(ctx.base, ctx.collab_channel, msg, metadata=nudge_meta)
        ctx.log(f"  nudge: {msg}")


def _generation_error_agents(msgs: list[dict]) -> list[str]:
    names: list[str] = []
    for m in msgs:
        if (m.get("type") or "") != "collaboration_discussion":
            continue
        meta = m.get("metadata") or {}
        if not (meta.get("generation_error") or meta.get("error_code")):
            continue
        who = (m.get("from") or {}).get("name", "").strip()
        if who and who not in names:
            names.append(who)
    return names


def step_wait_discussion(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    timeout = hub.parse_timeout(step.get("timeout", 90))
    min_total = int(step.get("min_total", 1))
    min_per = int(step.get("min_per_agent", 0))
    max_total = int(step.get("max_total", 0))
    required = step.get("required_agents") or ctx.scenario.get("required_agents") or []
    retries = max(0, int(step.get("retries", 0)))
    nudge_agents = step.get("nudge_agents") or []
    retry_on_gen_error = bool(step.get("retry_on_generation_error", True))
    nudge_silent = bool(step.get("nudge_silent_agents", True))
    nudged_silent: set[str] = set()
    started = time.time()

    for attempt in range(retries + 1):
        deadline = time.time() + timeout
        gen_error_nudged: set[str] = set()
        while time.time() < deadline:
            raw_msgs = hub.list_messages(ctx.base, ctx.collab_channel, 200)
            scoped = hub.messages_for_collab(raw_msgs, ctx.collab_id)
            msgs = hub.agent_messages(scoped, exclude_generation_errors=True)
            counts = hub.count_by_agent(msgs)
            total = len(msgs)
            if retry_on_gen_error:
                err_agents = _generation_error_agents(scoped)
                fresh = [a for a in err_agents if a not in gen_error_nudged]
                if fresh:
                    gen_error_nudged.update(fresh)
                    ctx.log(f"  wait_discussion: generation_error from {fresh}; nudging")
                    _nudge_discussion_agents(ctx, fresh)
                    deadline = min(deadline + 60.0, time.time() + timeout * 1.5)
                    time.sleep(3)
                    continue
            phase = (
                hub.collab_phase(ctx.base, ctx.collab_channel, ctx.collab_id)
                if ctx.collab_id
                else None
            )
            planning_ready = bool(
                ctx.collab_id
                and (
                    phase == "reviewing"
                    or hub.planning_discussion_ready(
                        ctx.base, ctx.collab_channel, ctx.collab_id
                    )
                )
            )
            per_agent_ok = (
                all(counts.get(name, 0) >= max(min_per, 1) for name in required)
                if required
                else (
                    min_per <= 0
                    or (bool(counts) and all(n >= min_per for n in counts.values()))
                )
            )
            requirements_met = _discussion_requirements_met(
                counts, total, min_total=min_total, min_per=min_per, required_agents=required
            )
            if requirements_met and per_agent_ok and (planning_ready or phase == "reviewing"):
                suffix = f" (after retry {attempt})" if attempt else ""
                return (
                    True,
                    f"messages total={total} by_agent={counts}; planning ready{suffix}",
                )
            # Do not short-circuit on participation alone: scenarios that next
            # wait for reviewing will flake if we proceed while still planning.
            if (
                bool(step.get("accept_participation", False))
                and requirements_met
                and per_agent_ok
            ):
                suffix = f" (after retry {attempt})" if attempt else ""
                return True, f"messages total={total} by_agent={counts}; participation ready{suffix}"
            if nudge_silent and required and time.time() - started > timeout * 0.35:
                silent = [
                    name
                    for name in required
                    if counts.get(name, 0) < max(min_per, 1) and name not in nudged_silent
                ]
                if silent:
                    nudged_silent.update(silent)
                    ctx.log(f"  wait_discussion: silent agents {silent}; nudging")
                    _nudge_discussion_agents(ctx, silent)
                    time.sleep(3)
                    continue
            if max_total > 0 and total > max_total:
                diag = hub.discussion_diagnosis(
                    ctx.base,
                    ctx.collab_channel,
                    required_agents=required,
                    collab_id=ctx.collab_id,
                )
                return False, f"too many messages ({total} > {max_total})\n{diag}"
            time.sleep(hub.POLL_INTERVAL)

        if attempt < retries:
            diag = hub.discussion_diagnosis(
                ctx.base,
                ctx.collab_channel,
                required_agents=required,
                collab_id=ctx.collab_id,
            )
            ctx.log(f"  wait_discussion attempt {attempt + 1} timed out; retrying\n{diag}")
            if nudge_agents:
                _nudge_discussion_agents(ctx, nudge_agents)
            time.sleep(5)
            continue

    diag = hub.discussion_diagnosis(
        ctx.base,
        ctx.collab_channel,
        required_agents=required,
        collab_id=ctx.collab_id,
    )
    scoped = hub.messages_for_collab(
        hub.list_messages(ctx.base, ctx.collab_channel, 200), ctx.collab_id
    )
    counts = hub.count_by_agent(hub.agent_messages(scoped))
    need = (
        f"need total>={min_total}"
        + (f", each of {required} >= {max(min_per,1)}" if required else "")
    )
    return False, f"discussion timeout ({need}): counts={counts}\n{diag}"


def step_assert_messages(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    msgs = hub.list_messages(ctx.base, ctx.collab_channel, 200)
    types = step.get("types")
    type_set = frozenset(types) if types else None
    pool = hub.agent_messages(msgs, types=type_set)
    if step.get("include_system_discussion"):
        pool = [m for m in msgs if m.get("type") in ("collaboration_discussion", "answer", "command_output")]

    phase_filter = (step.get("phase") or step.get("since_phase") or "").strip()
    if phase_filter:
        pool = [
            m
            for m in pool
            if ((m.get("metadata") or {}).get("collaboration_phase") or "") == phase_filter
        ]

    for pattern in step.get("none_match") or []:
        hits = hub.messages_matching(pool, pattern)
        if hits:
            who = (hits[0].get("from") or {}).get("name")
            return False, f"none_match violated {pattern!r} ({who})"

    if any_patterns := step.get("any_match") or []:
        if not any(hub.messages_matching(pool, pattern) for pattern in any_patterns):
            agents = sorted({(m.get("from") or {}).get("name", "?") for m in pool})
            return False, (
                f"any_match not found (want one of {any_patterns!r}) "
                f"(agents with discussion: {agents or 'none'})"
            )

    for pattern in step.get("all_match") or []:
        agent_msgs = [m for m in pool if (m.get("from") or {}).get("name") != "System"]
        if agent_msgs and not all(hub.messages_matching([m], pattern) for m in agent_msgs[:3]):
            pass  # all_match optional strictness
        if not hub.messages_matching(pool, pattern):
            return False, f"all_match not found: {pattern!r}"

    if step.get("require_task_status"):
        if not hub.messages_matching(pool, r"TASK_STATUS:\s*completed"):
            return False, "no TASK_STATUS: completed in messages"

    if step.get("deny_task_status_blocked"):
        if hub.messages_matching(pool, r"TASK_STATUS:\s*blocked"):
            return False, "TASK_STATUS: blocked present in messages"

    if step.get("deny_failed_command_output"):
        for m in msgs:
            if m.get("type") != "command_output":
                continue
            content = (m.get("content") or "").lower()
            if "findings.md" in content and "command not found" in content:
                return False, "findings.md was executed as shell command"

    if step.get("deny_suggested_stack_commands"):
        for m in pool:
            meta = m.get("metadata") or {}
            raw_cmds = meta.get("suggested_commands") or []
            if not isinstance(raw_cmds, list):
                continue
            for item in raw_cmds:
                if not isinstance(item, dict):
                    continue
                cmd = (item.get("command") or "").strip()
                if looks_like_stack_tool_command(cmd):
                    who = (m.get("from") or {}).get("name", "?")
                    return False, f"stack command suggestion from {who}: {cmd[:100]!r}"

    if step.get("deny_json_discussion"):
        for m in pool:
            if m.get("type") != "collaboration_discussion":
                continue
            content = (m.get("content") or "").strip()
            if content.startswith("{") and content.endswith("}"):
                who = (m.get("from") or {}).get("name", "?")
                return False, f"JSON discussion from {who}: {content[:120]!r}"

    if step.get("deny_file_change_after_cancel"):
        cancelled_at: int | None = None
        for i, m in enumerate(msgs):
            if m.get("type") == "system_info" and "Cancelled" in (m.get("content") or ""):
                cancelled_at = i
        if cancelled_at is None:
            return False, "deny_file_change_after_cancel: no cancellation system message"
        for m in msgs[cancelled_at + 1 :]:
            if m.get("type") != "file_change":
                continue
            who = (m.get("from") or {}).get("name", "?")
            return False, f"file_change after cancel from {who}"

    if step.get("deny_generation_errors"):
        allow_recovered = bool(step.get("allow_recovered_generation_errors"))
        errored: set[str] = set()
        recovered: set[str] = set()
        for m in pool:
            meta = m.get("metadata") or {}
            who = (m.get("from") or {}).get("name", "?")
            if meta.get("generation_error") or meta.get("error_code"):
                errored.add(who)
            elif who != "?":
                recovered.add(who)
        if allow_recovered:
            errored = {a for a in errored if a not in recovered}
        for m in pool:
            meta = m.get("metadata") or {}
            if not (meta.get("generation_error") or meta.get("error_code")):
                continue
            who = (m.get("from") or {}).get("name", "?")
            if allow_recovered and who in recovered:
                continue
            code = meta.get("error_code") or "generation_error"
            return False, f"generation error from {who}: {code}"

    return True, "message assertions ok"


def step_assert_collab(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    c = hub.fetch_collab(ctx.base, ctx.collab_channel, ctx.collab_id)
    if not c:
        return False, "collaboration snapshot missing"

    if "phase" in step:
        want = step["phase"]
        if c.get("phase") != want:
            return False, f"phase={c.get('phase')!r} want {want!r}"

    if "phase_in" in step:
        allowed = step["phase_in"]
        if not isinstance(allowed, list):
            allowed = [allowed]
        if c.get("phase") not in allowed:
            return False, f"phase={c.get('phase')!r} want one of {allowed!r}"

    src = (c.get("source_repo_path") or "").strip()
    if step.get("source_repo_path_empty") and src:
        return False, f"expected empty source_repo_path, got {src!r}"
    if step.get("source_repo_path_nonempty") and not src:
        return False, "expected non-empty source_repo_path"

    if step.get("expect_workspace_warning"):
        err = hub.last_system_error(ctx.base, ctx.channel)
        ok = ("Ignored" in err and "workspace" in err) or "deliverables folder" in err
        if not ok:
            return False, f"expected workspace warning on parent channel, got: {err!r}"

    if step.get("workspace_acknowledged") is True and not c.get("workspace_acknowledged"):
        return False, "expected workspace_acknowledged=true after sandbox auto-ack on approve"
    if step.get("workspace_acknowledged") is False and c.get("workspace_acknowledged"):
        return False, "expected workspace_acknowledged=false"

    if step.get("tasks_dispatched") is True and not c.get("tasks_dispatched"):
        return False, "expected tasks_dispatched=true"

    return True, "collab snapshot ok"


def step_assert_plan(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    c = hub.fetch_collab(ctx.base, ctx.collab_channel, ctx.collab_id)
    if not c:
        return False, "no collaboration"
    plan = c.get("plan") or {}
    content = (plan.get("content") or "").strip()
    tasks = c.get("tasks") or []

    min_tasks = int(step.get("min_tasks", 0))
    max_tasks = int(step.get("max_tasks", 0))
    if max_tasks > 0 and len(tasks) > max_tasks:
        return False, f"tasks={len(tasks)} want <={max_tasks} (parser explosion?)"
    if min_tasks > 0 and len(tasks) < min_tasks:
        effective = 0
        if content:
            effective = max(
                len(re.findall(r"(?m)^\s*[-*]\s+Task\s+\d+:", content, re.I)),
                len(re.findall(r"(?m)^\s*\d+\.\s+\*\*", content)),
                len(re.findall(r"(?m)^\s*[-*]\s+\*\*Task\s+\d+", content, re.I)),
            )
        if effective < min_tasks:
            return False, f"tasks={len(tasks)} plan_task_lines≈{effective} want >={min_tasks}"

    if step.get("tasks_have_assignee"):
        for t in tasks:
            if isinstance(t, dict) and not (t.get("assigned_to") or t.get("assigned_name")):
                return False, f"task without assignee: {(t.get('title') or '')[:50]}"

    for prefix in step.get("title_none_match") or []:
        for t in tasks:
            if not isinstance(t, dict):
                continue
            title = (t.get("title") or "").lower()
            if title.startswith(prefix.lower()):
                return False, f"weak task title: {t.get('title')!r}"

    for pattern in step.get("task_none_match") or []:
        for t in tasks:
            if not isinstance(t, dict):
                continue
            combined = f"{t.get('title') or ''} {t.get('description') or ''}"
            if re.search(pattern, combined, re.I):
                return False, f"task_none_match {pattern!r}: {combined[:100]!r}"

    assignee_min = step.get("assignee_min_tasks") or {}
    if isinstance(assignee_min, dict):
        for assignee, min_n in assignee_min.items():
            want = int(min_n)
            if want <= 0:
                continue
            count = sum(
                1
                for t in tasks
                if isinstance(t, dict)
                and (t.get("assigned_name") or "").strip().lower() == assignee.strip().lower()
            )
            if count < want:
                return False, f"assignee {assignee!r} tasks={count} want >={want}"

    plan_text = content + "\n" + "\n".join(
        (t.get("title") or "") for t in tasks if isinstance(t, dict)
    )
    for pattern in step.get("content_none_match") or []:
        if re.search(pattern, plan_text, re.I):
            return False, f"plan content_none_match {pattern!r}"

    for pattern in step.get("content_any_match") or []:
        if not re.search(pattern, plan_text, re.I):
            return False, f"plan content_any_match not found: {pattern!r}"

    if step.get("tasks_have_file_deliverable"):
        file_tasks = 0
        for t in tasks:
            if not isinstance(t, dict):
                continue
            combined = f"{t.get('title') or ''} {t.get('description') or ''}".lower()
            if "write " in combined or ".md" in combined or "collabs/" in combined:
                file_tasks += 1
        min_file = int(step.get("min_file_tasks", 1))
        if file_tasks < min_file:
            return False, f"tasks with file deliverable hints={file_tasks} want >={min_file}"

    return True, f"plan ok (tasks={len(tasks)})"


def step_send(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    content = (step.get("content") or "").strip().replace("<collab-id>", ctx.collab_id)
    channel = step.get("channel") or ctx.collab_channel or ctx.channel
    if not content:
        return False, "empty send content"
    timeout = float(step.get("timeout", 300 if content.startswith("/") else 60))
    code, _ = hub.send_message(ctx.base, channel, content, timeout=timeout)
    if code != 200:
        err = hub.last_system_error(ctx.base, channel)
        detail = f" ({err})" if err else ""
        return False, f"send failed ({code}){detail}"
    time.sleep(0.5)
    return True, content[:60]


def step_approve_plan(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    if not ctx.collab_id:
        return False, "no collab_id"
    code, _ = hub.send_message(
        ctx.base,
        ctx.collab_channel,
        f"/approve-plan {ctx.collab_id[:8]}",
        timeout=300,
    )
    if code != 200:
        err = hub.last_system_error(ctx.base, ctx.collab_channel)
        detail = f" ({err})" if err else ""
        return False, f"approve-plan failed ({code}){detail}"
    time.sleep(0.5)
    return True, "approve-plan sent"


def step_workspace_ack(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    path = step.get("source_repo_path") or ctx.workspace_root
    code = hub.workspace_ack(ctx.base, ctx.collab_id, path)
    if code != 204:
        return False, f"workspace ack failed ({code})"
    return True, "workspace ack"


def step_wait_tasks(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    if step.get("settle_only"):
        delay = hub.parse_timeout(step.get("timeout", 60))
        time.sleep(delay)
        c = hub.fetch_collab(ctx.base, ctx.collab_channel, ctx.collab_id)
        tasks = (c or {}).get("tasks") or []
        statuses = [t.get("status") for t in tasks if isinstance(t, dict)]
        return True, f"executing settle {delay}s statuses={statuses}"

    timeout = hub.parse_timeout(step.get("timeout", 180))
    want = step.get("status") or "completed"
    min_completed = int(step.get("min_completed", 0))
    all_match = bool(step.get("all_match", True))
    if hub.wait_tasks_status(
        ctx.base,
        ctx.collab_channel,
        ctx.collab_id,
        want_status=want,
        min_completed=min_completed,
        all_match=all_match,
        timeout=timeout,
    ):
        return True, f"tasks {want}"
    c = hub.fetch_collab(ctx.base, ctx.collab_channel, ctx.collab_id)
    tasks = (c or {}).get("tasks") or []
    statuses = [t.get("status") for t in tasks if isinstance(t, dict)]
    return False, f"task wait timeout statuses={statuses}"


def _is_deliverable_stub(path: Path) -> bool:
    if not path.is_file():
        return False
    try:
        body = path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return False
    return "_Initial stub created when the plan was approved" in body


def _deliverable_file_ready(
    full: Path,
    *,
    min_bytes: int,
    allow_fallback: bool,
) -> bool:
    _ = allow_fallback  # call-site compat; stubs never count as ready
    if not full.is_file() or full.stat().st_size < min_bytes:
        return False
    # Plan-approval placeholders must not satisfy approve_file_changes / on-disk checks.
    return not _is_deliverable_stub(full)


_HOME_HTML_NAMES = ("index.html", "homepage.html", "home.html")


def _html_file_priority(name: str) -> int:
    lower = name.lower()
    for idx, preferred in enumerate(_HOME_HTML_NAMES):
        if lower == preferred:
            return idx
    return len(_HOME_HTML_NAMES)


def _find_collab_deliverable_on_disk(
    workspace_root: str,
    collab_id: str,
    path_match: str,
    *,
    min_bytes: int,
    allow_fallback: bool,
    prefer_home_html: bool = False,
) -> Path | None:
    """True when CLI agents auto-applied files (--yolo) with nothing left in pending queue."""
    if not workspace_root or not collab_id or not path_match:
        return None
    collab_dir = Path(workspace_root) / "collabs" / collab_id
    if not collab_dir.is_dir():
        return None
    needle = path_match.lower().lstrip(".")
    candidates: list[Path] = []
    for path in collab_dir.rglob("*"):
        if not path.is_file():
            continue
        if needle not in path.name.lower():
            continue
        if _deliverable_file_ready(path, min_bytes=min_bytes, allow_fallback=allow_fallback):
            candidates.append(path)
    if not candidates:
        return None
    if prefer_home_html or needle == "html":
        candidates.sort(key=lambda p: (_html_file_priority(p.name), p.name))
    else:
        candidates.sort(key=lambda p: p.name)
    return candidates[0]


def step_approve_file_changes(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    if not ctx.collab_channel:
        return False, "no collab_channel"
    timeout = hub.parse_timeout(step.get("timeout", 60))
    path_match = (step.get("path_match") or step.get("path_contains") or "").strip()
    min_approved = int(step.get("min_approved", 1))
    min_bytes = int(step.get("min_bytes", 20))
    target_rel = (step.get("target_rel") or "").replace("<collab-id>", ctx.collab_id)
    expect_rel = (step.get("expect_path") or target_rel or "").replace("<collab-id>", ctx.collab_id)
    require_hub = bool(step.get("require_hub_approval", False))
    allow_fallback = bool(step.get("from_discussion_fallback", False))
    if env_or_automation_bool("NJ_SCENARIO_ALLOW_FILE_FALLBACK", "scenario_allow_file_fallback", False):
        allow_fallback = True

    # Auto-approve empties the pending queue quickly. When min_approved=0, poll for a
    # non-stub on-disk deliverable across the full timeout instead of exiting immediately.
    if min_approved == 0 and ctx.workspace_root and (path_match or expect_rel):
        deadline = time.time() + timeout
        last_ids: list[str] = []
        while time.time() < deadline:
            n, last_ids = hub.wait_and_approve_file_changes(
                ctx.base,
                ctx.collab_channel,
                path_contains=path_match,
                min_approved=0,
                timeout=min(2.0, max(0.5, deadline - time.time())),
            )
            if path_match and ctx.collab_id:
                on_disk = _find_collab_deliverable_on_disk(
                    ctx.workspace_root,
                    ctx.collab_id,
                    path_match,
                    min_bytes=min_bytes,
                    allow_fallback=allow_fallback,
                    prefer_home_html=path_match.strip().lower() in (".html", "html"),
                )
                if on_disk:
                    return True, f"deliverable on disk ({on_disk.name})"
            if expect_rel:
                full = Path(ctx.workspace_root) / expect_rel
                if _deliverable_file_ready(full, min_bytes=min_bytes, allow_fallback=allow_fallback):
                    return True, f"file exists ({full})"
            time.sleep(hub.POLL_INTERVAL)

        if allow_fallback:
            msgs = hub.list_messages(ctx.base, ctx.collab_channel, 200)
            written = hub.write_loose_file_change_from_messages(
                msgs,
                ctx.workspace_root,
                path_contains=path_match,
                target_rel=target_rel,
                min_bytes=min_bytes,
            )
            if written:
                return True, f"discussion fallback wrote {written}"
            if target_rel:
                written = _materialize_solo_findings(
                    msgs,
                    ctx.workspace_root,
                    target_rel,
                    min_bytes,
                )
                if written:
                    return True, f"discussion fallback materialized {written}"

        label = expect_rel or path_match or "deliverable"
        return False, (
            f"deliverable not on disk after auto-approve wait "
            f"(path={label}, pending_approved={len(last_ids)}, ids={last_ids})"
        )

    n, ids = hub.wait_and_approve_file_changes(
        ctx.base,
        ctx.collab_channel,
        path_contains=path_match,
        min_approved=min_approved,
        timeout=timeout,
    )

    if ctx.workspace_root and ctx.collab_id and path_match:
        on_disk = _find_collab_deliverable_on_disk(
            ctx.workspace_root,
            ctx.collab_id,
            path_match,
            min_bytes=min_bytes,
            allow_fallback=allow_fallback,
            prefer_home_html=path_match.strip().lower() in (".html", "html"),
        )
        if on_disk and (n >= min_approved or min_approved == 0):
            return True, f"deliverable on disk ({on_disk.name})"

    if expect_rel and ctx.workspace_root:
        full = Path(ctx.workspace_root) / expect_rel
        if _deliverable_file_ready(full, min_bytes=min_bytes, allow_fallback=allow_fallback):
            return True, f"file exists ({full})"

    if n >= min_approved and min_approved > 0:
        return True, f"hub approved {n}: {ids}"

    if require_hub and not allow_fallback:
        return False, f"require_hub_approval: approved={n} (need >={min_approved})"

    hub_approval_failed = min_approved > 0 and n < min_approved

    if expect_rel and ctx.workspace_root:
        full = Path(ctx.workspace_root) / expect_rel
        if full.is_file() and not require_hub and not _is_deliverable_stub(full):
            return True, f"file already exists ({full})"

    if allow_fallback and ctx.workspace_root:
        msgs = hub.list_messages(ctx.base, ctx.collab_channel, 200)
        written = hub.write_loose_file_change_from_messages(
            msgs,
            ctx.workspace_root,
            path_contains=path_match,
            target_rel=target_rel,
            min_bytes=min_bytes,
        )
        if written:
            if require_hub:
                return False, f"hub approval missing; only fallback wrote {written}"
            return True, f"discussion fallback wrote {written}"

        if target_rel and (hub_approval_failed or min_approved == 0):
            written = _materialize_solo_findings(
                msgs,
                ctx.workspace_root,
                target_rel,
                min_bytes,
            )
            if written:
                if require_hub:
                    return False, f"hub approval missing; only materialized {written}"
                return True, f"discussion fallback materialized {written}"

    if min_approved == 0 and expect_rel and ctx.workspace_root:
        full = Path(ctx.workspace_root) / expect_rel
        if _deliverable_file_ready(full, min_bytes=min_bytes, allow_fallback=allow_fallback):
            return True, f"file exists ({full})"

    return False, f"no file change approved (pending={n}, ids={ids})"


def step_assert_deliverable(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    spec = merge_deliverable_step(ctx.scenario, step)
    rel = (spec.get("path") or "").strip()
    if not rel:
        return False, "assert_deliverable: path required"
    root = step.get("root") or ctx.workspace_root
    if not root:
        return False, "assert_deliverable: no workspace root"
    alts = step.get("path_alternatives") or spec.get("path_alternatives") or []
    candidates = [rel] + [str(a).strip() for a in alts if str(a).strip()]
    last_detail = ""
    for candidate in candidates:
        ok, detail = check_file_deliverable(
            root=root,
            rel=candidate,
            spec=spec,
            question=scenario_question(ctx.scenario),
            collab_id=ctx.collab_id,
            hub_base=ctx.base,
        )
        if ok:
            return True, detail
        last_detail = detail

    scan_suffix = (step.get("path_any_in_collab") or spec.get("path_any_in_collab") or "").strip()
    if scan_suffix and ctx.collab_id:
        collab_dir = Path(root) / "collabs" / ctx.collab_id
        if collab_dir.is_dir():
            suffix = scan_suffix if scan_suffix.startswith(".") else f".{scan_suffix}"
            scanned: list[Path] = []
            # Home-page scan is for index.html aliases only — not every .html deliverable.
            home_only = suffix.lower() == ".html" and (
                rel.endswith("index.html")
                or step.get("path_alternatives")
                or spec.get("path_alternatives")
            )
            if home_only:
                for name in _HOME_HTML_NAMES:
                    path = collab_dir / name
                    if path.is_file():
                        scanned.append(path)
            else:
                for path in collab_dir.rglob("*"):
                    if path.is_file() and path.name.lower().endswith(suffix.lower()):
                        scanned.append(path)
            if suffix.lower() == ".html" and not home_only:
                scanned.sort(key=lambda p: p.name)
            elif suffix.lower() == ".html":
                scanned.sort(key=lambda p: (_html_file_priority(p.name), p.name))
            else:
                scanned.sort(key=lambda p: p.name)
            for path in scanned:
                rel_scan = str(path.relative_to(Path(root)))
                ok, detail = check_file_deliverable(
                    root=root,
                    rel=rel_scan,
                    spec=spec,
                    question=scenario_question(ctx.scenario),
                    collab_id=ctx.collab_id,
                    hub_base=ctx.base,
                )
                if ok:
                    return True, f"{detail} (scanned {rel_scan})"
                last_detail = detail

    return False, last_detail


def step_assert_files(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    return step_assert_deliverable(ctx, step)


def step_assert_deliverable_stubs(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    if not ctx.collab_id or not ctx.workspace_root:
        return False, "need collab_id and workspace root"
    rel_dir = f"collabs/{ctx.collab_id}"
    collab_dir = Path(ctx.workspace_root) / rel_dir
    if not collab_dir.is_dir():
        return False, f"missing deliverables dir {collab_dir}"

    min_stubs = int(step.get("min_stubs", 1))
    skip_names = {n.lower() for n in (step.get("skip_names") or ["README.md", "planning-summary.md", "session-summary.md"])}
    stubs = [
        p
        for p in collab_dir.glob("*.md")
        if p.is_file() and p.name.lower() not in skip_names
    ]
    if step.get("require_plan_md", True):
        plan_md = collab_dir / "plan.md"
        if not plan_md.is_file():
            return False, f"missing {plan_md}"
        stubs = [p for p in stubs if p.name.lower() != "plan.md"]
        deliverable_stubs = [p for p in collab_dir.glob("*.md") if p.name.lower() not in skip_names | {"plan.md"}]
    else:
        deliverable_stubs = stubs

    if len(deliverable_stubs) < min_stubs:
        names = sorted(p.name for p in collab_dir.glob("*"))
        return False, f"deliverable stubs={len(deliverable_stubs)} want >={min_stubs} in {collab_dir} files={names}"

    c = hub.fetch_collab(ctx.base, ctx.collab_channel, ctx.collab_id)
    wd = ((c or {}).get("working_directory") or "").strip()
    if wd and Path(wd).resolve() != collab_dir.resolve():
        return False, f"working_directory={wd!r} != {collab_dir!r}"

    return True, f"{len(deliverable_stubs)} stub(s) in {collab_dir}"


def step_setup_pack(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    pack_id = (step.get("pack_id") or "").strip()
    if not pack_id:
        return False, "pack_id required"
    code, out = hub.hub_request(ctx.base, "PUT", f"/api/packs/{pack_id}", {"enabled": True})
    if code != 200:
        return False, f"enable pack {pack_id}: {code} {out}"
    return True, f"enabled pack {pack_id}"


def _browser_post(base: str, path: str, body: dict) -> tuple[int, dict]:
    url = f"{base.rstrip('/')}{path}"
    raw = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=raw, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return resp.status, data if isinstance(data, dict) else {}
    except urllib.error.HTTPError as exc:
        try:
            data = json.loads(exc.read().decode("utf-8"))
        except Exception:
            data = {"error": exc.reason or str(exc)}
        return exc.code, data if isinstance(data, dict) else {"error": str(data)}


def _cad_post(base: str, path: str, body: dict) -> tuple[int, dict]:
    url = f"{base.rstrip('/')}{path}"
    raw = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=raw, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return resp.status, data if isinstance(data, dict) else {}
    except urllib.error.HTTPError as exc:
        try:
            data = json.loads(exc.read().decode("utf-8"))
        except Exception:
            data = {"error": exc.reason or str(exc)}
        return exc.code, data if isinstance(data, dict) else {"error": str(data)}


def step_assert_cad_render(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    pack_dir = (step.get("pack_dir") or os.environ.get("NJ_PACK_DIR") or "").strip()
    fixture = (step.get("fixture_scad") or step.get("path") or "").strip()
    if not fixture:
        return False, "fixture_scad or path required"
    scad_path = fixture
    if pack_dir and not Path(fixture).is_absolute():
        scad_path = str((Path(pack_dir) / fixture).resolve())
    elif ctx.workspace_root and not Path(fixture).is_absolute():
        scad_path = str((Path(ctx.workspace_root) / fixture).resolve())
    params = step.get("params") if isinstance(step.get("params"), dict) else {}
    body = {"path": scad_path, "params": params, "cad_sidecar_dry_run": True}
    code, data = _cad_post(ctx.base, "/api/cad/render", body)
    if code != 200:
        return False, data.get("error") or f"cad render HTTP {code}"
    b64 = (data.get("content_base64") or "").strip()
    min_bytes = int(step.get("min_stl_b64_bytes", 80))
    if len(b64) < min_bytes:
        return False, f"stl b64 len {len(b64)} < {min_bytes}"
    return True, f"cad render ok ({len(b64)} b64 bytes)"


def step_assert_browser_screenshot(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    ws_root = ctx.workspace_root or ctx.resolve_workspace_root()
    if not ws_root:
        return False, "workspace root required"
    rel_path = (step.get("rel_path") or step.get("path") or "").strip().lstrip("/")
    if not rel_path:
        return False, "rel_path or path required"
    rel_path = rel_path.replace("<collab-id>", ctx.collab_id or "")
    url = (step.get("url") or "").strip()
    if not url:
        ws_id = (step.get("workspace_id") or "").strip()
        if not ws_id:
            # Derive workspace id from hub is non-trivial; use file URL for local fixtures.
            file_path = Path(ws_root) / rel_path
            if not file_path.is_file():
                return False, f"missing file {file_path}"
            url = file_path.resolve().as_uri()
        else:
            from urllib.parse import urlencode

            hub_host = ctx.base.rstrip("/")
            url = f"{hub_host}/api/workspace-preview?{urlencode({'workspace': ws_id, 'path': rel_path})}"

    viewport = step.get("viewport") if isinstance(step.get("viewport"), dict) else None
    golden = (step.get("golden") or "").strip()
    min_match = float(step.get("min_match_pct", step.get("max_diff_pct", 90)))
    create_baseline = bool(step.get("create_baseline_if_missing"))

    if golden:
        golden_path = Path(golden)
        if not golden_path.is_absolute():
            golden_path = (Path(ws_root) / golden).resolve()
        baseline_path = str(golden_path)
        workspace_id = (step.get("workspace_id") or "").strip()
        body: dict[str, Any] = {
            "url": url,
            "baseline_path": baseline_path,
            "full_page": bool(step.get("full_page", True)),
        }
        if viewport:
            body["viewport"] = viewport
        if workspace_id:
            body["workspace_id"] = workspace_id
        else:
            body["workspace_root"] = ws_root
        code, data = _browser_post(ctx.base, "/api/browser/visual-diff", body)
        if code != 200:
            return False, data.get("error") or f"visual-diff HTTP {code}"
        if not data.get("baseline_exists") and create_baseline:
            accept_body = dict(body)
            accept_body["baseline_path"] = baseline_path
            if workspace_id:
                accept_body["workspace_id"] = workspace_id
            acode, adata = _browser_post(ctx.base, "/api/browser/accept-baseline", accept_body)
            if acode != 200:
                return False, adata.get("error") or f"accept-baseline HTTP {acode}"
            return True, f"created baseline {baseline_path}"
        match_pct = float(data.get("match_pct") or 0)
        if match_pct < min_match:
            return False, f"visual match {match_pct:.1f}% < {min_match}%"
        return True, f"visual match {match_pct:.1f}%"

    body = {"url": url, "full_page": bool(step.get("full_page", True))}
    if viewport:
        body["viewport"] = viewport
    code, data = _browser_post(ctx.base, "/api/browser/screenshot", body)
    if code != 200:
        return False, data.get("error") or f"screenshot HTTP {code}"
    b64 = (data.get("png_b64") or "").strip()
    if len(b64) < 100:
        return False, "screenshot payload too small"
    min_bytes = int(step.get("min_png_b64_bytes", 100))
    if len(b64) < min_bytes:
        return False, f"png b64 len {len(b64)} < {min_bytes}"
    return True, f"screenshot ok ({data.get('width')}x{data.get('height')})"


def _music_post(base: str, path: str, body: dict) -> tuple[int, dict]:
    url = f"{base.rstrip('/')}{path}"
    raw = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=raw, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=600) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return resp.status, data if isinstance(data, dict) else {}
    except urllib.error.HTTPError as exc:
        try:
            data = json.loads(exc.read().decode("utf-8"))
        except Exception:
            data = {"error": exc.reason or str(exc)}
        return exc.code, data if isinstance(data, dict) else {"error": str(data)}


def step_assert_music_generate(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    style = (step.get("style_tags") or "lo-fi").strip()
    duration = int(step.get("duration_sec") or 10)
    body = {
        "style_tags": style,
        "lyrics": step.get("lyrics") or "[Instrumental]",
        "duration_sec": duration,
        "instrumental": bool(step.get("instrumental", True)),
    }
    code, data = _music_post(ctx.base, "/api/music/generate", body)
    if code != 200:
        return False, data.get("error") or f"generate HTTP {code}"
    b64 = (data.get("data") or "").strip()
    min_bytes = int(step.get("min_wav_bytes", 100))
    try:
        import base64

        raw = base64.standard_b64decode(b64)
    except Exception as exc:
        return False, f"decode audio: {exc}"
    if len(raw) < min_bytes:
        return False, f"wav bytes {len(raw)} < {min_bytes}"
    ctx.last_music_path = data.get("path")
    return True, f"generated {len(raw)} bytes"


def step_assert_music_extract(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    audio_path = (step.get("audio_path") or ctx.last_music_path or "").strip()
    if not audio_path:
        if step.get("skip_if_unavailable"):
            return True, "extract skipped (no audio path)"
        return False, "audio_path required (run assert_music_generate first)"
    tracks = step.get("tracks") or ["vocals"]
    code, data = _music_post(ctx.base, "/api/music/extract", {"audio_path": audio_path, "tracks": tracks})
    if code == 409 and not step.get("requires_sft", True):
        return True, "extract skipped (turbo variant)"
    if code != 200:
        if step.get("skip_if_unavailable"):
            return True, f"extract skipped ({data.get('error') or code})"
        return False, data.get("error") or f"extract HTTP {code}"
    stems = data.get("stems") or []
    if not stems:
        return False, "no stems returned"
    return True, f"extracted {len(stems)} stem(s)"


def _arena_post(base: str, path: str, body: dict) -> tuple[int, dict]:
    url = f"{base.rstrip('/')}{path}"
    raw = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=raw, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return resp.status, data if isinstance(data, dict) else {}
    except urllib.error.HTTPError as exc:
        try:
            data = json.loads(exc.read().decode("utf-8"))
        except Exception:
            data = {"error": exc.reason or str(exc)}
        return exc.code, data if isinstance(data, dict) else {"error": str(data)}


def step_assert_arena_session(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    challenge = (step.get("challenge") or "connect4").strip()
    code, data = _arena_post(ctx.base, "/api/arena/sessions", {"challenge": challenge})
    if code != 200:
        if step.get("skip_if_unavailable"):
            return True, f"arena skipped ({data.get('error') or code})"
        return False, data.get("error") or f"create session HTTP {code}"
    session_id = (data.get("id") or "").strip()
    if not session_id:
        return False, "session missing id"
    moves = step.get("moves") or []
    for move in moves:
        body: dict = {"by": "scenario"}
        if challenge == "connect4":
            body["column"] = int(move)
        else:
            body["move"] = str(move)
        code, data = _arena_post(ctx.base, f"/api/arena/sessions/{session_id}/move", body)
        if code != 200:
            return False, data.get("error") or f"move HTTP {code}"
    state = data.get("state") if isinstance(data.get("state"), dict) else {}
    status = (state.get("status") or data.get("status") or "").strip()
    result = (state.get("result") or data.get("result") or "").strip()
    expected_status = (step.get("expected_status") or "finished").strip()
    expected_result = (step.get("expected_result") or "").strip()
    if expected_status and status != expected_status:
        return False, f"status {status!r} != {expected_status!r}"
    if expected_result and result != expected_result:
        return False, f"result {result!r} != {expected_result!r}"
    return True, f"session {session_id} status={status} result={result}"


def run_step(ctx: ScenarioContext, step: dict, label: str) -> bool:
    action = (step.get("action") or step.get("type") or "").strip()
    handlers = {
        "wait_phase": step_wait_phase,
        "wait_planning_recap": step_wait_planning_recap,
        "wait_discussion": step_wait_discussion,
        "assert_messages": step_assert_messages,
        "assert_collab": step_assert_collab,
        "assert_plan": step_assert_plan,
        "send": step_send,
        "approve_plan": step_approve_plan,
        "workspace_ack": step_workspace_ack,
        "wait_tasks": step_wait_tasks,
        "approve_file_changes": step_approve_file_changes,
        "assert_deliverable": step_assert_deliverable,
        "assert_files": step_assert_files,
        "assert_deliverable_stubs": step_assert_deliverable_stubs,
        "setup_pack": step_setup_pack,
        "assert_browser_screenshot": step_assert_browser_screenshot,
        "assert_cad_render": step_assert_cad_render,
        "assert_music_generate": step_assert_music_generate,
        "assert_music_extract": step_assert_music_extract,
        "assert_arena_session": step_assert_arena_session,
    }
    fn = handlers.get(action)
    if not fn:
        print(f"  FAIL [{label}]: unknown action {action!r}", file=sys.stderr)
        return False
    ok, detail = fn(ctx, step)
    mark = "✓" if ok else "✗"
    print(f"  {mark} [{label}] {action}: {detail}")
    if not ok:
        ctx.last_failure = detail
        dump_transcript(ctx)
        if action == "wait_discussion":
            required = step.get("required_agents") or ctx.scenario.get("required_agents") or []
            print(
                hub.discussion_diagnosis(
                    ctx.base,
                    ctx.collab_channel,
                    required_agents=required,
                    collab_id=ctx.collab_id,
                ),
                file=sys.stderr,
            )
    return ok


def run_setup_blocker(ctx: ScenarioContext, setup: dict, agents: str) -> bool:
    """Create an executing collab on blocker channel to test multi-collab isolation."""
    ch = setup.get("channel") or BLOCKER_CHANNEL
    if not hub.ensure_channel(ctx.base, ch):
        return False
    # Prefer distinct agents from the probe collab so the same specialist is not
    # busy in the executing blocker while the planning probe expects a reply.
    blocker_agents = (setup.get("agents") or "@BackendEngineer @SoftwareArchitect").strip()
    if not blocker_agents:
        blocker_agents = "@BackendEngineer @SoftwareArchitect"
    mini = {
        "collaborate": {
            "flags": setup.get("flags") or ["--rounds", "1", "--messages", "1"],
            "goal": setup.get("goal")
            or f"{blocker_agents} hold executing slot for isolation probe",
        },
        "channel": ch,
    }
    started = start_collaboration(ctx, ch, mini, blocker_agents)
    if not started:
        return False
    cid, cch = started
    blocker = {"id": cid, "channel": cch}
    ctx.extra_collabs.append(blocker)

    steps = setup.get("steps") or [
        {"action": "wait_phase", "phase": "reviewing", "timeout": 120},
        {"action": "wait_planning_recap", "timeout": 120},
        {"action": "approve_plan"},
        {"action": "workspace_ack"},
        {"action": "wait_phase", "phase": "executing", "timeout": 30},
    ]
    mini_ctx = ScenarioContext(ctx.base, mini, ctx.verbose)
    mini_ctx.collab_id = cid
    mini_ctx.collab_channel = cch
    for i, step in enumerate(steps, 1):
        if not run_step(mini_ctx, step, f"setup-{i}"):
            return False
    # Park the blocker: pause dispatch + skip open tasks so Ollama is free for the
    # planning probe, while the collab remains in executing for isolation.
    if not park_executing_blocker(ctx.base, cid, channel=cch):
        print(f"  WARN: could not park blocker {cid[:8]} (continuing anyway)", flush=True)
    else:
        # Let aborted generations release Ollama slots before the probe starts.
        time.sleep(2.0)
    print(f"  ✓ setup: blocker collab {cid[:8]} executing on {ch} ({blocker_agents})")
    return True


def park_executing_blocker(base: str, collab_id: str, *, channel: str = "") -> bool:
    """Pause task dispatch, skip open tasks, and abort in-flight gens on the blocker channel."""
    import urllib.parse

    cid = (collab_id or "").strip()
    if not cid:
        return False
    enc = urllib.parse.quote(cid, safe="")
    code, _ = hub.hub_request(base, "POST", f"/api/collaborations/{enc}/pause")
    if code not in (200, 201):
        return False
    collab = hub.fetch_collab(base, "", cid)
    if not collab:
        return True
    ok = True
    for task in collab.get("tasks") or []:
        if not isinstance(task, dict):
            continue
        status = (task.get("status") or "").strip().lower()
        if status in ("completed", "skipped"):
            continue
        tid = (task.get("id") or "").strip()
        if not tid:
            continue
        tenc = urllib.parse.quote(tid, safe="")
        tcode, _ = hub.hub_request(
            base, "POST", f"/api/collaborations/{enc}/tasks/{tenc}/skip"
        )
        if tcode not in (200, 201):
            ok = False
    # Cancel in-flight LLM work so the planning probe is not starved under
    # NJ_OLLAMA_MAX_CONCURRENCY (pause alone does not abort active streams).
    abort_ch = (channel or "").strip() or (collab.get("channel") or "").strip()
    if abort_ch:
        abort_ok, abort_detail = hub.abort_channel_agents(
            base, abort_ch, held_by="multi-collab-park"
        )
        if not abort_ok:
            print(f"  WARN: blocker abort failed: {abort_detail}", flush=True)
            ok = False
    return ok


def _solo_last_agent_chat(msgs: list[dict], agent: str, *, skip_status: bool = True) -> dict | None:
    pool = hub.chat_agent_messages(msgs)
    for msg in reversed(pool):
        if (msg.get("from") or {}).get("name") != agent:
            continue
        content = (msg.get("content") or "").strip()
        meta = msg.get("metadata") if isinstance(msg.get("metadata"), dict) else {}
        if skip_status and content and _solo_is_status_chat(content, meta):
            continue
        return msg
    return None


def _solo_is_status_chat(content: str, meta: dict | None = None) -> bool:
    if isinstance(meta, dict) and isinstance(meta.get("implementation_session_outcome"), dict):
        return True
    low = content.lower()
    markers = (
        "implementation session complete",
        "proposals submitted for approval",
        "verification skipped",
        "i submitted a file change proposal",
        "approve proposals to apply",
    )
    return any(m in low for m in markers)


def _solo_proposal_content_from_messages(
    msgs: list[dict], output_rel: str
) -> str | None:
    want = output_rel.replace("\\", "/").strip().lower()
    for msg in reversed(msgs):
        if (msg.get("type") or "") != "file_change":
            continue
        meta = msg.get("metadata") or {}
        proposal = meta.get("file_change_proposal") if isinstance(meta, dict) else None
        if not isinstance(proposal, dict):
            continue
        fp = (proposal.get("file_path") or proposal.get("FilePath") or "").replace("\\", "/")
        if fp.strip().lower() != want:
            continue
        body = (proposal.get("new_content") or proposal.get("NewContent") or "").strip()
        if body:
            return body
    return None


def _solo_content_has_bullets(content: str) -> bool:
    for raw in content.splitlines():
        line = raw.strip()
        if line and re.match(r"^(\d+\.|[-*])\s+", line):
            return True
    return False


def _write_solo_findings_file(workspace_root: str, output_rel: str, body: str) -> str | None:
    body = hub.sanitize_deliverable_body(body)
    if not body:
        return None
    if not body.endswith("\n"):
        body += "\n"
    dest = Path(workspace_root) / output_rel
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(body, encoding="utf-8")
    return str(dest)


def _materialize_solo_findings(
    msgs: list[dict],
    workspace_root: str,
    output_rel: str,
    min_bytes: int,
    *,
    agent: str | None = None,
) -> str | None:
    """Write solo output from agent chat when hub file-change approval misses."""
    proposal_body = _solo_proposal_content_from_messages(msgs, output_rel)
    if proposal_body and len(proposal_body.encode()) >= min_bytes:
        return _write_solo_findings_file(workspace_root, output_rel, proposal_body)

    if agent:
        last = _solo_last_agent_chat(msgs, agent)
    else:
        last = None
        for msg in reversed(hub.chat_agent_messages(msgs)):
            content = (msg.get("content") or "").strip()
            meta = msg.get("metadata") if isinstance(msg.get("metadata"), dict) else {}
            if content and not _solo_is_status_chat(content, meta):
                last = msg
                break
    if not last:
        return None
    content = (last.get("content") or "").strip()
    meta = last.get("metadata") if isinstance(last.get("metadata"), dict) else {}
    if not content or _solo_is_status_chat(content, meta):
        return None

    path_needle = Path(output_rel).parent.name or "parity-solo"
    has_fc = "[FILE_CHANGE]" in content
    has_bullets = _solo_content_has_bullets(content)

    if has_fc or has_bullets:
        written = hub.write_loose_file_change_from_messages(
            msgs,
            workspace_root,
            path_contains=path_needle,
            target_rel=output_rel,
            min_bytes=min_bytes,
        )
        if written:
            return written
        if has_fc:
            from lib.collab_hub import _extract_loose_file_changes  # noqa: PLC0415

            for _rel, body in _extract_loose_file_changes(content):
                body = hub.sanitize_deliverable_body(body)
                if len(body.encode()) >= min_bytes:
                    return _write_solo_findings_file(workspace_root, output_rel, body)
        if has_bullets:
            lines = [
                raw.strip()
                for raw in content.splitlines()
                if raw.strip() and re.match(r"^(\d+\.|[-*])\s+", raw.strip())
            ]
            if lines:
                body = "# Findings\n\n" + "\n".join(lines) + "\n"
                if len(body.encode()) >= min_bytes:
                    return _write_solo_findings_file(workspace_root, output_rel, body)
        return None

    if len(content.encode()) >= min_bytes:
        body = content if content.startswith("#") else f"# Findings\n\n{content}\n"
        if len(body.encode()) >= min_bytes:
            return _write_solo_findings_file(workspace_root, output_rel, body)
    return None


def run_solo_parity_leg(ctx: ScenarioContext, scenario: dict) -> tuple[bool, str]:
    """Run optional solo_leg. Returns (ok, detail) — detail is for flake-retry matching."""
    solo = scenario.get("solo_leg")
    if not isinstance(solo, dict):
        return True, ""

    ctx.workspace_root = ctx.resolve_workspace_root()
    if not ctx.workspace_root:
        print("  FAIL [solo]: no workspace root", file=sys.stderr)
        return False, "solo leg: no workspace root"

    channel = (solo.get("channel") or f"{ctx.channel}-solo").strip()
    solo_agents = solo.get("required_agents") or ["Assistant"]
    if isinstance(solo_agents, str):
        solo_agents = [solo_agents]
    solo_agent = str(solo_agents[0]).strip().lstrip("@")
    ok_join, failed = hub.ensure_channel_with_agents(
        ctx.base,
        channel,
        [str(a).strip().lstrip("@") for a in solo_agents if str(a).strip()],
        "Solo parity leg",
    )
    if not ok_join:
        print(f"  FAIL [solo]: could not join agents to {channel!r}: {', '.join(failed)}", file=sys.stderr)
        return False, f"solo leg: could not join agents: {', '.join(failed)}"

    output_rel = (solo.get("output_rel") or "collabs/parity-solo/findings.md").strip()
    message = (solo.get("message") or "").strip()
    if not message:
        print("  FAIL [solo]: empty message", file=sys.stderr)
        return False, "solo leg: empty message"

    solo_dir = Path(ctx.workspace_root) / Path(output_rel).parent
    solo_dir.mkdir(parents=True, exist_ok=True)

    print(f"  solo leg: channel={channel} output={output_rel}")
    # Abort any leftover implement-session streams from a prior attempt/suite.
    hub.abort_channel_agents(ctx.base, channel, held_by="scenario-solo-leg")
    hub.clear_channel_history(ctx.base, channel, max_retries=4)
    meta = ctx.build_send_metadata() or {}
    # Prefer a normal chat turn over a multi-step implementation session.
    # Forced implementation_session + local coder models often burns the full
    # reply_timeout without posting a counted chat/answer message.
    meta["conversation_mode"] = "chat"
    meta["implementation_session"] = False
    meta.setdefault("editor_mode", "ask")
    meta.setdefault("context_scope", "outline")
    code, _ = hub.send_message(ctx.base, channel, message, metadata=meta)
    if code != 200:
        print(f"  FAIL [solo]: send ({code})", file=sys.stderr)
        return False, f"solo leg: send failed ({code})"

    timeout = hub.parse_timeout(solo.get("timeout", "300s"))
    reply_timeout = hub.parse_timeout(solo.get("reply_timeout", solo.get("timeout", "240s")))
    reply_timeout = min(reply_timeout, timeout)
    min_bytes = int(solo.get("min_bytes", 40))
    allow_fallback = bool(solo.get("allow_discussion_fallback", True))
    baseline = hub.count_chat_agent_messages(
        hub.list_messages(ctx.base, channel, 200), from_agent=solo_agent
    )
    failure_retried = False
    ok_reply = False
    reply_detail = ""
    while True:
        ok_reply, reply_detail = hub.wait_chat_reply(
            ctx.base,
            channel,
            from_agent=solo_agent,
            baseline_count=baseline,
            timeout=reply_timeout,
        )
        if ok_reply:
            break
        msgs = hub.list_messages(ctx.base, channel, 200)
        last = _solo_last_agent_chat(msgs, solo_agent)
        if (
            not failure_retried
            and last
            and hub.is_agent_failure_message(last)
        ):
            failure_retried = True
            retry_msg = f"@{solo_agents[0]} Please try again — {message}"
            ctx.log(f"  solo leg: failure reply detected; re-sending")
            hub.abort_channel_agents(ctx.base, channel, held_by="scenario-solo-leg")
            code, _ = hub.send_message(ctx.base, channel, retry_msg, metadata=meta)
            if code != 200:
                print(f"  FAIL [solo]: retry send ({code})", file=sys.stderr)
                return False, f"solo leg: retry send failed ({code})"
            baseline = hub.count_chat_agent_messages(
                hub.list_messages(ctx.base, channel, 200), from_agent=solo_agent
            )
            continue
        print(f"  FAIL [solo]: no {solo_agent} reply ({reply_detail})", file=sys.stderr)
        hub.abort_channel_agents(ctx.base, channel, held_by="scenario-solo-leg")
        return False, f"solo leg: no {solo_agent} reply ({reply_detail})"
    ctx.log(f"  solo leg: {reply_detail}")
    path_needle = Path(output_rel).parent.name or "parity-solo"
    approve_timeout = hub.parse_timeout(solo.get("approve_timeout", "120s"))
    hub.wait_and_approve_file_changes(
        ctx.base,
        channel,
        path_contains=path_needle,
        min_approved=0,
        timeout=min(20.0, approve_timeout),
    )
    full = Path(ctx.workspace_root) / output_rel
    if allow_fallback and (not full.is_file() or full.stat().st_size < min_bytes):
        msgs = hub.list_messages(ctx.base, channel, 200)
        written = _materialize_solo_findings(
            msgs, ctx.workspace_root, output_rel, min_bytes, agent=solo_agent
        )
        if written and ctx.verbose:
            print(f"  solo leg: materialized findings {written}")
    deadline = time.time() + timeout
    while time.time() < deadline:
        if not full.is_file() or full.stat().st_size < min_bytes:
            remaining = max(0.5, deadline - time.time())
            approve_poll = min(20.0, approve_timeout, remaining)
            for path_needle in (path_needle, output_rel.replace("\\", "/"), ""):
                n, _ids = hub.wait_and_approve_file_changes(
                    ctx.base,
                    channel,
                    path_contains=path_needle,
                    min_approved=1,
                    timeout=approve_poll,
                )
                if n > 0:
                    if ctx.verbose:
                        print(f"  solo leg: approved {n} file change(s) path~={path_needle!r}")
                    break
            if allow_fallback and (not full.is_file() or full.stat().st_size < min_bytes):
                msgs = hub.list_messages(ctx.base, channel, 200)
                written = _materialize_solo_findings(
                    msgs, ctx.workspace_root, output_rel, min_bytes, agent=solo_agent
                )
                if written and ctx.verbose:
                    print(f"  solo leg: materialized findings {written}")
            time.sleep(hub.POLL_INTERVAL)
        if full.is_file() and full.stat().st_size >= min_bytes:
            body = full.read_text(encoding="utf-8", errors="replace")
            ok, detail = check_text_patterns(
                body,
                any_match=solo.get("any_match"),
                none_match=solo.get("none_match"),
                label="solo file",
            )
            if ok:
                print(f"  ✓ solo leg: {full} ({detail})")
                return True, ""
            print(f"  FAIL [solo]: {detail}", file=sys.stderr)
            return False, f"solo leg: {detail}"
        time.sleep(hub.POLL_INTERVAL)

    print(f"  FAIL [solo]: timeout waiting for {full}", file=sys.stderr)
    hub.abort_channel_agents(ctx.base, channel, held_by="scenario-solo-leg")
    return False, f"solo leg: timeout waiting for {output_rel}"


def cleanup_scenario_workspace(ctx: ScenarioContext) -> None:
    if not ctx.workspace_root or not ctx.collab_id:
        return
    collab_dir = Path(ctx.workspace_root) / "collabs" / ctx.collab_id
    if collab_dir.is_dir():
        shutil.rmtree(collab_dir, ignore_errors=True)


def cleanup_scenario_collabs(base: str, ctx: ScenarioContext, *, keep: bool) -> None:
    if keep:
        if ctx.collab_id:
            print(f"  --keep: collab {ctx.collab_id[:8]} left active")
        return
    if ctx.collab_id and ctx.collab_channel:
        hub.send_message(base, ctx.collab_channel, f"/cancel-plan {ctx.collab_id[:8]}")
        hub.wait_phase(base, ctx.collab_channel, ctx.collab_id, "cancelled", 10)
    for extra in ctx.extra_collabs:
        hub.cancel_collab(base, extra)
    cleanup_scenario_workspace(ctx)
    print("  ✓ cleanup: cancelled and removed workspace artifacts")


def ensure_hub_ready(base: str, context: str) -> bool:
    if ensure_hub_with_recovery(ROOT, base, context=context, env=release_prep_env(ROOT)):
        return True
    print("  FAIL: hub not healthy (recovery exhausted after 3 attempts)", file=sys.stderr)
    return False


def _restart_hub_between_scenarios_enabled(*, core: bool, soft: bool = False) -> bool:
    raw = os.environ.get("NJ_RESTART_HUB_BETWEEN_SCENARIOS", "").strip().lower()
    if raw in ("0", "false", "no"):
        return False
    if raw in ("1", "true", "yes"):
        return True
    # Edge suite defaults to soft reset (no full Ollama re-warm) unless overridden.
    if soft:
        return False
    # Default on for --core and --all to shed in-process collab state.
    return True


def soft_reset_between_scenarios(base: str, *, label: str) -> bool:
    """Cancel leftover collabs and clear pending proposals without restarting the hub."""
    from lib.hub_cleanup import clear_pending_file_changes

    print(f"\n>>> Soft reset between scenarios ({label})...")
    hub.free_scenario_capacity(base, DEFAULT_CHANNEL)
    hub.free_scenario_capacity(base, BLOCKER_CHANNEL)
    cleared = clear_pending_file_changes(base)
    if cleared:
        print(f"  cleared {cleared} pending file change(s)")
    if hub.check_health(base) is None:
        print("  FAIL: hub not healthy after soft reset", file=sys.stderr)
        return False
    print("  OK: soft reset complete")
    return True


def restart_hub_between_scenarios(
    base: str, *, label: str, core: bool, soft: bool = False
) -> bool:
    """Clear in-process agent/collab state between batch scenarios (reduces Ollama contention)."""
    if not _restart_hub_between_scenarios_enabled(core=core, soft=soft):
        if soft or os.environ.get("NJ_RESTART_HUB_BETWEEN_SCENARIOS", "").strip().lower() in (
            "0",
            "false",
            "no",
        ):
            return soft_reset_between_scenarios(base, label=label)
        return True
    from lib.regression_boot import restart_hub_for_live_run

    print(f"\n>>> Hub restart between scenarios ({label})...")
    return restart_hub_for_live_run(ROOT, base, label=label)


def run_scenario(
    base: str,
    name: str,
    *,
    profile: str | None = None,
    agents_override: str | None = None,
    verbose: bool = False,
    keep: bool = False,
) -> bool:
    from lib.scenario_flake_retry import maybe_retry_after_failure

    last_detail = ""
    max_attempts = 3 if name in {
        "make-me-a-website",
        "collaboration-station-website",
        "collaboration-station-website-sa",
        "solo-vs-collab-parity",
        "multi-collab-isolation",
        "plan-findings-task-regression",
    } else 2
    for attempt in range(1, max_attempts + 1):
        ok, last_detail = _run_scenario_once(
            base,
            name,
            profile=profile,
            agents_override=agents_override,
            verbose=verbose,
            keep=keep,
        )
        if ok:
            return True
        if not maybe_retry_after_failure(
            base,
            name,
            last_detail or f"collab scenario {name} failed",
            attempt,
            max_attempts=max_attempts,
        ):
            break
    return False


def _run_scenario_once(
    base: str,
    name: str,
    *,
    profile: str | None = None,
    agents_override: str | None = None,
    verbose: bool = False,
    keep: bool = False,
) -> tuple[bool, str]:
    scenario = load_scenario(name)
    from lib.fixture_baseline import reset_fixture_baseline
    from lib.regression_collab import apply_core_scenario_defaults, is_collab_core_scenario

    reset_fixture_baseline(scenario, root=ROOT)
    if is_collab_core_scenario(name):
        scenario = apply_core_scenario_defaults(scenario)
    if scenario_requires_gemini(scenario) and _gemini_probe_ok is False:
        print(f"=== FAIL: {name} ===", file=sys.stderr)
        print(f"  Gemini preflight failed: {_gemini_probe_detail}", file=sys.stderr)
        return False, "gemini preflight failed"
    if scenario_requires_claude(scenario) and _claude_probe_ok is False:
        print(f"=== FAIL: {name} ===", file=sys.stderr)
        print(f"  Claude preflight failed: {_claude_probe_detail}", file=sys.stderr)
        return False, "claude preflight failed"
    ctx = ScenarioContext(base, scenario, verbose)
    channel = ctx.channel
    agents = scenario.get("agents") or hub.resolve_agents(profile, agents_override)

    print(f"\n=== scenario: {name} ===")
    print(f"  hub={base} channel={channel} agents={agents}")

    try:
        if not ensure_hub_ready(base, f"collab:{name}"):
            return False, "hub not healthy"

        if not hub.ensure_channel(base, channel):
            print(f"  FAIL: could not ensure channel {channel!r}", file=sys.stderr)
            return False, f"could not ensure channel {channel!r}"

        collab_roster = hub.collaborate_agent_names(scenario, agents)
        if collab_roster:
            ok_join, failed_join = hub.ensure_channel_with_agents(
                base, channel, collab_roster, "Collab scenario roster"
            )
            if not ok_join:
                print(
                    f"  FAIL: could not join agents to {channel!r}: {', '.join(failed_join)}",
                    file=sys.stderr,
                )
                return False, f"could not join agents: {', '.join(failed_join)}"

        required = scenario.get("required_agents") or []
        if required:
            ok_agents, missing = hub.verify_agents_online(base, required)
            if not ok_agents:
                if scenario.get("optional"):
                    print(f"  SKIP (optional): required agents offline: {', '.join(missing)}")
                    return True, "optional skip"
                print(f"  FAIL: required agents offline: {', '.join(missing)}", file=sys.stderr)
                print("  Start hub with CLI agents or set NJ_COLLAB_SCENARIO_AGENTS", file=sys.stderr)
                return False, f"required agents offline: {', '.join(missing)}"

        collab_agents = hub.collaborate_agent_names(scenario, agents)
        if collab_agents:
            ok_collab, missing_collab = hub.verify_agents_online(base, collab_agents)
            if not ok_collab:
                if scenario.get("optional"):
                    print(f"  SKIP (optional): collaborate agents offline: {', '.join(missing_collab)}")
                    return True, "optional skip"
                print(
                    f"  FAIL: collaborate agents offline: {', '.join(missing_collab)}",
                    file=sys.stderr,
                )
                print(f"  collaborate roster: {', '.join(collab_agents)}", file=sys.stderr)
                return False, f"collaborate agents offline: {', '.join(missing_collab)}"

        if not hub.free_scenario_capacity(base, channel):
            print("  FAIL: collab capacity full", file=sys.stderr)
            return False, "collab capacity full"

        ok_solo, solo_detail = run_solo_parity_leg(ctx, scenario)
        if not ok_solo:
            print(f"=== FAIL: {name} (solo leg) ===\n", file=sys.stderr)
            return False, solo_detail or f"{name} solo leg failed"

        setup = scenario.get("setup")
        if isinstance(setup, dict) and setup.get("action") == "executing_blocker":
            if not run_setup_blocker(ctx, setup, agents):
                return False, "executing_blocker setup failed"

        ctx.workspace_root = ctx.resolve_workspace_root()

        if not scenario.get("collaborate"):
            all_ok = True
            steps = list(scenario.get("steps") or []) + expand_deliverable_steps(scenario)
            for i, step in enumerate(steps, 1):
                if not run_step(ctx, step, f"{i}"):
                    all_ok = False
                    break
            if all_ok:
                print(f"=== PASS: {name} ===\n")
                return True, ""
            print(f"=== FAIL: {name} ===\n", file=sys.stderr)
            return False, ctx.last_failure or f"scenario {name} failed"

        started = start_collaboration(ctx, channel, scenario, agents)
        if not started:
            return False, "start_collaboration failed"
        ctx.collab_id, ctx.collab_channel = started
        print(f"  started collab {ctx.collab_id[:8]} → {ctx.collab_channel}")

        all_ok = True
        steps = list(scenario.get("steps") or []) + expand_deliverable_steps(scenario)
        for i, step in enumerate(steps, 1):
            if not run_step(ctx, step, f"{i}"):
                all_ok = False
                break

        if all_ok:
            print(f"=== PASS: {name} ===\n")
            return True, ""
        print(f"=== FAIL: {name} ===\n", file=sys.stderr)
        return False, ctx.last_failure or f"scenario {name} failed"
    finally:
        cleanup = scenario.get("cleanup", "cancel")
        if cleanup == "cancel":
            cleanup_scenario_collabs(base, ctx, keep=keep)
        elif ctx.collab_id and ctx.collab_channel:
            # Scenarios with cleanup "none" still free channel capacity for later runs.
            hub.free_scenario_capacity(base, ctx.channel)


def main() -> int:
    p = argparse.ArgumentParser(description="Run collaboration scenarios (live hub)")
    p.add_argument("--list", action="store_true", help="List scenario names")
    p.add_argument("--scenario", help="Scenario name (without .json)")
    p.add_argument("--all", action="store_true", help="Run all scenarios")
    p.add_argument(
        "--core",
        action="store_true",
        help="Run collab-core scenarios only (participation/planning; default hub restart between)",
    )
    p.add_argument(
        "--suite",
        choices=["edge"],
        default=None,
        help="Named scenario suite (edge = collab-scenario-regression list; soft reset between)",
    )
    p.add_argument("--hub", default=hub.DEFAULT_HUB)
    p.add_argument("--profile", choices=["fast", "realistic", "core"], default=None)
    p.add_argument("--agents", default=None, help="Override agent mentions")
    p.add_argument("--verbose", "-v", action="store_true")
    p.add_argument("--keep", action="store_true", help="Do not cancel collab after run")
    p.add_argument(
        "--require-gemini",
        action="store_true",
        help="Fail when Gemini API probe fails (opt-in; collab scenarios use Ollama agents only)",
    )
    p.add_argument(
        "--require-claude",
        action="store_true",
        help="Fail when Claude Code CLI probe fails (for @Claude scenarios)",
    )
    p.add_argument("--pack-dir", help="Pack repo root; scenarios read from <pack-dir>/scenarios/collab")
    args = p.parse_args()

    global SCENARIOS_DIR
    if args.pack_dir:
        SCENARIOS_DIR = Path(args.pack_dir).resolve() / "scenarios" / "collab"
    elif os.environ.get("NJ_PACK_SCENARIOS_DIR", "").strip():
        SCENARIOS_DIR = Path(os.environ["NJ_PACK_SCENARIOS_DIR"]).resolve()

    if args.list:
        for name in list_scenarios():
            print(name)
        return 0

    modes = sum(bool(x) for x in (args.all, args.core, args.suite, args.scenario))
    if modes > 1:
        p.error("use only one of --all, --core, --suite, or --scenario")

    base = args.hub.rstrip("/")
    suite_soft = False
    if args.core:
        from lib.collab_core_scenarios import collab_core_scenarios

        scenario_names = collab_core_scenarios()
    elif args.suite == "edge":
        from lib.collab_edge_scenarios import collab_edge_scenarios

        scenario_names = collab_edge_scenarios()
        suite_soft = True
    elif args.all:
        scenario_names = list_scenarios()
    else:
        scenario_names = [args.scenario] if args.scenario else []
    if scenario_names and not prepare_gemini_for_collab(
        scenario_names=scenario_names,
        require=args.require_gemini,
    ):
        return 1
    if scenario_names and not prepare_claude_for_collab(
        scenario_names=scenario_names,
        require=args.require_claude,
    ):
        return 1
    if args.core:
        os.environ["NJ_REQUIRE_FULL_BOOT"] = "1"
        os.environ.setdefault("NJ_REGRESSION_SLIM_ROSTER", "1")
        os.environ.setdefault("NJ_SCENARIO_PROFILE", "core")
        os.environ.pop("SKIP_BOOT", None)
        os.environ.pop("NJ_BOOT_DONE", None)
    if args.suite == "edge":
        os.environ["NJ_REQUIRE_FULL_BOOT"] = "1"
        os.environ.setdefault("NJ_REGRESSION_SLIM_ROSTER", "1")
        os.environ.setdefault("NJ_REGRESSION_COLLAB_EDGE", "1")
        os.environ.setdefault("NJ_OLLAMA_MAX_CONCURRENCY", "2")
        os.environ.pop("SKIP_BOOT", None)
        os.environ.pop("NJ_BOOT_DONE", None)
    if not maybe_boot_regression(base, root=ROOT, label="collab-scenarios"):
        return 1
    if args.all or args.core or args.suite:
        if args.core:
            label = "collab-scenarios-core"
        elif args.suite == "edge":
            label = "collab-scenarios-edge"
        else:
            label = "collab-scenarios preflight"
        preflight_regression_run(ROOT, base, label=label)
        names = scenario_names
        if not names:
            print("No scenarios found", file=sys.stderr)
            return 1
        failed: list[str] = []
        for i, n in enumerate(names):
            if i > 0 and not restart_hub_between_scenarios(
                base,
                label=f"after {names[i - 1]}",
                core=args.core,
                soft=suite_soft,
            ):
                print("FAIL: hub reset between scenarios failed", file=sys.stderr)
                return 1
            if not run_scenario(
                base,
                n,
                profile=args.profile,
                agents_override=args.agents,
                verbose=args.verbose,
                keep=args.keep,
            ):
                failed.append(n)
        return 1 if failed else 0

    if not args.scenario:
        p.error("specify --scenario <name>, --suite edge, --core, or --all")
    ok = run_scenario(
        base,
        args.scenario,
        profile=args.profile,
        agents_override=args.agents,
        verbose=args.verbose,
        keep=args.keep,
    )
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
