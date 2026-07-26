"""Restore scenario fixture files from per-fixture .scenario-baseline snapshots."""
from __future__ import annotations

import json
import shutil
from pathlib import Path

# Runtime / install artifacts that may exist beside a partial baseline snapshot.
_PRUNE_SKIP_TOP = frozenset(
    {
        ".scenario-baseline",
        "node_modules",
        ".git",
        "dist",
        "build",
        "target",
        "vendor",
        ".venv",
        "__pycache__",
    }
)


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


def _prune_baseline_orphans(fixture_root: Path, baseline: Path, base_files: set[Path]) -> None:
    """Remove live files that sit in baseline-tracked dirs but are not in the snapshot.

    Baselines should cover the full fixture (or at least every directory they
    track). We only prune siblings of snapshotted files so agent-created orphans
    (e.g. core/sample/main.go) cannot poison later scenarios.

    Safety: never treat the fixture root as a prune zone unless the baseline
    already owns every root-level file. Partial baselines that snapshot only
    ``package.json`` / ``Makefile`` used to wipe the rest of the fixture.
    """
    tracked_dirs = {rel.parent for rel in base_files}
    root = Path(".")
    if root in tracked_dirs:
        live_root_files = {
            p.relative_to(fixture_root)
            for p in fixture_root.iterdir()
            if p.is_file() and p.name != ".DS_Store"
        }
        baseline_root_files = {rel for rel in base_files if rel.parent == root}
        if not live_root_files.issubset(baseline_root_files):
            tracked_dirs.discard(root)
    for live in list(fixture_root.rglob("*")):
        if not live.is_file():
            continue
        try:
            rel = live.relative_to(fixture_root)
        except ValueError:
            continue
        if any(part in _PRUNE_SKIP_TOP for part in rel.parts):
            continue
        if rel in base_files:
            continue
        if rel.parent not in tracked_dirs:
            continue
        try:
            live.unlink()
        except OSError:
            pass

    junk = fixture_root / ".neural-junkie"
    if junk.is_dir():
        shutil.rmtree(junk, ignore_errors=True)


def baseline_diverged_paths(scenario: dict, *, root: Path) -> list[str]:
    """Return relative paths whose live bytes differ from (or are missing vs) baseline."""
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    fixture = (ws_cfg.get("fixture") or "").strip()
    if not fixture:
        return []
    fixture_root = root / "scenarios" / "fixtures" / fixture
    baseline = fixture_root / ".scenario-baseline"
    if not baseline.is_dir():
        return []
    diverged: list[str] = []
    for src in baseline.rglob("*"):
        if not src.is_file():
            continue
        rel = src.relative_to(baseline)
        dest = fixture_root / rel
        if not dest.is_file():
            diverged.append(str(rel))
            continue
        if dest.read_bytes() != src.read_bytes():
            diverged.append(str(rel))
    return diverged


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
    base_files: set[Path] = set()
    for src in baseline.rglob("*"):
        if not src.is_file():
            continue
        rel = src.relative_to(baseline)
        base_files.add(rel)
        dest = fixture_root / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_bytes(src.read_bytes())
    _prune_baseline_orphans(fixture_root, baseline, base_files)
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
