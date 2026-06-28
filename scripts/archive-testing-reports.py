#!/usr/bin/env python3
"""Move older timestamped docs/testing reports to docs/archive/testing/, keeping the newest N per prefix."""
from __future__ import annotations

import argparse
import re
import shutil
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
TESTING = ROOT / "docs" / "testing"
ARCHIVE = ROOT / "docs" / "archive" / "testing"

# Timestamped run artifacts (basename prefix → keep newest N files sharing that prefix).
PREFIXES = (
    "test-everything-",
    "release-prep-",
    "release-prep-retry-",
    "model-benchmark-quick-",
    "model-benchmark-release-",
    "parity-stable",
    "parity-stable-restart",
    "regression-bundle-",
)

# Permanent docs that never move.
KEEP_NAMES = {
    "MODEL_BENCHMARK.md",
    "stable-platform-smoke.md",
    "collab-matrix.tsv",
}


def report_prefix(name: str) -> str | None:
    for prefix in PREFIXES:
        if name.startswith(prefix):
            return prefix
    return None


def group_key(path: Path) -> str:
    name = path.name
    prefix = report_prefix(name)
    if prefix:
        return prefix
    m = re.match(r"^(collab-sweep|remote-ssh-dogfood|utility-tier-ab)-", name)
    if m:
        return m.group(1) + "-"
    return name


def main() -> int:
    p = argparse.ArgumentParser(description="Archive old docs/testing timestamped reports")
    p.add_argument("--keep", type=int, default=5, help="Keep newest N files per prefix group")
    p.add_argument("--dry-run", action="store_true", help="Print moves without executing")
    args = p.parse_args()

    if not TESTING.is_dir():
        print(f"missing {TESTING}")
        return 1

    groups: dict[str, list[Path]] = defaultdict(list)
    for path in sorted(TESTING.iterdir()):
        if not path.is_file():
            continue
        if path.name in KEEP_NAMES:
            continue
        if path.suffix not in {".md", ".json", ".tsv", ".log"}:
            continue
        groups[group_key(path)].append(path)

    ARCHIVE.mkdir(parents=True, exist_ok=True)
    moved = 0
    for key, paths in sorted(groups.items()):
        if len(paths) <= args.keep:
            continue
        paths.sort(key=lambda p: p.name, reverse=True)
        for path in paths[args.keep :]:
            dest = ARCHIVE / path.name
            if args.dry_run:
                print(f"would move {path.relative_to(ROOT)} -> {dest.relative_to(ROOT)}")
            else:
                shutil.move(str(path), str(dest))
                print(f"moved {path.name} -> docs/archive/testing/")
            moved += 1

    print(f"{'would move' if args.dry_run else 'moved'} {moved} file(s); kept {args.keep} newest per group in docs/testing/")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
