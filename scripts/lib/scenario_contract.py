"""Validate scenario JSON deliverable contracts (CI smoke, no live hub)."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from scenario_assert import scenario_all_steps

ROOT = Path(__file__).resolve().parents[2]
IMPLEMENT_DIR = ROOT / "scenarios" / "implement"
COLLAB_DIR = ROOT / "scenarios" / "collab"
CHAT_DIR = ROOT / "scenarios" / "chat"
PARITY_DIR = ROOT / "scenarios" / "parity"
USER_FLOW_IMPLEMENT_DIR = ROOT / "scenarios" / "user-flows" / "implement"
USER_FLOW_COLLAB_DIR = ROOT / "scenarios" / "user-flows" / "collab"


def _has_quality_bar(spec: dict[str, Any]) -> bool:
    if spec.get("llm_judge"):
        return True
    if spec.get("contains"):
        return True
    if spec.get("any_match") or spec.get("content_any_match"):
        return True
    for_question = spec.get("for_question")
    if isinstance(for_question, dict):
        if for_question.get("any_match") or for_question.get("none_match") or for_question.get("contains_all"):
            return True
    return False


def implement_requires_deliverable_contract(scenario: dict) -> bool:
    if scenario.get("expect_no_deliverables"):
        return False
    for step in scenario_all_steps(scenario):
        if (step.get("action") or "").strip() != "send":
            continue
        meta = step.get("metadata") if isinstance(step.get("metadata"), dict) else {}
        mode = (meta.get("editor_mode") or "").strip().lower()
        if mode in ("ask", "plan"):
            continue
        if mode == "agent":
            return True
        if meta.get("implementation_session") and mode not in ("ask", "plan"):
            return True
    return False


def _agent_implement_sends(scenario: dict) -> list[dict]:
    sends: list[dict] = []
    for step in scenario_all_steps(scenario):
        if (step.get("action") or "").strip() != "send":
            continue
        meta = step.get("metadata") if isinstance(step.get("metadata"), dict) else {}
        mode = (meta.get("editor_mode") or "").strip().lower()
        if mode in ("ask", "plan"):
            continue
        if mode == "agent" or (meta.get("implementation_session") and mode not in ("ask", "plan")):
            sends.append(step)
    return sends


def validate_implement_wait_gates(scenario_relpath: str, scenario: dict) -> list[str]:
    """File-producing implement/parity waits must use disk and/or metadata — not chat phrases alone."""
    errors: list[str] = []
    if scenario.get("expect_no_deliverables"):
        return errors
    if not (
        scenario_relpath.startswith("implement/")
        or scenario_relpath.startswith("user-flows/implement/")
        or scenario_relpath.startswith("parity/")
    ):
        return errors
    if not implement_requires_deliverable_contract(scenario):
        return errors

    for index, step in enumerate(scenario_all_steps(scenario)):
        if (step.get("action") or "").strip() != "wait_reply":
            continue
        if step.get("until_any_match") and not (
            step.get("until_file_match")
            or step.get("until_file_exists")
            or step.get("until_files_exist")
            or step.get("until_file_absent")
            or step.get("until_files_absent")
            or step.get("until_metadata_keys")
        ):
            errors.append(
                f"{scenario_relpath}: steps wait_reply[{index}] uses until_any_match alone; "
                "prefer until_file_match / until_metadata_keys"
            )
    # At least one wait after an agent implement send should gate on disk or metadata.
    has_gated_wait = False
    for step in scenario_all_steps(scenario):
        if (step.get("action") or "").strip() != "wait_reply":
            continue
        if (
            step.get("until_file_match")
            or step.get("until_file_exists")
            or step.get("until_files_exist")
            or step.get("until_file_absent")
            or step.get("until_files_absent")
            or step.get("until_metadata_keys")
        ):
            has_gated_wait = True
            break
    if _agent_implement_sends(scenario) and not has_gated_wait:
        errors.append(
            f"{scenario_relpath}: file-producing scenario needs at least one wait_reply with "
            "until_file_match / until_file_exists / until_file_absent / until_metadata_keys"
        )
    return errors


def collab_requires_deliverable_contract(scenario: dict) -> bool:
    if scenario.get("expect_no_deliverables"):
        return False
    for step in scenario_all_steps(scenario):
        action = (step.get("action") or "").strip()
        if action in ("assert_files", "assert_deliverable", "approve_file_changes"):
            return True
    return False


def validate_deliverable_contract(scenario_relpath: str, scenario: dict) -> list[str]:
    errors: list[str] = []
    requires = False
    if scenario_relpath.startswith("implement/") or scenario_relpath.startswith(
        "user-flows/implement/"
    ) or scenario_relpath.startswith("parity/"):
        requires = implement_requires_deliverable_contract(scenario)
    elif scenario_relpath.startswith("collab/") or scenario_relpath.startswith(
        "user-flows/collab/"
    ):
        requires = collab_requires_deliverable_contract(scenario)

    if not requires:
        return errors

    if scenario.get("expect_no_deliverables"):
        return errors

    expect = scenario.get("expect_deliverables")
    if not isinstance(expect, list) or not expect:
        errors.append(f"{scenario_relpath}: file-producing scenario must declare expect_deliverables")
        return errors

    for i, raw in enumerate(expect):
        if not isinstance(raw, dict):
            errors.append(f"{scenario_relpath}: expect_deliverables[{i}] must be an object")
            continue
        path = (raw.get("path") or "").strip()
        if not path:
            errors.append(f"{scenario_relpath}: expect_deliverables[{i}] missing path")
            continue
        must_exist = raw.get("must_exist", True)
        if raw.get("must_not_exist"):
            must_exist = False
        if must_exist and not _has_quality_bar(raw):
            errors.append(
                f"{scenario_relpath}: expect_deliverables[{i}] ({path}) needs for_question, contains, or llm_judge"
            )
    return errors


def validate_scenario_shape(scenario_relpath: str, scenario: dict) -> list[str]:
    errors: list[str] = []
    if not str(scenario.get("name") or "").strip():
        errors.append(f"{scenario_relpath}: missing name")
    steps = scenario.get("steps", scenario.get("turns"))
    if not isinstance(steps, list) or not steps:
        errors.append(f"{scenario_relpath}: steps must be a non-empty list")
        return errors
    for index, step in enumerate(steps):
        if not isinstance(step, dict):
            errors.append(f"{scenario_relpath}: steps[{index}] must be an object")
        elif not str(step.get("action") or step.get("type") or "").strip():
            errors.append(f"{scenario_relpath}: steps[{index}] missing action")

    evaluation = scenario.get("evaluation")
    if isinstance(evaluation, dict) and evaluation.get("long_horizon") is True:
        sends = [
            step
            for step in scenario_all_steps(scenario)
            if str(step.get("action") or "").strip() == "send"
        ]
        minimum = int(evaluation.get("min_user_turns") or 4)
        if len(sends) < minimum:
            errors.append(
                f"{scenario_relpath}: long-horizon scenario needs at least {minimum} send turns"
            )
        if scenario_relpath.startswith("chat/") and not any(
            str(step.get("action") or "").strip() == "assert_transcript_metrics"
            for step in scenario_all_steps(scenario)
        ):
            errors.append(
                f"{scenario_relpath}: long-horizon chat scenario needs assert_transcript_metrics"
            )

    tags = {str(t).strip().lower() for t in (scenario.get("tags") or []) if str(t).strip()}
    if scenario_relpath.startswith("chat/") and ("dialogue" in tags or "coherence" in tags):
        errors.extend(validate_dialogue_scenario_contract(scenario_relpath, scenario))
    return errors


def validate_dialogue_scenario_contract(scenario_relpath: str, scenario: dict) -> list[str]:
    """Dialogue/coherence chat scenarios must prove multi-turn continuity."""
    errors: list[str] = []
    sends = [
        step
        for step in scenario_all_steps(scenario)
        if str(step.get("action") or "").strip() == "send"
    ]
    if len(sends) < 3:
        errors.append(f"{scenario_relpath}: dialogue/coherence tag requires at least 3 send turns")

    has_metrics = any(
        str(step.get("action") or "").strip() == "assert_transcript_metrics"
        for step in scenario_all_steps(scenario)
    )
    has_continuity_assert = False
    for step in scenario_all_steps(scenario):
        if str(step.get("action") or "").strip() != "assert_messages":
            continue
        if step.get("any_match") and step.get("none_match"):
            has_continuity_assert = True
            break
    if not has_metrics and not has_continuity_assert:
        errors.append(
            f"{scenario_relpath}: dialogue/coherence needs assert_transcript_metrics "
            "and/or assert_messages with any_match + none_match continuity guards"
        )
    return errors


def load_scenario_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, dict):
        raise ValueError(f"{path}: expected JSON object")
    return data


def validate_all_scenario_contracts() -> list[str]:
    errors: list[str] = []
    for directory, prefix in (
        (IMPLEMENT_DIR, "implement"),
        (COLLAB_DIR, "collab"),
        (CHAT_DIR, "chat"),
        (PARITY_DIR, "parity"),
        (USER_FLOW_IMPLEMENT_DIR, "user-flows/implement"),
        (USER_FLOW_COLLAB_DIR, "user-flows/collab"),
    ):
        if not directory.is_dir():
            continue
        for path in sorted(directory.glob("*.json")):
            rel = f"{prefix}/{path.name}"
            try:
                scenario = load_scenario_json(path)
            except (OSError, json.JSONDecodeError, ValueError) as exc:
                errors.append(f"{rel}: invalid JSON ({exc})")
                continue
            errors.extend(validate_scenario_shape(rel, scenario))
            errors.extend(validate_deliverable_contract(rel, scenario))
            errors.extend(validate_implement_wait_gates(rel, scenario))
    return errors


def main() -> int:
    errors = validate_all_scenario_contracts()
    if errors:
        for err in errors:
            print(err, file=__import__("sys").stderr)
        print(f"\n{len(errors)} deliverable contract error(s)", file=__import__("sys").stderr)
        return 1
    print("OK: scenario shapes and deliverable contracts are valid")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
