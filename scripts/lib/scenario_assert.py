"""Shared text-pattern and deliverable assertions for scenario runners."""

from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Any

try:
    from lib.deliverable_judge import judge_deliverable
except ImportError:
    from deliverable_judge import judge_deliverable  # type: ignore[no-redef]

# Mirrors internal/protocol/command_context.go stack-tool prefixes for scenario checks.
_STACK_CMD_HEAD_RE = re.compile(
    r"^\s*(docker(?:-compose)?|compose|npm|yarn|pnpm|npx|kubectl|helm|terraform|make|mvn|gradle)\b",
    re.I,
)

DELIVERABLE_ASSERT_ACTIONS = frozenset(
    {"assert_deliverable", "assert_file_exists", "assert_files", "assert_file_absent"}
)

# Scenario harness control sends (not the user's original request).
_HARNESS_CONTROL_SEND_CMDS = frozenset(
    {
        "/resume-plan",
        "/complete-collab",
        "/cancel-plan",
    }
)

def looks_like_stack_tool_command(command: str) -> bool:
    command = (command or "").strip()
    if not command:
        return False
    first_line = command.split("\n", 1)[0].strip()
    return bool(_STACK_CMD_HEAD_RE.match(first_line))


_READ_ONLY_PREFIXES = (
    "ls",
    "pwd",
    "cd ",
    "cat ",
    "head ",
    "tail ",
    "grep ",
    "find ",
    "file ",
    "tree",
    "wc ",
    "sort ",
    "uniq ",
    "which ",
    "whereis ",
    "git status",
    "git log",
    "git diff",
    "git show",
    "git branch",
)


def looks_like_read_only_inspection_command(command: str) -> bool:
    command = (command or "").strip().lower()
    if not command:
        return False
    return any(command.startswith(prefix) for prefix in _READ_ONLY_PREFIXES)


def check_text_patterns(
    text: str,
    *,
    any_match: list[str] | None = None,
    none_match: list[str] | None = None,
    label: str = "content",
) -> tuple[bool, str]:
    for pattern in none_match or []:
        if re.search(pattern, text, re.I | re.MULTILINE):
            return False, f"{label} none_match {pattern!r}"
    if any_match:
        if not any(re.search(p, text, re.I | re.MULTILINE) for p in any_match):
            return False, f"{label} any_match not found (want one of {any_match!r})"
    return True, "ok"


_MARKDOWN_BULLET_RE = re.compile(r"^\s*(?:[-*+]\s+|\d+\.\s+)", re.MULTILINE)


def count_markdown_bullets(text: str) -> int:
    return len(_MARKDOWN_BULLET_RE.findall(text))


def check_min_markdown_bullets(
    text: str,
    minimum: int,
    *,
    label: str = "content",
) -> tuple[bool, str]:
    count = count_markdown_bullets(text)
    if count < minimum:
        return False, f"{label} min_markdown_bullets {count} < {minimum}"
    return True, "ok"


def check_contains_all(text: str, patterns: list[str], *, label: str = "content") -> tuple[bool, str]:
    for pattern in patterns:
        try:
            if not re.search(pattern, text, re.I | re.MULTILINE):
                return False, f"{label} missing contains_all {pattern!r}"
        except re.error as exc:
            return False, f"{label} invalid contains_all pattern {pattern!r}: {exc}"
    return True, "ok"


def scenario_all_steps(scenario: dict) -> list[dict]:
    steps: list[dict] = []
    for key in ("setup", "steps"):
        for step in scenario.get(key) or []:
            if isinstance(step, dict):
                steps.append(step)
    return steps


def is_harness_control_send(content: str) -> bool:
    """True when send content is a scenario harness slash command, not a user request."""
    text = (content or "").strip()
    if not text.startswith("/"):
        return False
    cmd = text.split(None, 1)[0].lower()
    return cmd in _HARNESS_CONTROL_SEND_CMDS


def scenario_question(scenario: dict) -> str:
    collaborate = scenario.get("collaborate") if isinstance(scenario.get("collaborate"), dict) else {}
    goal = (collaborate.get("goal") or "").strip()
    if goal:
        return goal
    for step in scenario_all_steps(scenario):
        if (step.get("action") or "").strip() != "send":
            continue
        content = (step.get("content") or "").strip()
        if content and not is_harness_control_send(content):
            return content
    return (scenario.get("description") or "").strip()


def lookup_expect_deliverable(scenario: dict, path: str) -> dict[str, Any]:
    """Return the expect_deliverables contract entry for a path, if any."""
    expect = scenario.get("expect_deliverables")
    if not isinstance(expect, list):
        return {}
    want = (path or "").strip()
    for spec in expect:
        if isinstance(spec, dict) and (spec.get("path") or "").strip() == want:
            return dict(spec)
    return {}


def merge_deliverable_step(scenario: dict, step: dict) -> dict[str, Any]:
    """Merge explicit assert step with expect_deliverables contract (step wins on conflicts)."""
    path = (step.get("path") or "").strip()
    merged: dict[str, Any] = dict(lookup_expect_deliverable(scenario, path))
    action = (step.get("action") or "").strip()
    if action == "assert_file_absent":
        merged["must_exist"] = False

    for key, val in step.items():
        if key == "action":
            continue
        if key == "for_question" and isinstance(val, dict):
            base = merged.get("for_question")
            fq = dict(base) if isinstance(base, dict) else {}
            fq.update(val)
            merged["for_question"] = fq
            continue
        merged[key] = val

    if path:
        merged["path"] = path
    return merged


