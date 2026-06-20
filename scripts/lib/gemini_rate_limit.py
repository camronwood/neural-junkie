"""Pacing for Gemini API free-tier RPM limits (shared by judge + smokes)."""

from __future__ import annotations

import os
import time

_last_gemini_api_call_at: float = 0.0


def gemini_min_interval_s() -> float | None:
    raw = os.environ.get("NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S", "").strip()
    if not raw:
        return None
    try:
        value = float(raw)
    except ValueError:
        return None
    return value if value > 0 else None


def throttle_gemini_api_call() -> None:
    """Sleep so consecutive Gemini API calls stay under free-tier RPM."""
    min_interval = gemini_min_interval_s()
    if min_interval is None:
        return
    global _last_gemini_api_call_at
    now = time.monotonic()
    if _last_gemini_api_call_at > 0:
        wait = min_interval - (now - _last_gemini_api_call_at)
        if wait > 0:
            time.sleep(wait)
    _last_gemini_api_call_at = time.monotonic()
