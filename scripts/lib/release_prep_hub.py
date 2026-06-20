"""Hub readiness for release-prep (health, cloud-first judge smoke, optional restart)."""

from __future__ import annotations

import sys
import time
from pathlib import Path

from lib import collab_hub as hub
from lib.deliverable_judge_smoke import check_deliverable_judge_smoke
from lib.hub_regression import restart_regression_hub, start_regression_hub, wait_for_hub
from lib.release_prep_env import apply_release_prep_env, release_prep_env

ROOT = Path(__file__).resolve().parents[2]


def ensure_hub_for_release_prep(
    hub_url: str,
    *,
    root: Path = ROOT,
    allow_restart: bool = True,
    verbose: bool = False,
) -> bool:
    """Verify hub health + deliverable judge (cloud-first, Ollama fallback)."""
    env = apply_release_prep_env(root)
    base = hub_url.rstrip("/")

    for line in summarize_lines(env):
        print(f"  {line}")

    def hub_ready() -> bool:
        try:
            return hub.check_health(base)
        except Exception:
            return False

    if not hub_ready():
        print(f"Hub not healthy at {base}")
        if not allow_restart:
            print("FAIL: hub down (--no-restart-hub)", file=sys.stderr)
            return False
        print("Starting regression hub (make server-regression)...")
        if start_regression_hub(root, env=env) is None:
            print("FAIL: could not start regression hub", file=sys.stderr)
            return False
        if not wait_for_hub(base, timeout_s=120.0):
            print("FAIL: hub did not become healthy after start", file=sys.stderr)
            return False
        print(f"OK: hub healthy at {base}")
        time.sleep(3.0)
    else:
        print(f"OK: hub healthy at {base}")

    jok, jdetail = check_deliverable_judge_smoke(base, timeout_s=90.0)
    if jok:
        if "ollama fallback" in jdetail.lower():
            print(f"WARN: {jdetail}")
        else:
            print(f"OK: {jdetail}")
        return True

    print(f"WARN: deliverable judge smoke failed: {jdetail}")
    if not allow_restart:
        print("FAIL: judge not ready (--no-restart-hub)", file=sys.stderr)
        return False

    print("Restarting regression hub (pick up API key / agent env)...")
    if not restart_regression_hub(root, base, timeout_s=120.0, env=env):
        print("FAIL: hub restart failed", file=sys.stderr)
        return False
    print(f"OK: hub restarted at {base}")
    time.sleep(3.0)

    jok, jdetail = check_deliverable_judge_smoke(base, timeout_s=90.0)
    if jok:
        if "ollama fallback" in jdetail.lower():
            print(f"WARN: {jdetail}")
        else:
            print(f"OK: {jdetail}")
        return True

    print(f"FAIL: deliverable judge still failing: {jdetail}", file=sys.stderr)
    print("Hint: ollama serve && ollama pull qwen2.5-coder:14b", file=sys.stderr)
    return False


def summarize_lines(env: dict[str, str]) -> list[str]:
    from lib.release_prep_env import summarize_release_prep_env

    return summarize_release_prep_env(env)
