"""Detect live scenario flakes and optionally retry once."""

from __future__ import annotations

import os
import re

# Implement harness: wait_reply timeouts on loaded Ollama / frontend sessions.
IMPLEMENT_FLAKE_MARKERS = (
    "timeout waiting for",
    "encountered an error while generating",
    "timed out before completion",
    "provider_error",
)

# Collab harness: silent agents and generation failures during planning discussion.
COLLAB_FLAKE_MARKERS = (
    "no collaboration_discussion",
    "silent or shouldrespond blocked",
    "generation_error",
    "wait_discussion attempt",
    "could not complete this turn",
)

SHARED_FLAKE_MARKERS = (
    "sorry, i encountered an error",
    "remote disconnected",
    "connection reset",
    "quota",
    "rate limit",
    "429",
)


def _normalized(text: str) -> str:
    return (text or "").lower()


def detail_is_flake(detail: str, *, kind: str = "any") -> bool:
    """True when a single-step failure detail looks like load/model variance."""
    lower = _normalized(detail)
    pools: list[tuple[str, ...]] = [SHARED_FLAKE_MARKERS]
    if kind in ("any", "implement"):
        pools.append(IMPLEMENT_FLAKE_MARKERS)
    if kind in ("any", "collab"):
        pools.append(COLLAB_FLAKE_MARKERS)
    markers = tuple(m for pool in pools for m in pool)
    return any(m in lower for m in markers)


def output_is_flake(output: str, *, kind: str = "any") -> bool:
    """True when combined stage output contains flake signatures."""
    if detail_is_flake(output, kind=kind):
        return True
    lower = _normalized(output)
    if kind in ("any", "implement") and re.search(
        r"wait_reply:\s*timeout waiting for", lower
    ):
        return True
    if kind in ("any", "collab") and "fail: @" in lower and "no collaboration_discussion" in lower:
        return True
    return False


def flake_retry_enabled() -> bool:
    raw = (os.environ.get("NJ_SCENARIO_FLAKE_RETRY") or "1").strip().lower()
    return raw not in ("0", "false", "no", "off")


def flake_retry_sleep_s() -> float:
    try:
        return float(os.environ.get("NJ_SCENARIO_FLAKE_RETRY_SLEEP_S", "8"))
    except ValueError:
        return 8.0
