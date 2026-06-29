"""Restore scenario fixture files from per-fixture .scenario-baseline snapshots."""
from __future__ import annotations

import json
from pathlib import Path


def _looks_like_json_package(path: Path) -> bool:
    if not path.is_file():
        return True
    try:
        text = path.read_text(encoding="utf-8").strip()
    except OSError:
        return False
    if not text.startswith("{"):
        return False
    try:
        json.loads(text)
    except json.JSONDecodeError:
        return False
    return True


def reset_fixture_baseline(scenario: dict, *, root: Path) -> None:
    """Restore known-good fixture files before/after implement scenarios run."""
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    fixture = (ws_cfg.get("fixture") or "").strip()
    if not fixture:
        return
    fixture_root = root / "scenarios" / "fixtures" / fixture
    baseline = fixture_root / ".scenario-baseline"
    if not baseline.is_dir():
        return
    for src in baseline.rglob("*"):
        if not src.is_file():
            continue
        rel = src.relative_to(baseline)
        dest = fixture_root / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_bytes(src.read_bytes())
    pkg = fixture_root / "package.json"
    if pkg.is_file() and not _looks_like_json_package(pkg):
        baseline_pkg = baseline / "package.json"
        if baseline_pkg.is_file():
            pkg.write_bytes(baseline_pkg.read_bytes())


def reset_all_fixture_baselines(*, root: Path) -> None:
    """Reset every fixture that ships a .scenario-baseline directory."""
    fixtures_root = root / "scenarios" / "fixtures"
    if not fixtures_root.is_dir():
        return
    for baseline in sorted(fixtures_root.glob("*/.scenario-baseline")):
        fixture = baseline.parent.name
        reset_fixture_baseline({"workspace": {"fixture": fixture}}, root=root)
