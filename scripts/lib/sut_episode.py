"""Load and validate SUT self-improve episode JSON under scenarios/sut/."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
SUT_DIR = ROOT / "scenarios" / "sut"

DEFAULT_FIX_ALLOW_PATHS = [
    "internal/",
    "cmd/",
    "desktop/src/",
    "scripts/",
    "docs/",
]


def list_episodes() -> list[str]:
    if not SUT_DIR.is_dir():
        return []
    return sorted(p.stem for p in SUT_DIR.glob("*.json"))


def episode_path(name: str) -> Path:
    stem = name.strip().removesuffix(".json")
    return SUT_DIR / f"{stem}.json"


def load_episode(name: str) -> dict[str, Any]:
    path = episode_path(name)
    with path.open(encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, dict):
        raise ValueError(f"{path}: expected JSON object")
    return data


def validate_sut_episode(scenario_relpath: str, scenario: dict[str, Any]) -> list[str]:
    """Contract for scenarios/sut/*.json — adaptive human + judge, optional seed steps."""
    errors: list[str] = []
    if not str(scenario.get("name") or "").strip():
        errors.append(f"{scenario_relpath}: missing name")

    target = str(scenario.get("target_agent") or "").strip()
    if not target:
        errors.append(f"{scenario_relpath}: missing target_agent")

    human = scenario.get("human")
    if not isinstance(human, dict):
        errors.append(f"{scenario_relpath}: human must be an object")
    else:
        if not str(human.get("persona") or "").strip():
            errors.append(f"{scenario_relpath}: human.persona required")
        if not str(human.get("goal") or "").strip():
            errors.append(f"{scenario_relpath}: human.goal required")
        max_turns = human.get("max_turns", 4)
        try:
            if int(max_turns) < 1:
                errors.append(f"{scenario_relpath}: human.max_turns must be >= 1")
        except (TypeError, ValueError):
            errors.append(f"{scenario_relpath}: human.max_turns must be an int")

    judge = scenario.get("judge")
    if not isinstance(judge, dict):
        errors.append(f"{scenario_relpath}: judge must be an object")
    else:
        if not str(judge.get("rubric") or "").strip():
            errors.append(f"{scenario_relpath}: judge.rubric required")

    fix = scenario.get("fix")
    if fix is not None and not isinstance(fix, dict):
        errors.append(f"{scenario_relpath}: fix must be an object when present")

    steps = scenario.get("steps")
    if steps is not None:
        if not isinstance(steps, list):
            errors.append(f"{scenario_relpath}: steps must be a list when present")
        else:
            for index, step in enumerate(steps):
                if not isinstance(step, dict):
                    errors.append(f"{scenario_relpath}: steps[{index}] must be an object")
                elif not str(step.get("action") or step.get("type") or "").strip():
                    errors.append(f"{scenario_relpath}: steps[{index}] missing action")

    return errors


def fix_enabled(episode: dict[str, Any]) -> bool:
    fix = episode.get("fix")
    if not isinstance(fix, dict):
        return True
    return bool(fix.get("enabled", True))


def fix_allow_paths(episode: dict[str, Any]) -> list[str]:
    fix = episode.get("fix") if isinstance(episode.get("fix"), dict) else {}
    raw = fix.get("allow_paths") if isinstance(fix, dict) else None
    if isinstance(raw, list) and raw:
        return [str(p).strip() for p in raw if str(p).strip()]
    return list(DEFAULT_FIX_ALLOW_PATHS)
