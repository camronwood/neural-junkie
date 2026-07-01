"""Hub hygiene helpers for clean overnight / release-prep runs."""

from __future__ import annotations

import sys
import time
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib import collab_hub as hub  # noqa: E402
from lib.release_prep_env import apply_release_prep_env  # noqa: E402

# collab-preflight default roster + implement specialists commonly required.
DEFAULT_OVERNIGHT_ROSTER = (
    "Assistant",
    "BackendEngineer",
    "FrontendEngineer",
    "SoftwareArchitect",
    "PlatformEngineer",
)


def clear_pending_file_changes(base: str, *, user_id: str = "default") -> int:
    """Reject all pending file-change proposals; returns count cleared."""
    base = base.rstrip("/")
    pending = hub.list_pending_file_changes(base, user_id=user_id)
    cleared = 0
    for change in pending:
        if not isinstance(change, dict):
            continue
        cid = (change.get("id") or "").strip()
        if not cid:
            continue
        code, _ = hub.hub_request(
            base,
            "POST",
            f"/api/file-changes/reject/{cid}?user_id={user_id}",
        )
        if code == 200:
            cleared += 1
        else:
            print(f"  WARN: reject {cid[:8]}… HTTP {code}", file=sys.stderr)
    return cleared


def wait_for_agent_roster(
    base: str,
    required: list[str] | tuple[str, ...] | None = None,
    *,
    timeout_s: float = 180.0,
    poll_s: float = 3.0,
) -> tuple[bool, list[str]]:
    """Poll until required agent display names are online (not paused)."""
    want = list(required or DEFAULT_OVERNIGHT_ROSTER)
    deadline = time.time() + timeout_s
    last_missing: list[str] = want
    while time.time() < deadline:
        ok, missing = hub.verify_agents_online(base, want)
        if ok:
            return True, []
        last_missing = missing
        time.sleep(poll_s)
    return False, last_missing


def clean_hub_for_regression(
    root: Path,
    hub_url: str,
    *,
    label: str = "overnight clean",
) -> bool:
    """Fixture reset, channel cleanup, and pending file-change rejection."""
    from lib.fixture_cleanup import preflight_regression_run  # noqa: E402

    apply_release_prep_env(root)
    base = hub_url.rstrip("/")
    if hub.check_health(base) is None:
        print(f"FAIL: hub not healthy at {base}", file=sys.stderr)
        return False

    n_pending = clear_pending_file_changes(base)
    print(f"  rejected {n_pending} pending file change(s)")

    preflight_regression_run(root, base, label=label)
    return True
