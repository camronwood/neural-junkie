#!/usr/bin/env python3
"""Return 0 if the N most recent away-morning reports are all-green.

Usage:
  python3 scripts/check-consecutive-away-green.py
  python3 scripts/check-consecutive-away-green.py --count 2
  CONSECUTIVE_GREEN=2 python3 scripts/check-consecutive-away-green.py

A report is green when every Gate results row is **PASS** (no **FAIL**).
"""

from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
TESTING = ROOT / "docs" / "testing"


def report_is_green(path: Path) -> bool:
    text = path.read_text(encoding="utf-8", errors="replace")
    # Prefer Gate results table rows
    fails = re.findall(r"\|\s*`[^`]+`\s*\|\s*\*\*FAIL\*\*", text)
    passes = re.findall(r"\|\s*`[^`]+`\s*\|\s*\*\*PASS\*\*", text)
    if fails:
        return False
    if passes:
        return True
    # Fallback: Summary FAIL count
    m = re.search(r"- FAIL:\s*(\d+)", text)
    if m:
        return int(m.group(1)) == 0
    return False


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--count",
        type=int,
        default=int(os.environ.get("CONSECUTIVE_GREEN") or "2"),
        help="Number of consecutive green mornings required",
    )
    args = ap.parse_args()
    if args.count < 1:
        print("count must be >= 1", file=sys.stderr)
        return 2

    if not TESTING.is_dir():
        print(f"missing {TESTING}", file=sys.stderr)
        return 1

    reports = sorted(
        TESTING.glob("away-morning-????-??-??.md"),
        key=lambda p: p.name,
        reverse=True,
    )
    # Prefer dated stamp reports only (exclude odd names without ISO date)
    reports = [p for p in reports if re.match(r"away-morning-\d{4}-\d{2}-\d{2}\.md$", p.name)]
    if len(reports) < args.count:
        print(
            f"NEED_MORE: have {len(reports)} dated away-morning reports, need {args.count}",
            flush=True,
        )
        return 1

    chosen = reports[: args.count]
    for p in chosen:
        ok = report_is_green(p)
        print(f"{'GREEN' if ok else 'RED '} {p.relative_to(ROOT)}", flush=True)
        if not ok:
            return 1

    print(f"OK: {args.count} consecutive green away mornings", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
