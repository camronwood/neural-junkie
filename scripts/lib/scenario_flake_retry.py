"""One automatic retry for known-flaky live scenario outcomes (timeouts, 401, silence)."""

from __future__ import annotations

import os
import time

RETRY_MARKERS = (
    "timeout waiting for",
    "send failed (401)",
    "send failed (0)",
    "approve-plan failed",
    "no collaboration_discussion",
    "silent or shouldrespond blocked",
    "could not complete this turn",
    "generation_error",
    "hub not healthy",
)

DEFAULT_PAUSE_S = 5.0


def flake_retry_enabled() -> bool:
    return os.environ.get("NJ_SCENARIO_FLAKE_RETRY", "1").strip().lower() not in (
        "0",
        "false",
        "no",
        "off",
    )


def is_retryable_failure(detail: str) -> bool:
    if not detail.strip():
        return False
    lower = detail.lower()
    return any(marker.lower() in lower for marker in RETRY_MARKERS)


def pause_before_retry(pause_s: float = DEFAULT_PAUSE_S) -> None:
    time.sleep(max(0.0, pause_s))


def refresh_auth_for_retry(hub_url: str) -> None:
    from lib.hub_auth import refresh_hub_auth_after_restart

    refresh_hub_auth_after_restart(hub_url.rstrip("/"))


def maybe_retry_after_failure(
    hub_url: str,
    scenario_name: str,
    detail: str,
    attempt: int,
    *,
    max_attempts: int = 2,
) -> bool:
    """Return True when the caller should run the scenario again."""
    if attempt >= max_attempts:
        return False
    if not flake_retry_enabled():
        return False
    if not is_retryable_failure(detail):
        return False
    print(
        f"\n>>> flake retry {attempt + 1}/{max_attempts - 1} for {scenario_name}: {detail[:160]}",
        flush=True,
    )
    refresh_auth_for_retry(hub_url)
    pause_before_retry()
    return True
