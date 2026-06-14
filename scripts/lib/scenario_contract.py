"""Validate scenario JSON deliverable contracts (CI smoke, no live hub)."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from scenario_assert import scenario_all_steps

ROOT = Path(__file__).resolve().parents[2]
IMPLEMENT_DIR = ROOT / "scenarios" / "implement"
COLLAB_DIR = ROOT / "scenarios" / "collab"


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
        if (meta.get("editor_mode") or "").strip().lower() == "ask":
            return False
        if (meta.get("editor_mode") or "").strip().lower() == "agent":
            return True
        if meta.get("implementation_session") and (meta.get("editor_mode") or "").strip().lower() != "ask":
            return True
    return False


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
    if scenario_relpath.startswith("implement/"):
        requires = implement_requires_deliverable_contract(scenario)
    elif scenario_relpath.startswith("collab/"):
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


def load_scenario_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, dict):
        raise ValueError(f"{path}: expected JSON object")
    return data


def validate_all_scenario_contracts() -> list[str]:
    errors: list[str] = []
    for directory, prefix in ((IMPLEMENT_DIR, "implement"), (COLLAB_DIR, "collab")):
        if not directory.is_dir():
            continue
        for path in sorted(directory.glob("*.json")):
            rel = f"{prefix}/{path.name}"
            try:
                scenario = load_scenario_json(path)
            except (OSError, json.JSONDecodeError, ValueError) as exc:
                errors.append(f"{rel}: invalid JSON ({exc})")
                continue
            errors.extend(validate_deliverable_contract(rel, scenario))
    return errors


def main() -> int:
    errors = validate_all_scenario_contracts()
    if errors:
        for err in errors:
            print(err, file=__import__("sys").stderr)
        print(f"\n{len(errors)} deliverable contract error(s)", file=__import__("sys").stderr)
        return 1
    print("OK: all file-producing scenarios declare expect_deliverables with quality bars")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
