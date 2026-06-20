#!/usr/bin/env python3
"""Smoke-test Gemini CLI auth for deliverable judging (headless PASS/FAIL prompt)."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts"))

from lib.gemini_judge_auth import check_gemini_judge  # noqa: E402


def main() -> int:
    p = argparse.ArgumentParser(description="Smoke-test Gemini deliverable judge auth")
    p.add_argument("--timeout", type=float, default=60.0, help="Seconds to wait for gemini")
    args = p.parse_args()

    ok, detail = check_gemini_judge(timeout_s=args.timeout)
    if ok:
        print(detail)
        return 0
    print(f"FAIL: {detail}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
