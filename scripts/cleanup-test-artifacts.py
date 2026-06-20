#!/usr/bin/env python3
"""Remove Neural Junkie test artifacts from ~/.neural-junkie (repo caches, scenario channels).

Safe defaults: deletes repo indexes under temp/test paths and widget-expert caches only.
Use --dry-run to preview. Use --hub to clear scenario channel history on a running hub.
"""
from __future__ import annotations

import argparse
import json
import shutil
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR / "lib"))

from lib.fixture_cleanup import (  # noqa: E402
    cleanup_fixture_collabs,
    cleanup_scenario_channels,
    SCENARIO_CHANNELS,
)

DEFAULT_HOME = Path.home() / ".neural-junkie"
DEFAULT_HUB = "http://127.0.0.1:18765"
ROOT = SCRIPT_DIR.parent

TEST_PATH_MARKERS = (
    "/tmp/",
    "/T/",
    "/var/folders/",
    "TestRepoAgent",
    "TestRepositoryAgent",
    "TestHelperAgent",
    "neural-junkie-test-home",
    "neural-junkie-agent-test-home",
)

TEST_AGENT_NAMES = {
    "widget-expert",
    "testrepoagent",
    "testrepoagent1",
    "testrepoagent2",
    "dmrepoagent",
}

SCENARIO_CHANNELS = SCENARIO_CHANNELS  # re-export for backwards compatibility


def is_test_path(path: str) -> bool:
    p = (path or "").strip()
    if not p:
        return False
    return any(m in p for m in TEST_PATH_MARKERS)


def is_test_agent_name(name: str) -> bool:
    n = (name or "").strip().lower()
    if not n:
        return False
    if n in TEST_AGENT_NAMES:
        return True
    return n.startswith("test")


def is_test_repo_metadata(meta: dict) -> bool:
    path = meta.get("path") or meta.get("Path") or ""
    if is_test_path(str(path)):
        return True
    names = meta.get("agent_names") or meta.get("AgentNames") or []
    return any(is_test_agent_name(str(n)) for n in names)


def cleanup_repo_caches(home: Path, *, dry_run: bool) -> list[str]:
    repos_dir = home / "repos"
    if not repos_dir.is_dir():
        return []

    removed: list[str] = []
    for cache_dir in sorted(repos_dir.iterdir()):
        if not cache_dir.is_dir():
            continue
        meta_file = cache_dir / "metadata.json"
        if not meta_file.is_file():
            continue
        try:
            meta = json.loads(meta_file.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            continue
        if not is_test_repo_metadata(meta):
            continue
        label = f"{cache_dir.name} ({meta.get('path', '?')})"
        if dry_run:
            print(f"  would remove repo cache: {label}")
        else:
            shutil.rmtree(cache_dir, ignore_errors=True)
            print(f"  removed repo cache: {label}")
        removed.append(label)
    return removed


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--home", type=Path, default=DEFAULT_HOME, help="Neural Junkie data dir")
    p.add_argument("--dry-run", action="store_true", help="Preview without deleting")
    p.add_argument("--hub", default="", help=f"Hub URL to clear scenario channels (default: skip)")
    p.add_argument(
        "--fixture-collabs",
        action="store_true",
        help="Remove gitignored collab dirs under scenarios/fixtures/*/collabs",
    )
    args = p.parse_args()

    home: Path = args.home.expanduser()
    if not home.is_dir():
        print(f"Nothing to clean: {home} does not exist")
        return 0

    print(f"Cleaning test artifacts under {home}")
    repo_removed = cleanup_repo_caches(home, dry_run=args.dry_run)
    print(f"Repo caches: {len(repo_removed)} {'would be ' if args.dry_run else ''}removed")

    if args.fixture_collabs:
        print(f"Fixture collab cleanup → {ROOT / 'scenarios' / 'fixtures'}")
        collab_removed = cleanup_fixture_collabs(ROOT, dry_run=args.dry_run)
        for label in collab_removed:
            prefix = "would remove" if args.dry_run else "removed"
            print(f"  {prefix} fixture collab: {label}")
        print(f"Fixture collabs: {len(collab_removed)} {'would be ' if args.dry_run else ''}removed")

    if args.hub:
        print(f"Hub channel cleanup → {args.hub}")
        cleared = cleanup_scenario_channels(args.hub, dry_run=args.dry_run)
        for channel in cleared:
            prefix = "would clear" if args.dry_run else "cleared"
            print(f"  {prefix} channel history: {channel}")
        print(f"Scenario channels: {len(cleared)} {'would be ' if args.dry_run else ''}cleared")

    if args.dry_run:
        print("Dry run complete — re-run without --dry-run to apply.")
    else:
        print("Cleanup complete.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
