#!/usr/bin/env python3
"""Run implementation session scenarios against a live hub."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.eval_telemetry import (  # noqa: E402
    absorb_messages,
    absorb_metadata,
    finish,
    metrics_payload,
    new_run,
    record_reason,
)
from lib.hub_regression import ensure_hub_with_recovery  # noqa: E402
from lib.regression_boot import maybe_boot_regression, restart_hub_for_live_run  # noqa: E402
from lib.release_prep_env import release_prep_env  # noqa: E402
from lib.scenario_assert import (  # noqa: E402
    check_file_deliverable,
    check_text_patterns,
    expand_deliverable_steps,
    looks_like_read_only_inspection_command,
    merge_deliverable_step,
    scenario_question,
)
from lib.scenario_wait import (  # noqa: E402
    disk_wait_satisfied,
    metadata_get,
    normalize_meta_keys,
    step_has_disk_wait,
)
from lib.fixture_baseline import baseline_diverged_paths, reset_all_fixture_baselines, reset_fixture_baseline  # noqa: E402
from lib.workspace_context import enrich_send_metadata, ide_route_for_target_agent  # noqa: E402

SCENARIOS_DIR = ROOT / "scenarios" / "implement"
DEFAULT_CHANNEL = "implement-scenarios"
DEFAULT_FROM = "ImplementScenario"
SCENARIO_APPROVE_CHANNELS = frozenset(
    {"implement-scenarios", "user-flow-scenarios", "parity-scenarios"}
)
MID_WAIT_APPROVE_INTERVAL_S = 8.0


def _truthy_env(name: str) -> bool:
    return (os.environ.get(name) or "").strip().lower() in ("1", "true", "yes", "on")


def _scenario_auto_approve_enabled(channel: str) -> bool:
    ch = (channel or "").strip()
    if _truthy_env("NJ_REGRESSION"):
        return True
    if ch in SCENARIO_APPROVE_CHANNELS:
        return True
    return ch.endswith("-scenarios")


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
    def __init__(self, base: str, scenario: dict, telemetry: dict | None = None) -> None:
        self.base = base.rstrip("/")
        self.scenario = scenario
        self.channel = (scenario.get("channel") or DEFAULT_CHANNEL).strip()
        self.target_agent = (scenario.get("target_agent") or "BackendEngineer").strip().lstrip("@")
        self.baseline_agent_count: dict[str, int] = {}
        self.file_snapshots: dict[str, str] = {}
        self.telemetry = telemetry or new_run("implement", str(scenario.get("name") or "unknown"))


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
    timeout = step.get("timeout", "540s")
    secs = 540
    if isinstance(timeout, str) and timeout.endswith("s"):
        try:
            secs = int(timeout[:-1])
        except ValueError:
            pass
    if _truthy_env("NJ_REGRESSION_USER_FLOWS"):
        try:
            floor = int((os.environ.get("NJ_IMPLEMENT_WAIT_REPLY_FLOOR_SECS") or "1200").strip())
        except ValueError:
            floor = 1200
        secs = max(secs, floor)
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    # Prefer disk/metadata completion signals. Chat-phrase until_any_match is legacy.
    until_any = step.get("until_any_match")
    until_meta_keys = normalize_meta_keys(step)
    has_disk = step_has_disk_wait(step)
    baseline = int(step.get("baseline", ctx.baseline_agent_count.get(from_name, _chat_baseline(ctx, from_name))))
    root = Path(scenario_repo_root(ctx.scenario))
    deadline = time.time() + secs
    nudged = False
    last_approve = 0.0
    auto_approve = _scenario_auto_approve_enabled(ctx.channel)
    while time.time() < deadline:
        if auto_approve and time.time() - last_approve >= MID_WAIT_APPROVE_INTERVAL_S:
            last_approve = time.time()
            n, ids = hub.wait_and_approve_file_changes(
                ctx.base,
                ctx.channel,
                min_approved=0,
                timeout=2.0,
            )
            if n > 0:
                print(f"  wait_reply: auto-approved {n} file change(s) ids={ids}", flush=True)
        disk_ok, disk_detail = disk_wait_satisfied(root, step)
        msgs = hub.list_messages(ctx.base, ctx.channel, 200)
        # Implementation turns often surface as file_change / user_question without a plain chat row.
        pool = hub.agent_messages(
            msgs,
            types=hub.CHAT_REPLY_TYPES | {"file_change"},
        )
        candidates = [m for m in pool if m.get("from", {}).get("name") == from_name]

        if has_disk and disk_ok and not until_meta_keys:
            return True, f"disk ready ({disk_detail})"

        for msg in candidates[baseline:]:
            text = msg.get("content") or ""
            meta = msg.get("metadata") if isinstance(msg.get("metadata"), dict) else {}
            proposal = meta.get("file_change_proposal") if isinstance(meta, dict) else None
            if isinstance(proposal, dict):
                path = str(proposal.get("path") or proposal.get("rel_path") or "")
                if path:
                    text = f"{text}\n{path}"
            if until_meta_keys:
                if not all(metadata_get(meta, key) is not None for key in until_meta_keys):
                    continue
                if has_disk and not disk_ok:
                    continue
                suffix = f"; {disk_detail}" if disk_detail else ""
                return True, f"reply from {from_name} (metadata: {', '.join(until_meta_keys)}{suffix})"
            if has_disk:
                # Waiting on disk — keep polling; do not accept a bare chat reply as completion.
                continue
            if until_any:
                ok, detail = check_text_patterns(text, any_match=until_any)
                if ok:
                    return True, f"reply from {from_name} ({detail})"
            else:
                return True, f"reply from {from_name}"
        # Mid-wait nudge once if the assignee is silent (common under Ollama soak).
        if not nudged and time.time() > deadline - (secs * 0.45):
            nudged = True
            mention = from_name if from_name.startswith("@") else f"@{from_name}"
            route = ide_route_for_target_agent(from_name)
            nudge_meta: dict = {
                "conversation_mode": "code",
                "implementation_session": True,
                "editor_mode": "agent",
            }
            if route:
                nudge_meta["ide_route_agent_type"] = route
            hub.send_message(
                ctx.base,
                ctx.channel,
                f"{mention} please continue the implementation from the prior request.",
                metadata=nudge_meta,
                from_name=DEFAULT_FROM,
            )
            record_reason(ctx.telemetry, "nudge_reasons", f"silent agent after {secs * 0.55:.0f}s")
            print(f"  wait_reply: nudged silent @{from_name}", flush=True)
        time.sleep(2)
    hub.abort_channel_agents(ctx.base, ctx.channel, held_by="ImplementScenario")
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
            # optional demotes chat-phrase any_match only; none_match stays hard.
            if step.get("optional") or step.get("any_match_optional"):
                print(f"  soft any_match miss: {detail}", flush=True)
            else:
                return False, detail
    if none_match := step.get("none_match"):
        ok, detail = check_text_patterns(text, none_match=none_match)
        if not ok:
            return False, detail
    if max_chars := step.get("max_chars"):
        limit = int(max_chars)
        if len(text) > limit:
            return False, f"reply exceeds max_chars {limit} (got {len(text)})"
    return True, "message assertions ok"


def step_assert_deliverable(ctx: ImplementContext, step: dict, *, skip_llm_judge: bool = False) -> tuple[bool, str]:
    root = scenario_repo_root(ctx.scenario)
    spec = merge_deliverable_step(ctx.scenario, step)
    if skip_llm_judge:
        spec.pop("llm_judge", None)
    rel = (spec.get("path") or "").strip()
    if not rel:
        return False, "assert_deliverable: path required"
    alts = step.get("path_alternatives") or spec.get("path_alternatives") or []
    candidates = [rel] + [str(a).strip() for a in alts if str(a).strip()]
    last_detail = ""
    root_path = Path(root)
    for i, candidate in enumerate(candidates):
        ok, detail = check_file_deliverable(
            root=root,
            rel=candidate,
            spec=spec,
            question=scenario_question(ctx.scenario),
            hub_base=ctx.base,
        )
        if ok:
            return True, detail
        if i == 0 and (root_path / candidate).is_file():
            return False, detail
        last_detail = detail
    return False, last_detail


def step_assert_file_exists(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    return step_assert_deliverable(ctx, step, skip_llm_judge=True)


def step_assert_file_absent(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    return step_assert_deliverable(ctx, {**step, "action": "assert_file_absent"})


def _last_agent_message(ctx: ImplementContext, from_name: str) -> dict | None:
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    pool = hub.chat_agent_messages(msgs)
    candidates = [m for m in pool if m.get("from", {}).get("name") == from_name]
    if not candidates:
        return None
    # Prefer replies from this scenario turn only — otherwise a prior
    # implementation_session_outcome on a shared channel poisons plan/ask asserts.
    baseline = int(ctx.baseline_agent_count.get(from_name, 0))
    turn = candidates[baseline:] if baseline < len(candidates) else []
    if not turn:
        turn = candidates[-1:]
    # Always use the latest turn reply. Walking backward for an outcome resurrects
    # prior-scenario metadata when the current reply is plan/ask (no outcome).
    return turn[-1]


def step_assert_no_file_change(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    # Pass = disk matches fixture baseline. Chat metadata from a prior turn on a
    # shared channel must not fail plan/ask scenarios when the tree is untouched.
    diverged = baseline_diverged_paths(ctx.scenario, root=ROOT)
    if diverged:
        return False, f"files changed on disk: {', '.join(diverged[:8])}"
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    msg = _last_agent_message(ctx, from_name)
    if msg:
        body = (msg.get("content") or "").lower()
        if "[file_change]" in body:
            return False, "file_change in reply"
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


def step_assert_message_metadata(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    from_name = (step.get("from") or ctx.target_agent).strip().lstrip("@")
    msg = _last_agent_message(ctx, from_name)
    require_keys = [str(k) for k in (step.get("require_keys") or [])]
    # When requiring session outcome, scan this turn for a message that has it —
    # the absolute latest row may be a status chat without metadata.
    if msg and require_keys:
        msgs = hub.list_messages(ctx.base, ctx.channel, 200)
        pool = hub.chat_agent_messages(msgs)
        candidates = [m for m in pool if m.get("from", {}).get("name") == from_name]
        baseline = int(ctx.baseline_agent_count.get(from_name, 0))
        turn = candidates[baseline:] if baseline < len(candidates) else candidates[-1:]
        for candidate in reversed(turn):
            meta = candidate.get("metadata") or {}
            if all(metadata_get(meta, key) is not None for key in require_keys):
                msg = candidate
                break
    if not msg:
        if step.get("optional"):
            return True, "skipped (no agent reply)"
        return False, f"no messages from {from_name}"
    meta = msg.get("metadata") or {}
    for key, expected in (step.get("equals") or {}).items():
        got = metadata_get(meta, str(key))
        if got != expected:
            if step.get("optional"):
                return True, f"skipped optional metadata mismatch on {key!r}"
            return False, f"metadata {key!r}: got {got!r} want {expected!r}"
    for key, pattern in (step.get("match") or {}).items():
        got = metadata_get(meta, str(key))
        text = "" if got is None else str(got)
        if not re.search(str(pattern), text, re.I):
            if step.get("optional"):
                return True, f"skipped optional metadata pattern on {key!r}"
            return False, f"metadata {key!r} did not match {pattern!r} (got {text!r})"
    for key in require_keys:
        if metadata_get(meta, key) is None:
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


def step_assert_shell(ctx: ImplementContext, step: dict) -> tuple[bool, str]:
    root = scenario_repo_root(ctx.scenario)
    command = (step.get("command") or step.get("shell") or "").strip()
    if not command:
        return False, "assert_shell: command required"
    proc = subprocess.run(
        command,
        shell=True,
        cwd=root,
        capture_output=True,
        text=True,
        timeout=float(step.get("timeout_s", 120)),
    )
    combined = (proc.stdout or "") + (proc.stderr or "")
    if proc.returncode != 0:
        tail = combined[-800:] if len(combined) > 800 else combined
        return False, f"exit {proc.returncode}: {tail.strip()}"
    none_match = step.get("none_match") or []
    if isinstance(none_match, str):
        none_match = [none_match]
    for pattern in none_match:
        if re.search(pattern, combined, re.IGNORECASE):
            return False, f"output matched forbidden pattern {pattern!r}"
    return True, "shell command ok"


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
    "assert_shell": step_assert_shell,
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


def bootstrap_ci(values: list[float], iterations: int = 1000, ci: float = 0.95) -> tuple[float, float]:
    """Non-parametric bootstrap CI for the mean of values in [0,1]."""
    if not values:
        return 0.0, 0.0
    import random

    rng = random.Random(42)
    n = len(values)
    means: list[float] = []
    for _ in range(iterations):
        sample = [values[rng.randrange(n)] for _ in range(n)]
        means.append(sum(sample) / n)
    means.sort()
    lo = means[int((1 - ci) / 2 * iterations)]
    hi = means[int((1 + ci) / 2 * iterations) - 1]
    return lo, hi


def _outcome_from_last_reply(ctx: ImplementContext, from_name: str) -> dict:
    """Prefer an outcome stamped on this turn; never resurrect prior-scenario metadata."""
    msgs = hub.list_messages(ctx.base, ctx.channel, 200)
    pool = hub.chat_agent_messages(msgs)
    candidates = [m for m in pool if m.get("from", {}).get("name") == from_name]
    if not candidates:
        return {}
    baseline = int(ctx.baseline_agent_count.get(from_name, 0))
    turn = candidates[baseline:] if baseline < len(candidates) else candidates[-1:]
    fallback: dict = {}
    for msg in reversed(turn):
        meta = msg.get("metadata") or {}
        outcome = meta.get("implementation_session_outcome")
        if isinstance(outcome, dict):
            # Nudge continuations often take a conversational path and stamp session_not_run
            # even when the primary turn ran (or is still running) an implementation session.
            if outcome.get("routing_reason") != "session_not_run":
                return outcome
            if not fallback:
                fallback = outcome
    return fallback


def run_scenario_with_stats(base: str, name: str, *, runs: int, best_of_k: int, keep: bool) -> bool:
    scenario = load_scenario(name)
    target = (scenario.get("target_agent") or "BackendEngineer").strip().lstrip("@")
    passes: list[bool] = []
    failure_types: dict[str, int] = {}
    repair_attempts: list[int] = []

    for run_idx in range(1, runs + 1):
        if runs > 1:
            print(f"\n--- run {run_idx}/{runs} ---")
        ok, outcome = run_scenario(base, name, keep=keep)
        passes.append(ok)
        ft = str(outcome.get("failure_type") or ("pass" if ok else "unknown"))
        failure_types[ft] = failure_types.get(ft, 0) + 1
        attempts = outcome.get("repair_attempts")
        if isinstance(attempts, (int, float)):
            repair_attempts.append(int(attempts))

    if runs <= 1:
        return passes[0]

    pass_rate = sum(1 for p in passes if p) / len(passes)
    ci_lo, ci_hi = bootstrap_ci([1.0 if p else 0.0 for p in passes])
    print(f"\n=== stats: {name} ({runs} runs) ===")
    print(f"  pass_rate: {pass_rate:.1%}  bootstrap_95%_CI: [{ci_lo:.1%}, {ci_hi:.1%}]")
    if repair_attempts:
        repair_attempts.sort()
        median_repairs = repair_attempts[len(repair_attempts) // 2]
        print(f"  median_repair_attempts: {median_repairs}")
    if failure_types:
        print("  failure_type_breakdown:")
        for key in sorted(failure_types):
            print(f"    {key}: {failure_types[key]}")

    if best_of_k > 1:
        groups = [passes[i : i + best_of_k] for i in range(0, len(passes), best_of_k)]
        groups = [g for g in groups if len(g) == best_of_k]
        if groups:
            bok_rate = sum(1 for g in groups if any(g)) / len(groups)
            bok_ci = bootstrap_ci([1.0 if any(g) else 0.0 for g in groups])
            print(f"  best_of_{best_of_k}_pass_rate: {bok_rate:.1%}  CI: [{bok_ci[0]:.1%}, {bok_ci[1]:.1%}]")
            return bok_rate > 0

    return pass_rate > 0


def run_scenario(base: str, name: str, *, keep: bool = False) -> tuple[bool, dict]:
    from lib.scenario_flake_retry import maybe_retry_after_failure, pause_before_retry

    outcome: dict = {}
    last_detail = ""
    telemetry = new_run("implement", name)
    # Selection-scoped extract is late in alpha order and flaky under Ollama soak — allow 3 attempts.
    # Greenfield user-flow implement journeys are long-running and flake under soak too.
    user_flow_implement = {
        "trip-research-vacation",
        "rust-blackjack-2d",
        "nodejs-user-crud",
        "ios-trivia-swift",
        "journey-crud-clarify-correct",
        "journey-blackjack-cli-correction",
        "journey-boot-fix-then-feature",
        "journey-notes-rename-to-memos",
        "journey-landing-brand-correction",
    }
    max_attempts = 3 if name in user_flow_implement or name == "selection-scoped-edit" else 2
    for attempt in range(1, max_attempts + 1):
        telemetry["attempts"] = attempt
        ok, outcome, last_detail = _run_scenario_once(
            base, name, keep=keep, telemetry=telemetry
        )
        absorb_metadata(telemetry, outcome)
        if ok:
            telemetry["passed_at_1"] = attempt == 1
            report = finish(telemetry, eventual_pass=True)
            print("EVAL_JSON:" + json.dumps(report, separators=(",", ":")))
            print(
                "METRICS_JSON:"
                + json.dumps(metrics_payload(report, outcome), separators=(",", ":"))
            )
            return ok, {**outcome, "evaluation": report}
        should_retry = maybe_retry_after_failure(
            base, name, last_detail, attempt, max_attempts=max_attempts
        )
        if not should_retry:
            break
        record_reason(telemetry, "retry_reasons", last_detail)
        # Clear stuck implementation turns before the next attempt.
        hub.abort_channel_agents(base, "implement-scenarios", held_by="ImplementScenario")
        pause_before_retry(8.0)
    report = finish(telemetry, eventual_pass=False)
    print("EVAL_JSON:" + json.dumps(report, separators=(",", ":")))
    print(
        "METRICS_JSON:"
        + json.dumps(metrics_payload(report, outcome), separators=(",", ":"))
    )
    return False, {**outcome, "evaluation": report}


def _run_scenario_once(
    base: str,
    name: str,
    *,
    keep: bool = False,
    telemetry: dict | None = None,
) -> tuple[bool, dict, str]:
    scenario = load_scenario(name)
    ctx = ImplementContext(base, scenario, telemetry=telemetry)
    empty_outcome: dict = {}
    print(f"\n=== implement: {name} ===")
    if not ensure_hub_ready(base, f"implement:{name}"):
        return False, empty_outcome, "hub unhealthy"
    required = scenario.get("required_agents") or [ctx.target_agent]
    required = [str(ag).strip().lstrip("@") for ag in required if str(ag).strip()]
    from lib.regression_collab import unpause_keep_agents

    unpause_keep_agents(base, required)
    ok, missing = hub.verify_agents_online(base, required)
    if not ok:
        print(f"  FAIL: offline agents: {missing}", file=sys.stderr)
        return False, empty_outcome, f"offline agents: {missing}"
    if not ensure_channel(ctx):
        return False, empty_outcome, "could not ensure channel"
    hub.abort_channel_agents(ctx.base, ctx.channel, held_by="ImplementScenario")
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
    last_detail = ""
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
                last_detail = f"unknown action {action}"
                print(f"  FAIL {last_detail}", file=sys.stderr)
                all_ok = False
                break
            ok, detail = fn(ctx, step)
            print(f"  {'✓' if ok else '✗'} [{i}] {action}: {detail}")
            if not ok:
                last_detail = detail
                all_ok = False
                break
    finally:
        outcome = _outcome_from_last_reply(ctx, ctx.target_agent)
        absorb_messages(ctx.telemetry, hub.list_messages(ctx.base, ctx.channel, 200))
        reset_fixture_baseline(scenario, root=ROOT)
        if scenario.get("cleanup", "clear") == "clear" and not keep:
            hub.clear_channel_history(ctx.base, ctx.channel)
    print(f"=== {'PASS' if all_ok else 'FAIL'}: {name} ===\n")
    return all_ok, outcome, last_detail


def _list_implement_names(*, include_optional: bool) -> list[str]:
    names: list[str] = []
    for path in sorted(SCENARIOS_DIR.glob("*.json")):
        try:
            sc = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            names.append(path.stem)
            continue
        if sc.get("optional") and not include_optional:
            continue
        names.append(path.stem)
    return names


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--list", action="store_true")
    p.add_argument("--scenario")
    p.add_argument("--all", action="store_true")
    p.add_argument(
        "--include-optional",
        action="store_true",
        help="Include optional/soak twin scenarios in --all (default: skip optional)",
    )
    p.add_argument("--runs", type=int, default=1, help="Repeat each scenario N times and report pass-rate stats")
    p.add_argument("--best-of-k", type=int, default=0, help="Report best-of-K pass rate across run groups (requires --runs >= K)")
    p.add_argument("--hub", default=hub.DEFAULT_HUB)
    p.add_argument("--keep", action="store_true")
    p.add_argument("--pack-dir", help="Pack repo root; scenarios read from <pack-dir>/scenarios/implement")
    args = p.parse_args()
    global SCENARIOS_DIR
    if args.pack_dir:
        SCENARIOS_DIR = Path(args.pack_dir).resolve() / "scenarios" / "implement"
    elif os.environ.get("NJ_PACK_SCENARIOS_DIR", "").strip():
        SCENARIOS_DIR = Path(os.environ["NJ_PACK_SCENARIOS_DIR"]).resolve()
    include_optional = args.include_optional or os.environ.get("NJ_IMPLEMENT_INCLUDE_OPTIONAL", "").strip().lower() in (
        "1",
        "true",
        "yes",
    )
    if args.list:
        for f in sorted(SCENARIOS_DIR.glob("*.json")):
            try:
                sc = json.loads(f.read_text(encoding="utf-8"))
                opt = " (optional)" if sc.get("optional") else ""
            except (OSError, json.JSONDecodeError):
                opt = ""
            print(f"{f.stem}{opt}")
        return 0
    if not maybe_boot_regression(args.hub, root=ROOT, label="implement-scenarios"):
        return 1
    if args.all:
        names = _list_implement_names(include_optional=include_optional)
        skipped = [
            f.stem
            for f in sorted(SCENARIOS_DIR.glob("*.json"))
            if f.stem not in names
        ]
        if skipped and not include_optional:
            print(f"  skipping {len(skipped)} optional soak twin(s): {', '.join(skipped)}")
    else:
        names = [args.scenario] if args.scenario else []
    if not names:
        p.print_help()
        return 1
    if args.all:
        no_restart = os.environ.get("NO_RESTART_HUB", "").strip().lower() in ("1", "true", "yes")
        if not args.keep and not no_restart:
            if not restart_hub_for_live_run(ROOT, args.hub, label="implement-scenarios"):
                return 1
        reset_all_fixture_baselines(root=ROOT)
        all_agents: set[str] = set()
        for n in names:
            sc = load_scenario(n)
            for ag in sc.get("required_agents") or [sc.get("target_agent") or "BackendEngineer"]:
                if ag:
                    all_agents.add(str(ag).strip().lstrip("@"))
        ok, missing = hub.verify_agents_online(args.hub, sorted(all_agents))
        if not ok:
            print(f"  FAIL: required agents offline: {missing}", file=sys.stderr)
            print("  Enable software-development pack and ensure hub agents are active.", file=sys.stderr)
            return 1
        print(f"  preflight: {len(all_agents)} agent(s) online: {', '.join(sorted(all_agents))} ({len(names)} scenarios)")
    failed = sum(
        1
        for n in names
        if not run_scenario_with_stats(args.hub, n, runs=max(1, args.runs), best_of_k=args.best_of_k, keep=args.keep)
    )
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
