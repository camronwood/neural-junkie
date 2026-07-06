#!/usr/bin/env python3
"""Smoke-test Gemini CLI auth for deliverable judging (headless PASS/FAIL prompt)."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts"))

from lib.gemini_judge_auth import ensure_gemini_for_testing  # noqa: E402


def main() -> int:
    p = argparse.ArgumentParser(description="Smoke-test Gemini deliverable judge auth")
    p.add_argument("--timeout", type=float, default=60.0, help="Seconds to wait per probe")
    p.add_argument("--model", help="Probe only this model (default: fast → pro → fast-light)")
    args = p.parse_args()

    sel = ensure_gemini_for_testing(
        root=ROOT,
        timeout_s=args.timeout,
        explicit_model=args.model or "",
        retry_quota=True,
    )
    if sel.ok:
        print(sel.detail)
        return 0
    print(f"FAIL: {sel.detail}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
