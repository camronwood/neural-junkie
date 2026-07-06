#!/usr/bin/env python3
"""Smoke-test Claude Code CLI for deliverable judging and collab preflight."""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from lib.claude_judge_auth import ensure_claude_for_testing  # noqa: E402


def main() -> int:
    sel = ensure_claude_for_testing(timeout_s=120.0, smoke=True)
    if sel.ok:
        print(f"OK: {sel.detail}")
        return 0
    print(f"FAIL: {sel.detail}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
