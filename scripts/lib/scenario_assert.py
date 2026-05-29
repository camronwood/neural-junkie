"""Shared text-pattern assertions for collab and chat scenario runners."""

from __future__ import annotations

import re


def check_text_patterns(
    text: str,
    *,
    any_match: list[str] | None = None,
    none_match: list[str] | None = None,
    label: str = "content",
) -> tuple[bool, str]:
    for pattern in none_match or []:
        if re.search(pattern, text, re.I | re.MULTILINE):
            return False, f"{label} none_match {pattern!r}"
    if any_match:
        if not any(re.search(p, text, re.I | re.MULTILINE) for p in any_match):
            return False, f"{label} any_match not found (want one of {any_match!r})"
    return True, "ok"
