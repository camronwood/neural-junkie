"""Hub readiness for release-prep (health, cloud-first judge smoke, optional restart)."""

from __future__ import annotations

import os
import sys
import time
from pathlib import Path

from lib import collab_hub as hub
from lib.deliverable_judge_smoke import check_deliverable_judge_smoke
from lib.gemini_judge_auth import select_gemini_judge_model
from lib.hub_regression import restart_regression_hub, start_regression_hub, wait_for_hub
from lib.release_prep_env import explicit_gemini_judge_model, release_prep_env

ROOT = Path(__file__).resolve().parents[2]


def ensure_hub_for_release_prep(
    hub_url: str,
    *,
    root: Path = ROOT,
    allow_restart: bool = True,
    verbose: bool = False,
) -> bool:
    """Verify hub health + deliverable judge (cloud-first, Ollama fallback)."""
    explicit_gemini = explicit_gemini_judge_model(root)
    env = release_prep_env(root)
    os.environ.update(env)

    provider = (env.get("NJ_DELIVERABLE_JUDGE_PROVIDER") or "claude").strip().lower()
    hub_needs_model_restart = False
    if provider == "claude":
        from lib.claude_judge_auth import ensure_claude_for_testing

        sel = ensure_claude_for_testing(timeout_s=15.0, smoke=False)
        if sel.ok:
            print(f"OK: Claude judge probe → {sel.detail}")
        else:
            print(f"WARN: Claude judge probe failed: {sel.detail}")
    elif provider == "gemini" and env.get("GEMINI_API_KEY"):
        model, probe_ok, probe_detail = select_gemini_judge_model(
            timeout_s=30.0,
            explicit_model=explicit_gemini,
            retry_quota=False,
        )
        if probe_ok and model:
            env["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = model
            os.environ["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = model
            hub_needs_model_restart = True
            print(f"OK: Gemini judge probe → {model} ({probe_detail})")
        elif explicit_gemini:
            print(f"WARN: configured Gemini judge model failed: {probe_detail}")
        else:
            print(f"WARN: Gemini judge probe failed: {probe_detail}")

    base = hub_url.rstrip("/")

    for line in summarize_lines(env):
        print(f"  {line}")

    def hub_ready() -> bool:
        try:
            return hub.check_health(base)
        except Exception:
            return False

    hub_was_running = hub_ready()
    if not hub_was_running:
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

    if hub_needs_model_restart and hub_was_running and allow_restart:
        print("Restarting regression hub (apply selected Gemini judge model)...")
        if not restart_regression_hub(root, base, timeout_s=120.0, env=env):
            print("FAIL: hub restart failed after Gemini judge probe", file=sys.stderr)
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
