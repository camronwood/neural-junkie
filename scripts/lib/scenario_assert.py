"""Shared text-pattern assertions for collab and chat scenario runners."""

from __future__ import annotations

import re

# Mirrors internal/protocol/command_context.go stack-tool prefixes for scenario checks.
_STACK_CMD_HEAD_RE = re.compile(
    r"^\s*(docker(?:-compose)?|compose|npm|yarn|pnpm|npx|kubectl|helm|terraform|make|mvn|gradle)\b",
    re.I,
)


def looks_like_stack_tool_command(command: str) -> bool:
    command = (command or "").strip()
    if not command:
        return False
    first_line = command.split("\n", 1)[0].strip()
    return bool(_STACK_CMD_HEAD_RE.match(first_line))


_READ_ONLY_PREFIXES = (
    "ls",
    "pwd",
    "cd ",
    "cat ",
    "head ",
    "tail ",
    "grep ",
    "find ",
    "file ",
    "tree",
    "wc ",
    "sort ",
    "uniq ",
    "which ",
    "whereis ",
    "git status",
    "git log",
    "git diff",
    "git show",
    "git branch",
)


def looks_like_read_only_inspection_command(command: str) -> bool:
    command = (command or "").strip().lower()
    if not command:
        return False
    return any(command.startswith(prefix) for prefix in _READ_ONLY_PREFIXES)


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
