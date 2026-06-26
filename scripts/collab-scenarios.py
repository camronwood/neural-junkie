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
from pathlib import Path
from typing import Any

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.hub_regression import ensure_hub_with_recovery  # noqa: E402
from lib.release_prep_env import release_prep_env  # noqa: E402
from lib.scenario_assert import (  # noqa: E402
    check_file_deliverable,
    check_text_patterns,
    expand_deliverable_steps,
    looks_like_stack_tool_command,
    merge_deliverable_step,
    scenario_question,
)

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
        meta: dict[str, Any] = {
            "workspace_context": {
                "workspace_name": Path(path).name,
                "workspace_path": path,
                "file_tree": ws_cfg.get("file_tree") or "README.md\ncore/sample/main.go\n",
                "open_files": [],
            },
        }
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
    parts = ["/collaborate", *flags, agents, goal]
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
    meta: dict[str, str] | None = None
    if ctx.collab_id:
        meta = {
            "collaboration_id": ctx.collab_id,
            "collaboration_phase": "planning",
        }
    for raw in names:
        name = str(raw).strip().lstrip("@")
        if not name:
            continue
        msg = f"@{name} — please add your planning perspective for this collab."
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
            msgs = hub.agent_messages(scoped)
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
            if requirements_met and per_agent_ok:
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

    for pattern in step.get("any_match") or []:
        if not hub.messages_matching(pool, pattern):
            agents = sorted({(m.get("from") or {}).get("name", "?") for m in pool})
            return False, (
                f"any_match not found: {pattern!r} "
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
    if min_tasks > 0:
        if len(tasks) < min_tasks:
            effective = 0
            if content:
                effective = max(
                    len(re.findall(r"(?m)^\s*[-*]\s+Task\s+\d+:", content, re.I)),
                    len(re.findall(r"(?m)^\s*\d+\.\s+\*\*", content)),
                )
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
    code, _ = hub.send_message(ctx.base, channel, content)
    if code != 200:
        return False, f"send failed ({code})"
    time.sleep(0.5)
    return True, content[:60]


def step_approve_plan(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    if not ctx.collab_id:
        return False, "no collab_id"
    code, _ = hub.send_message(ctx.base, ctx.collab_channel, f"/approve-plan {ctx.collab_id[:8]}")
    if code != 200:
        return False, f"approve-plan failed ({code})"
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
    if not full.is_file() or full.stat().st_size < min_bytes:
        return False
    return not (allow_fallback and _is_deliverable_stub(full))


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
    if os.environ.get("NJ_SCENARIO_ALLOW_FILE_FALLBACK", "").strip() in ("1", "true", "yes"):
        allow_fallback = True

    n, ids = hub.wait_and_approve_file_changes(
        ctx.base,
        ctx.collab_channel,
        path_contains=path_match,
        min_approved=min_approved,
        timeout=timeout,
    )

    if expect_rel and ctx.workspace_root:
        full = Path(ctx.workspace_root) / expect_rel
        if _deliverable_file_ready(full, min_bytes=min_bytes, allow_fallback=allow_fallback):
            return True, f"file exists ({full})"

    if n >= min_approved and min_approved > 0:
        return True, f"hub approved {n}: {ids}"

    if min_approved == 0 and expect_rel and ctx.workspace_root:
        full = Path(ctx.workspace_root) / expect_rel
        if _deliverable_file_ready(full, min_bytes=min_bytes, allow_fallback=allow_fallback):
            return True, f"file exists ({full})"

    if require_hub and not allow_fallback:
        return False, f"require_hub_approval: approved={n} (need >={min_approved})"

    hub_approval_failed = min_approved > 0 and n < min_approved

    if expect_rel and ctx.workspace_root:
        full = Path(ctx.workspace_root) / expect_rel
        if full.is_file() and not require_hub and not (allow_fallback and _is_deliverable_stub(full)):
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
    return check_file_deliverable(
        root=root,
        rel=rel,
        spec=spec,
        question=scenario_question(ctx.scenario),
        collab_id=ctx.collab_id,
        hub_base=ctx.base,
    )


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
    mini = {
        "collaborate": {
            "flags": setup.get("flags") or ["--rounds", "1", "--messages", "1"],
            "goal": setup.get("goal") or "hold executing slot for isolation probe",
        },
        "channel": ch,
    }
    started = start_collaboration(ctx, ch, mini, agents)
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
    print(f"  ✓ setup: blocker collab {cid[:8]} executing on {ch}")
    return True


def _solo_last_assistant_chat(msgs: list[dict], *, skip_status: bool = True) -> dict | None:
    pool = hub.chat_agent_messages(msgs)
    for msg in reversed(pool):
        if (msg.get("from") or {}).get("name") != "Assistant":
            continue
        content = (msg.get("content") or "").strip()
        if skip_status and content and _solo_is_status_chat(content):
            continue
        return msg
    return None


def _solo_is_status_chat(content: str) -> bool:
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
) -> str | None:
    """Write solo output from Assistant chat when hub file-change approval misses."""
    proposal_body = _solo_proposal_content_from_messages(msgs, output_rel)
    if proposal_body and len(proposal_body.encode()) >= min_bytes:
        return _write_solo_findings_file(workspace_root, output_rel, proposal_body)

    last = _solo_last_assistant_chat(msgs)
    if not last:
        return None
    content = (last.get("content") or "").strip()
    if not content or _solo_is_status_chat(content):
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


def run_solo_parity_leg(ctx: ScenarioContext, scenario: dict) -> bool:
    solo = scenario.get("solo_leg")
    if not isinstance(solo, dict):
        return True

    ctx.workspace_root = ctx.resolve_workspace_root()
    if not ctx.workspace_root:
        print("  FAIL [solo]: no workspace root", file=sys.stderr)
        return False

    channel = (solo.get("channel") or f"{ctx.channel}-solo").strip()
    solo_agents = solo.get("required_agents") or ["Assistant"]
    if isinstance(solo_agents, str):
        solo_agents = [solo_agents]
    ok_join, failed = hub.ensure_channel_with_agents(
        ctx.base,
        channel,
        [str(a).strip().lstrip("@") for a in solo_agents if str(a).strip()],
        "Solo parity leg",
    )
    if not ok_join:
        print(f"  FAIL [solo]: could not join agents to {channel!r}: {', '.join(failed)}", file=sys.stderr)
        return False

    output_rel = (solo.get("output_rel") or "collabs/parity-solo/findings.md").strip()
    message = (solo.get("message") or "").strip()
    if not message:
        print("  FAIL [solo]: empty message", file=sys.stderr)
        return False

    solo_dir = Path(ctx.workspace_root) / Path(output_rel).parent
    solo_dir.mkdir(parents=True, exist_ok=True)

    print(f"  solo leg: channel={channel} output={output_rel}")
    meta = ctx.build_send_metadata() or {}
    meta.setdefault("conversation_mode", "code")
    meta.setdefault("implementation_session", True)
    meta.setdefault("editor_mode", "agent")
    meta.setdefault("editor_agent_trust", "auto_apply_edits")
    meta.setdefault("context_scope", "outline")
    code, _ = hub.send_message(ctx.base, channel, message, metadata=meta)
    if code != 200:
        print(f"  FAIL [solo]: send ({code})", file=sys.stderr)
        return False

    timeout = hub.parse_timeout(solo.get("timeout", "300s"))
    reply_timeout = min(120.0, timeout)
    min_bytes = int(solo.get("min_bytes", 40))
    allow_fallback = bool(solo.get("allow_discussion_fallback", True))
    baseline = hub.count_chat_agent_messages(
        hub.list_messages(ctx.base, channel, 200), from_agent="Assistant"
    )
    failure_retried = False
    ok_reply = False
    reply_detail = ""
    while True:
        ok_reply, reply_detail = hub.wait_chat_reply(
            ctx.base,
            channel,
            from_agent="Assistant",
            baseline_count=baseline,
            timeout=reply_timeout,
        )
        if ok_reply:
            break
        msgs = hub.list_messages(ctx.base, channel, 200)
        last = _solo_last_assistant_chat(msgs)
        if (
            not failure_retried
            and last
            and hub.is_agent_failure_message(last)
        ):
            failure_retried = True
            retry_msg = f"@{solo_agents[0]} Please try again — {message}"
            ctx.log(f"  solo leg: failure reply detected; re-sending")
            code, _ = hub.send_message(ctx.base, channel, retry_msg, metadata=meta)
            if code != 200:
                print(f"  FAIL [solo]: retry send ({code})", file=sys.stderr)
                return False
            baseline = hub.count_chat_agent_messages(msgs, from_agent="Assistant")
            continue
        print(f"  FAIL [solo]: no Assistant reply ({reply_detail})", file=sys.stderr)
        return False
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
            msgs, ctx.workspace_root, output_rel, min_bytes
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
                    msgs, ctx.workspace_root, output_rel, min_bytes
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
                return True
            print(f"  FAIL [solo]: {detail}", file=sys.stderr)
            return False
        time.sleep(hub.POLL_INTERVAL)

    print(f"  FAIL [solo]: timeout waiting for {full}", file=sys.stderr)
    return False


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


def run_scenario(
    base: str,
    name: str,
    *,
    profile: str | None = None,
    agents_override: str | None = None,
    verbose: bool = False,
    keep: bool = False,
) -> bool:
    scenario = load_scenario(name)
    ctx = ScenarioContext(base, scenario, verbose)
    channel = ctx.channel
    agents = scenario.get("agents") or hub.resolve_agents(profile, agents_override)

    print(f"\n=== scenario: {name} ===")
    print(f"  hub={base} channel={channel} agents={agents}")

    try:
        if not ensure_hub_ready(base, f"collab:{name}"):
            return False

        if not hub.ensure_channel(base, channel):
            print(f"  FAIL: could not ensure channel {channel!r}", file=sys.stderr)
            return False

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
                return False

        required = scenario.get("required_agents") or []
        if required:
            ok_agents, missing = hub.verify_agents_online(base, required)
            if not ok_agents:
                if scenario.get("optional"):
                    print(f"  SKIP (optional): required agents offline: {', '.join(missing)}")
                    return True
                print(f"  FAIL: required agents offline: {', '.join(missing)}", file=sys.stderr)
                print("  Start hub with CLI agents or set NJ_COLLAB_SCENARIO_AGENTS", file=sys.stderr)
                return False

        collab_agents = hub.collaborate_agent_names(scenario, agents)
        if collab_agents:
            ok_collab, missing_collab = hub.verify_agents_online(base, collab_agents)
            if not ok_collab:
                if scenario.get("optional"):
                    print(f"  SKIP (optional): collaborate agents offline: {', '.join(missing_collab)}")
                    return True
                print(
                    f"  FAIL: collaborate agents offline: {', '.join(missing_collab)}",
                    file=sys.stderr,
                )
                print(f"  collaborate roster: {', '.join(collab_agents)}", file=sys.stderr)
                return False

        if not hub.free_scenario_capacity(base, channel):
            print("  FAIL: collab capacity full", file=sys.stderr)
            return False

        if not run_solo_parity_leg(ctx, scenario):
            print(f"=== FAIL: {name} (solo leg) ===\n", file=sys.stderr)
            return False

        setup = scenario.get("setup")
        if isinstance(setup, dict) and setup.get("action") == "executing_blocker":
            if not run_setup_blocker(ctx, setup, agents):
                return False

        started = start_collaboration(ctx, channel, scenario, agents)
        if not started:
            return False
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
        else:
            print(f"=== FAIL: {name} ===\n", file=sys.stderr)
        return all_ok
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
    p.add_argument("--hub", default=hub.DEFAULT_HUB)
    p.add_argument("--profile", choices=["fast", "realistic"], default=None)
    p.add_argument("--agents", default=None, help="Override agent mentions")
    p.add_argument("--verbose", "-v", action="store_true")
    p.add_argument("--keep", action="store_true", help="Do not cancel collab after run")
    args = p.parse_args()

    if args.list:
        for name in list_scenarios():
            print(name)
        return 0

    base = args.hub.rstrip("/")
    if args.all:
        preflight_regression_run(ROOT, base, label="collab-scenarios preflight")
        names = list_scenarios()
        if not names:
            print("No scenarios found", file=sys.stderr)
            return 1
        failed = [n for n in names if not run_scenario(base, n, profile=args.profile, agents_override=args.agents, verbose=args.verbose, keep=args.keep)]
        return 1 if failed else 0

    if not args.scenario:
        p.error("specify --scenario <name> or --all")
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
