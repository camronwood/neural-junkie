"""Tests for fixture baseline restore + orphan prune."""
from __future__ import annotations

from pathlib import Path

from lib.fixture_baseline import reset_fixture_baseline


def test_reset_prunes_orphan_sibling_in_tracked_dir(tmp_path: Path) -> None:
    root = tmp_path
    fixture = root / "scenarios" / "fixtures" / "demo"
    baseline = fixture / ".scenario-baseline" / "core" / "sample"
    baseline.mkdir(parents=True)
    (baseline / "math.go").write_text("package sample\n", encoding="utf-8")
    live_dir = fixture / "core" / "sample"
    live_dir.mkdir(parents=True)
    (live_dir / "math.go").write_text("package sample\nbuggy\n", encoding="utf-8")
    (live_dir / "main.go").write_text("package main\n", encoding="utf-8")
    # Outside tracked dirs — must survive (partial baselines).
    (fixture / "README.md").write_text("keep me\n", encoding="utf-8")
    junk = fixture / ".neural-junkie" / "backups"
    junk.mkdir(parents=True)
    (junk / "x").write_text("x", encoding="utf-8")

    reset_fixture_baseline({"workspace": {"fixture": "demo"}}, root=root)

    assert (live_dir / "math.go").read_text(encoding="utf-8") == "package sample\n"
    assert not (live_dir / "main.go").exists()
    assert (fixture / "README.md").exists()
    assert not (fixture / ".neural-junkie").exists()


def test_reset_does_not_wipe_fixture_root_on_partial_baseline(tmp_path: Path) -> None:
    """Partial baselines that only snapshot package.json must not prune siblings."""
    root = tmp_path
    fixture = root / "scenarios" / "fixtures" / "demo"
    baseline = fixture / ".scenario-baseline"
    baseline.mkdir(parents=True)
    (baseline / "package.json").write_text('{"name":"demo"}\n', encoding="utf-8")
    (baseline / "src").mkdir()
    (baseline / "src" / "App.tsx").write_text("export default function App(){}\n", encoding="utf-8")
    (fixture / "package.json").write_text('{"name":"demo","dirty":true}\n', encoding="utf-8")
    (fixture / "index.html").write_text("<html></html>\n", encoding="utf-8")
    (fixture / "vite.config.ts").write_text("export default {}\n", encoding="utf-8")
    src = fixture / "src"
    src.mkdir()
    (src / "App.tsx").write_text("export default function App(){ /* dirty */ }\n", encoding="utf-8")
    (src / "main.tsx").write_text("import './App'\n", encoding="utf-8")

    reset_fixture_baseline({"workspace": {"fixture": "demo"}}, root=root)

    assert (fixture / "package.json").read_text(encoding="utf-8") == '{"name":"demo"}\n'
    assert (fixture / "index.html").exists()
    assert (fixture / "vite.config.ts").exists()
    assert (src / "App.tsx").read_text(encoding="utf-8") == "export default function App(){}\n"
    # src/ is tracked and baseline is incomplete for src — orphan main.tsx still pruned
    assert not (src / "main.tsx").exists()


def test_baseline_diverged_paths_detects_edits(tmp_path: Path) -> None:
    from lib.fixture_baseline import baseline_diverged_paths

    root = tmp_path
    fixture = root / "scenarios" / "fixtures" / "demo"
    baseline = fixture / ".scenario-baseline" / "core" / "sample"
    baseline.mkdir(parents=True)
    (baseline / "math.go").write_text("ok\n", encoding="utf-8")
    live = fixture / "core" / "sample"
    live.mkdir(parents=True)
    (live / "math.go").write_text("changed\n", encoding="utf-8")
    scenario = {"workspace": {"fixture": "demo"}}
    assert baseline_diverged_paths(scenario, root=root) == ["core/sample/math.go"]
    (live / "math.go").write_text("ok\n", encoding="utf-8")
    assert baseline_diverged_paths(scenario, root=root) == []
