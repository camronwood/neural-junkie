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

DEFAULT_HOME = Path.home() / ".neural-junkie"
DEFAULT_HUB = "http://127.0.0.1:18765"

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

SCENARIO_CHANNELS = (
    "chat-scenarios",
    "collab-scenarios",
    "learning-scenarios",
    "implement-scenarios",
)


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


def cleanup_scenario_channels(hub_url: str, *, dry_run: bool) -> list[str]:
    try:
        import collab_hub as hub_api  # type: ignore
    except ImportError:
        print("  skip hub channel cleanup (collab_hub import failed)", file=sys.stderr)
        return []

    cleared: list[str] = []
    for channel in SCENARIO_CHANNELS:
        if dry_run:
            print(f"  would clear channel history: {channel}")
            cleared.append(channel)
            continue
        ok = hub_api.clear_channel_history(hub_url, channel)
        if ok:
            print(f"  cleared channel history: {channel}")
            cleared.append(channel)
        else:
            print(f"  ⚠ clear-history failed: {channel}", file=sys.stderr)
    return cleared


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--home", type=Path, default=DEFAULT_HOME, help="Neural Junkie data dir")
    p.add_argument("--dry-run", action="store_true", help="Preview without deleting")
    p.add_argument("--hub", default="", help=f"Hub URL to clear scenario channels (default: skip)")
    args = p.parse_args()

    home: Path = args.home.expanduser()
    if not home.is_dir():
        print(f"Nothing to clean: {home} does not exist")
        return 0

    print(f"Cleaning test artifacts under {home}")
    repo_removed = cleanup_repo_caches(home, dry_run=args.dry_run)
    print(f"Repo caches: {len(repo_removed)} {'would be ' if args.dry_run else ''}removed")

    if args.hub:
        print(f"Hub channel cleanup → {args.hub}")
        cleared = cleanup_scenario_channels(args.hub, dry_run=args.dry_run)
        print(f"Scenario channels: {len(cleared)} {'would be ' if args.dry_run else ''}cleared")

    if args.dry_run:
        print("Dry run complete — re-run without --dry-run to apply.")
    else:
        print("Cleanup complete.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
