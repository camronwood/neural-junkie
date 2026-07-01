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
        # Drop stale collabs that hold channel state and block clear-history.
        for collab in hub_api.list_active_collaborations(hub_url):
            if not isinstance(collab, dict):
                continue
            collab_ch = (collab.get("channel") or "").strip()
            starter = (collab.get("starter_channel") or collab.get("source_channel") or "").strip()
            if collab_ch == channel or starter == channel:
                cid = (collab.get("id") or "").strip()
                if cid:
                    hub_api.cancel_collab(hub_url, {"id": cid, "channel": collab_ch or channel})
                    time.sleep(0.25)
        ok = hub_api.clear_channel_history(hub_url, channel, max_retries=8)
        if not ok:
            # Last resort: drop any remaining active collabs then retry once.
            for collab in hub_api.list_active_collaborations(hub_url):
                if not isinstance(collab, dict):
                    continue
                cid = (collab.get("id") or "").strip()
                if not cid:
                    continue
                collab_ch = (collab.get("channel") or "").strip()
                hub_api.cancel_collab(hub_url, {"id": cid, "channel": collab_ch or channel})
                time.sleep(0.25)
            ok = hub_api.clear_channel_history(hub_url, channel, max_retries=4)
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

    if not dry_run and hub_url:
        try:
            from lib import collab_hub as hub_api  # type: ignore

            n = hub_api.cancel_all_active_collabs(hub_url)
            if n:
                print(f"  cancelled {n} active collaboration(s)")
            removed_agents = hub_api.cleanup_test_agents(hub_url)
            if removed_agents:
                print(f"  deleted test agents: {', '.join(removed_agents)}")
        except ImportError:
            print("  skip hub collab/agent cleanup (collab_hub import failed)", file=sys.stderr)

    cleared = cleanup_scenario_channels(hub_url, dry_run=dry_run)
    if cleared:
        action = "would clear" if dry_run else "cleared"
        print(f"  {action} {len(cleared)} scenario channel(s)")
    else:
        print("  scenario channels: none cleared")

    return len(collab_removed), len(cleared)
