#!/usr/bin/env python3
"""Regression checks for agent terminal wait-cue + workspace-denial false positives.

Runs without a live hub (pure Go tests). Use after code changes:

  python3 scripts/check-terminal-wait-cue.py
"""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def run(cmd: list[str]) -> int:
    print("+", " ".join(cmd), flush=True)
    return subprocess.call(cmd, cwd=ROOT)


def main() -> int:
    packages = [
        "./internal/protocol/",
        "./internal/agent/",
    ]
    run_filters = [
        ("internal/protocol", "DetectCommands"),
        ("internal/agent", "ApplySuggestedCommands_skips|LooksLikeAsksUserToPaste_flags|FalseShell|TerminalBash|ValidateResponseKeeps"),
    ]
    failed = 0
    for pkg, pattern in run_filters:
        rc = run(["go", "test", f"./{pkg}/", f"-run={pattern}", "-count=1"])
        if rc != 0:
            failed = rc
    if failed == 0:
        print("OK: terminal wait-cue + workspace-denial regressions passed")
    return failed


if __name__ == "__main__":
    sys.exit(main())