def deliverable_step_paths(steps: list[dict]) -> set[str]:
    paths: set[str] = set()
    for step in steps:
        action = (step.get("action") or "").strip()
        if action in DELIVERABLE_ASSERT_ACTIONS:
            path = (step.get("path") or "").strip()
            if path:
                paths.add(path)
    return paths


def expand_deliverable_steps(scenario: dict) -> list[dict]:
    """Append assert_deliverable steps from expect_deliverables when path not already asserted."""
    expect = scenario.get("expect_deliverables")
    if not isinstance(expect, list) or not expect:
        return []
    existing = deliverable_step_paths(scenario_all_steps(scenario))
    extra: list[dict] = []
    for spec in expect:
        if not isinstance(spec, dict):
            continue
        path = (spec.get("path") or "").strip()
        if not path or path in existing:
            continue
        step = {"action": "assert_deliverable", **spec}
        extra.append(step)
        existing.add(path)
    return extra


def _llm_judge_enabled(spec: dict) -> bool:
    llm = spec.get("llm_judge")
    if llm is True:
        return True
    if isinstance(llm, dict) and llm.get("enabled", True):
        return True
    return False


def _llm_judge_criteria(spec: dict) -> str:
    llm = spec.get("llm_judge")
    if isinstance(llm, dict):
        return (llm.get("criteria") or "").strip()
    return ""


def judge_strict_enabled() -> bool:
    """When NJ_DELIVERABLE_JUDGE_STRICT is set, judge fail/exception fails the scenario."""
    raw = (os.environ.get("NJ_DELIVERABLE_JUDGE_STRICT") or "").strip().lower()
    return raw in ("1", "true", "yes", "on")


def _format_judge_detail(kind: str, detail: str, score: float | None) -> str:
    if score is not None:
        return f"judge:{kind}:SCORE={score:.2f}:{detail}"
    return f"judge:{kind}:{detail}"


def check_file_deliverable(
    *,
    root: str,
    rel: str,
    spec: dict[str, Any],
    question: str = "",
    collab_id: str = "",
    hub_base: str = "",
) -> tuple[bool, str]:
    rel = rel.replace("<collab-id>", collab_id)
    if not rel:
        return False, "path required"

    full = Path(root) / rel
    must_exist = spec.get("must_exist", True)
    if spec.get("must_not_exist"):
        must_exist = False

    if not must_exist:
        if full.is_file():
            return False, f"expected absent, still exists: {full}"
        return True, f"{rel} absent"

    if not full.is_file():
        return False, f"missing {full}"

    min_bytes = int(spec.get("min_bytes", 0))
    if min_bytes > 0 and full.stat().st_size < min_bytes:
        return False, f"file too small: {full} ({full.stat().st_size} < {min_bytes})"

    body = full.read_text(encoding="utf-8", errors="replace")

    if spec.get("deny_task_status") and re.search(r"TASK_STATUS:\s*\S+", body, re.I):
        return False, "file contains TASK_STATUS (chat leakage)"

    if want := spec.get("contains"):
        if want not in body:
            return False, f"{rel} missing contains {want!r}"

    for_question = spec.get("for_question") if isinstance(spec.get("for_question"), dict) else {}

    min_bullets = for_question.get("min_markdown_bullets")
    if min_bullets is None:
        min_bullets = spec.get("min_markdown_bullets")
    if min_bullets is not None:
        ok, detail = check_min_markdown_bullets(body, int(min_bullets), label=f"file {rel}")
        if not ok:
            return False, detail

    contains_all = for_question.get("contains_all") or spec.get("contains_all")
    if contains_all:
        ok, detail = check_contains_all(body, list(contains_all), label=f"file {rel}")
        if not ok:
            return False, detail

    any_match = (
        for_question.get("any_match")
        or spec.get("any_match")
        or spec.get("content_any_match")
    )
    none_match = (
        for_question.get("none_match")
        or spec.get("none_match")
        or spec.get("content_none_match")
    )
    ok, detail = check_text_patterns(
        body,
        any_match=any_match,
        none_match=none_match,
        label=f"file {rel}",
    )
    if not ok:
        snippet = "\n".join(body.splitlines()[:20])
        if snippet:
            detail = f"{detail}\n--- file snippet (first 20 lines) ---\n{snippet}"
        return False, detail

    if _llm_judge_enabled(spec):
        strict = judge_strict_enabled()
        try:
            ok, detail, score = judge_deliverable(
                question=question,
                rel_path=rel,
                file_body=body,
                criteria=_llm_judge_criteria(spec),
                llm_judge_spec=spec.get("llm_judge"),
                hub_base=hub_base,
                work_dir=root,
            )
        except Exception as exc:  # noqa: BLE001
            if strict:
                return False, f"judge:fail:exception:{exc}"
            return True, f"judge:warn:exception:{exc}"
        if not ok:
            if strict:
                return False, _format_judge_detail("fail", detail, score)
            return True, _format_judge_detail("warn", detail, score)
        return True, _format_judge_detail("pass", detail, score)

    return True, rel