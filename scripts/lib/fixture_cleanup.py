"""Shared cleanup for live scenario / release-prep regression runs."""

from __future__ import annotations

import shutil
import sys
import time
from pathlib import Path

SCENARIO_CHANNELS = (
    "chat-scenarios",
    "collab-scenarios",
    "collab-scenarios-solo",
    "learning-scenarios",
    "implement-scenarios",
)


def cleanup_fixture_collabs(root: Path, *, dry_run: bool = False) -> list[str]:
    """Remove gitignored runtime dirs under scenarios/fixtures/*/collabs."""
    fixtures = root / "scenarios" / "fixtures"
    if not fixtures.is_dir():
        return []

    removed: list[str] = []
    for collabs_dir in sorted(fixtures.glob("*/collabs")):
        if not collabs_dir.is_dir():
            continue
        for child in sorted(collabs_dir.iterdir()):
            if not child.is_dir():
                continue
            rel = child.relative_to(root)
            label = str(rel)
            if dry_run:
                print(f"  would remove fixture collab: {label}")
            else:
                shutil.rmtree(child, ignore_errors=True)
            removed.append(label)
    return removed


def cleanup_scenario_channels(hub_url: str, *, dry_run: bool = False) -> list[str]:
    """Clear history on standard scenario hub channels."""
    try:
        from lib import collab_hub as hub_api  # type: ignore
    except ImportError:
        print("  skip hub channel cleanup (collab_hub import failed)", file=sys.stderr)
        return []

    cleared: list[str] = []
    for channel in SCENARIO_CHANNELS:
        if dry_run:
            print(f"  would clear channel history: {channel}")
            cleared.append(channel)
            continue
        hub_api.ensure_channel(hub_url, channel, description="Scenario regression channel")
        ok = hub_api.clear_channel_history(hub_url, channel, max_retries=8)
        if ok:
            cleared.append(channel)
        else:
            print(f"  ⚠ clear-history failed: {channel}", file=sys.stderr)
        time.sleep(0.5)
    return cleared


def preflight_regression_run(
    root: Path,
    hub_url: str,
    *,
    dry_run: bool = False,
    label: str = "regression preflight",
) -> tuple[int, int]:
    """Reset fixture collabs and scenario hub channels before a long live sweep."""
    print(f"\n>>> [{label}] fixture collabs + hub channel cleanup")
    collab_removed = cleanup_fixture_collabs(root, dry_run=dry_run)
    if collab_removed:
        action = "would remove" if dry_run else "removed"
        print(f"  {action} {len(collab_removed)} fixture collab dir(s)")
    else:
        print("  fixture collabs: already clean")

    cleared = cleanup_scenario_channels(hub_url, dry_run=dry_run)
    if cleared:
        action = "would clear" if dry_run else "cleared"
        print(f"  {action} {len(cleared)} scenario channel(s)")
    else:
        print("  scenario channels: none cleared")

    return len(collab_removed), len(cleared)
