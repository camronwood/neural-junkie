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
import sys
import time
from pathlib import Path
from typing import Any

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.scenario_assert import check_text_patterns  # noqa: E402

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
        ws = scenario_workspace(self.scenario)
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
    repo = ""
    if ws:
        if ws.get("path"):
            p = Path(ws["path"])
            if not p.is_absolute():
                p = ROOT / p
            if p.is_dir():
                repo = str(p.resolve())
        elif ws.get("path_env"):
            raw = os.environ.get((ws.get("path_env") or "").strip(), "").strip()
            if raw:
                p = Path(raw)
                if p.is_dir():
                    repo = str(p.resolve())
    if repo and "--repo" not in flags:
        flags = ["--repo", repo, *flags]
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


def step_wait_discussion(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    timeout = hub.parse_timeout(step.get("timeout", 90))
    min_total = int(step.get("min_total", 1))
    min_per = int(step.get("min_per_agent", 0))
    max_total = int(step.get("max_total", 0))
    required = step.get("required_agents") or ctx.scenario.get("required_agents") or []
    deadline = time.time() + timeout
    while time.time() < deadline:
        msgs = hub.agent_messages(hub.list_messages(ctx.base, ctx.collab_channel, 200))
        counts = hub.count_by_agent(msgs)
        total = len(msgs)
        if _discussion_requirements_met(
            counts, total, min_total=min_total, min_per=min_per, required_agents=required
        ):
            return True, f"messages total={total} by_agent={counts}"
        if max_total > 0 and total > max_total:
            diag = hub.discussion_diagnosis(
                ctx.base, ctx.collab_channel, required_agents=required
            )
            return False, f"too many messages ({total} > {max_total})\n{diag}"
        time.sleep(hub.POLL_INTERVAL)
    diag = hub.discussion_diagnosis(ctx.base, ctx.collab_channel, required_agents=required)
    counts = hub.count_by_agent(hub.agent_messages(hub.list_messages(ctx.base, ctx.collab_channel, 200)))
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

    return True, "message assertions ok"


def step_assert_collab(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    c = hub.fetch_collab(ctx.base, ctx.collab_channel, ctx.collab_id)
    if not c:
        return False, "collaboration snapshot missing"

    if "phase" in step:
        want = step["phase"]
        if c.get("phase") != want:
            return False, f"phase={c.get('phase')!r} want {want!r}"

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


def step_approve_file_changes(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    if not ctx.collab_channel:
        return False, "no collab_channel"
    timeout = hub.parse_timeout(step.get("timeout", 60))
    path_match = (step.get("path_match") or step.get("path_contains") or "").strip()
    min_approved = int(step.get("min_approved", 1))
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
        if full.is_file() and full.stat().st_size >= int(step.get("min_bytes", 20)):
            return True, f"file exists ({full})"

    if n >= min_approved and min_approved > 0:
        return True, f"hub approved {n}: {ids}"

    if require_hub and not allow_fallback:
        return False, f"require_hub_approval: approved={n} (need >={min_approved})"

    if expect_rel and ctx.workspace_root:
        full = Path(ctx.workspace_root) / expect_rel
        if full.is_file() and not require_hub:
            return True, f"file already exists ({full})"

    if allow_fallback and ctx.workspace_root:
        msgs = hub.list_messages(ctx.base, ctx.collab_channel, 200)
        min_bytes = int(step.get("min_bytes", 20))
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

    return False, f"no file change approved (pending={n}, ids={ids})"


def step_assert_files(ctx: ScenarioContext, step: dict) -> tuple[bool, str]:
    rel = (step.get("path") or "").strip()
    if not rel:
        return False, "assert_files: path required"
    rel = rel.replace("<collab-id>", ctx.collab_id)
    root = step.get("root") or ctx.workspace_root
    if not root:
        return False, "assert_files: no workspace root"
    full = Path(root) / rel
    if step.get("must_exist", True) and not full.is_file():
        return False, f"missing file {full}"
    if step.get("must_not_exist") and full.exists():
        return False, f"file should not exist: {full}"
    min_bytes = int(step.get("min_bytes", 0))
    if min_bytes > 0 and full.is_file() and full.stat().st_size < min_bytes:
        return False, f"file too small: {full}"
    if full.is_file():
        body = full.read_text(encoding="utf-8", errors="replace")
        if step.get("deny_task_status") and re.search(r"TASK_STATUS:\s*\S+", body, re.I):
            return False, "file contains TASK_STATUS (chat leakage)"
        ok, detail = check_text_patterns(
            body,
            any_match=step.get("any_match") or step.get("content_any_match"),
            none_match=step.get("none_match") or step.get("content_none_match"),
            label="file",
        )
        if not ok:
            return False, detail
    return True, str(full)


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
                hub.discussion_diagnosis(ctx.base, ctx.collab_channel, required_agents=required),
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

    health = hub.check_health(base)
    if not health:
        print("  FAIL: hub not healthy", file=sys.stderr)
        return False

    if not hub.ensure_channel(base, channel):
        print(f"  FAIL: could not ensure channel {channel!r}", file=sys.stderr)
        return False

    required = scenario.get("required_agents") or []
    if required:
        ok_agents, missing = hub.verify_agents_online(base, required)
        if not ok_agents:
            print(f"  FAIL: required agents offline: {', '.join(missing)}", file=sys.stderr)
            print("  Start hub with CLI agents or set NJ_COLLAB_SCENARIO_AGENTS", file=sys.stderr)
            return False

    if not hub.free_scenario_capacity(base, channel):
        print("  FAIL: collab capacity full", file=sys.stderr)
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
    for i, step in enumerate(scenario.get("steps") or [], 1):
        if not run_step(ctx, step, f"{i}"):
            all_ok = False
            break

    cleanup = scenario.get("cleanup", "cancel")
    if cleanup == "cancel" and not keep:
        hub.send_message(base, ctx.collab_channel, f"/cancel-plan {ctx.collab_id[:8]}")
        hub.wait_phase(base, ctx.collab_channel, ctx.collab_id, "cancelled", 10)
        for extra in ctx.extra_collabs:
            hub.cancel_collab(base, extra)
        print("  ✓ cleanup: cancelled")
    elif keep:
        print(f"  --keep: collab {ctx.collab_id[:8]} left active")

    if all_ok:
        print(f"=== PASS: {name} ===\n")
    else:
        print(f"=== FAIL: {name} ===\n", file=sys.stderr)
    return all_ok


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
