#!/usr/bin/env python3
"""Preflight checklist before make release-prep (hub, judge, fixtures, smoke)."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
PY = sys.executable

sys.path.insert(0, str(SCRIPTS_DIR))

from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.release_prep_env import apply_release_prep_env  # noqa: E402
from lib.release_prep_hub import ensure_hub_for_release_prep  # noqa: E402
from lib.hub_regression import restart_regression_hub  # noqa: E402

SMOKE_IMPLEMENT = (
    "go-test-failure-repair",
    "at-file-explicit-path",
    "deny-destructive-command",
)
SMOKE_COLLAB = ("collab-participation-two-agent-strict",)


def git_checkout_fixtures(root: Path) -> list[str]:
    notes: list[str] = []
    proc = subprocess.run(
        ["git", "checkout", "--", "scenarios/fixtures/"],
        cwd=root,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        notes.append(f"git checkout fixtures failed: {proc.stderr.strip()}")
    orphan = root / "scenarios/fixtures/react-vite-corrupt-appjs/src/App.js"
    if orphan.is_file():
        orphan.unlink()
        notes.append("removed orphan react-vite-corrupt-appjs/src/App.js")
    dirty = subprocess.run(
        ["git", "status", "--porcelain", "scenarios/fixtures/"],
        cwd=root,
        capture_output=True,
        text=True,
    )
    lines = [ln for ln in (dirty.stdout or "").splitlines() if ln.strip()]
    if lines:
        notes.append(f"fixtures still dirty: {len(lines)} path(s)")
    else:
        notes.append("fixtures clean (git)")
    return notes


def run_go_tests(root: Path) -> tuple[bool, str]:
    proc = subprocess.run(
        [
            "go",
            "test",
            "./internal/agent/...",
            "./internal/collaboration/...",
            "./internal/protocol/...",
            "-count=1",
        ],
        cwd=root,
        capture_output=True,
        text=True,
    )
    tail = (proc.stdout or "") + (proc.stderr or "")
    if len(tail) > 4000:
        tail = tail[-4000:]
    return proc.returncode == 0, tail


def run_smoke(hub: str, *, skip_smoke: bool) -> tuple[bool, list[str]]:
    if skip_smoke:
        return True, ["smoke scenarios skipped"]
    lines: list[str] = []
    ok = True
    for name in SMOKE_IMPLEMENT:
        proc = subprocess.run(
            [PY, str(SCRIPTS_DIR / "implement-scenarios.py"), "--scenario", name, "--hub", hub],
            cwd=ROOT,
            capture_output=True,
            text=True,
            env={**os.environ, "NEURAL_JUNKIE_RATE_LIMIT": "0", "PYTHONUNBUFFERED": "1"},
        )
        passed = proc.returncode == 0
        lines.append(f"implement/{name}: {'PASS' if passed else 'FAIL'}")
        ok = ok and passed
    for name in SMOKE_COLLAB:
        proc = subprocess.run(
            [
                PY,
                str(SCRIPTS_DIR / "collab-scenarios.py"),
                "--scenario",
                name,
                "--hub",
                hub,
            ],
            cwd=ROOT,
            capture_output=True,
            text=True,
            env={**os.environ, "NEURAL_JUNKIE_RATE_LIMIT": "0", "PYTHONUNBUFFERED": "1"},
        )
        passed = proc.returncode == 0
        lines.append(f"collab/{name}: {'PASS' if passed else 'FAIL'}")
        ok = ok and passed
    return ok, lines


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--no-restart-hub", action="store_true")
    p.add_argument("--skip-smoke", action="store_true", help="Skip live implement/collab smoke scenarios")
    p.add_argument("--skip-go-test", action="store_true")
    args = p.parse_args()

    apply_release_prep_env(ROOT)
    hub = args.hub.rstrip("/")
    failures: list[str] = []

    print("=== release-prep-ready ===\n")

    print(">>> fixtures")
    for note in git_checkout_fixtures(ROOT):
        print(f"  {note}")

    if not args.skip_go_test:
        print("\n>>> go test (agent, collaboration, protocol)")
        ok, detail = run_go_tests(ROOT)
        print(detail.rstrip() or "(no output)")
        if not ok:
            failures.append("go test")

    print("\n>>> hub + deliverable judge")
    if not args.no_restart_hub:
        if not restart_regression_hub(ROOT, hub, timeout_s=120.0, env=os.environ.copy()):
            failures.append("hub restart")
    if not ensure_hub_for_release_prep(hub, root=ROOT, allow_restart=not args.no_restart_hub):
        failures.append("hub/judge")

    print("\n>>> regression preflight cleanup")
    preflight_regression_run(ROOT, hub, label="release-prep-ready")

    print("\n>>> collab-preflight")
    proc = subprocess.run(
        [
            PY,
            str(SCRIPTS_DIR / "collab-preflight.py"),
            "--hub",
            hub,
            "--skip-judge-smoke",
        ],
        cwd=ROOT,
        env={**os.environ, "NEURAL_JUNKIE_RATE_LIMIT": "0"},
    )
    if proc.returncode != 0:
        failures.append("collab-preflight")

    print("\n>>> smoke scenarios")
    smoke_ok, smoke_lines = run_smoke(hub, skip_smoke=args.skip_smoke)
    for line in smoke_lines:
        print(f"  {line}")
    if not smoke_ok:
        failures.append("smoke scenarios")

    print()
    if failures:
        print(f"NOT READY — failed: {', '.join(failures)}", file=sys.stderr)
        return 1

    print("READY for release prep.")
    print()
    print("Start full gate:")
    print("  cd", ROOT)
    print("  make release-prep")
    print()
    print("Or:")
    print(f"  NEURAL_JUNKIE_RATE_LIMIT=0 {PY} scripts/release-prep.py --hub {hub}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
