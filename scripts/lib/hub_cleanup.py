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

REGRESSION_ROSTER_TYPES = {
    "Assistant": "assistant",
    "BackendEngineer": "backend",
    "FrontendEngineer": "frontend",
    "SoftwareArchitect": "architecture",
    "PlatformEngineer": "devops",
    "SecurityReviewer": "security",
}


def merge_roster_enabled(agents: list[dict], names: list[str] | tuple[str, ...]) -> tuple[list[dict], bool]:
    """Return agents with required names enabled. Does not mutate the input list."""
    out = [dict(a) for a in agents if isinstance(a, dict)]
    by_name = {str(a.get("name") or ""): a for a in out}
    changed = False
    for name in names:
        if name == "Claude":
            continue
        row = by_name.get(name)
        if row is None:
            typ = REGRESSION_ROSTER_TYPES.get(name)
            if not typ:
                continue
            row = {"type": typ, "name": name, "enabled": True}
            out.append(row)
            by_name[name] = row
            changed = True
        elif not row.get("enabled"):
            row["enabled"] = True
            changed = True
    return out, changed


def ensure_regression_roster_enabled(
    base: str,
    names: list[str] | tuple[str, ...] | None = None,
) -> tuple[bool, str]:
    """Enable required specialists in hub settings so wait_for_roster can see them."""
    want = list(names or DEFAULT_OVERNIGHT_ROSTER)
    code, cfg = hub.hub_request(base, "GET", "/api/settings")
    if code != 200 or not isinstance(cfg, dict):
        return False, f"GET /api/settings failed: {code}"
    agents = cfg.get("agents") or []
    if not isinstance(agents, list):
        agents = []
    merged, changed = merge_roster_enabled(agents, want)
    if not changed:
        return True, "roster already enabled"
    cfg["agents"] = merged
    code, out = hub.hub_request(base, "PUT", "/api/settings", cfg)
    if code != 200:
        return False, f"PUT /api/settings failed: {code} {out}"
    code, _ = hub.hub_request(base, "POST", "/api/agents/restart")
    if code != 200:
        return False, f"POST /api/agents/restart failed: {code}"
    return True, "enabled " + ", ".join(want)


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
