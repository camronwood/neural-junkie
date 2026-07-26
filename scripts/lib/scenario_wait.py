"""Shared wait_reply helpers: disk + metadata completion (not chat-phrase gates)."""

from __future__ import annotations

from pathlib import Path
from typing import Any

try:
    from scenario_assert import check_text_patterns
except ImportError:  # imported as lib.scenario_wait (scripts/ on sys.path)
    from lib.scenario_assert import check_text_patterns


def metadata_get(meta: dict, dotted: str) -> Any:
    cur: object = meta
    for part in dotted.split("."):
        if not isinstance(cur, dict):
            return None
        cur = cur.get(part)
    return cur


def step_has_disk_wait(step: dict) -> bool:
    return bool(
        step.get("until_file_exists")
        or step.get("until_files_exist")
        or step.get("until_file_absent")
        or step.get("until_files_absent")
        or step.get("until_file_match")
    )


def normalize_meta_keys(step: dict) -> list[str]:
    keys = step.get("until_metadata_keys") or []
    if isinstance(keys, str):
        keys = [keys]
    return [str(k).strip() for k in keys if str(k).strip()]


def disk_wait_satisfied(root: Path, step: dict) -> tuple[bool, str]:
    """AND all configured disk wait conditions. Empty config → (True, "")."""
    until_files = step.get("until_file_exists") or step.get("until_files_exist") or []
    if isinstance(until_files, str):
        until_files = [until_files]
    until_files = [str(p).strip() for p in until_files if str(p).strip()]

    until_absent = step.get("until_file_absent") or step.get("until_files_absent") or []
    if isinstance(until_absent, str):
        until_absent = [until_absent]
    until_absent = [str(p).strip() for p in until_absent if str(p).strip()]

    until_file_match = step.get("until_file_match")
    if until_file_match is not None and not isinstance(until_file_match, dict):
        until_file_match = None

    details: list[str] = []
    has_disk = bool(until_files or until_absent or until_file_match)
    if not has_disk:
        return True, ""

    for rel in until_files:
        path = (root / rel).resolve()
        if not (path.is_file() and path.stat().st_size > 0):
            return False, f"waiting for file {rel}"
        details.append(f"exists:{rel}")

    for rel in until_absent:
        path = (root / rel).resolve()
        if path.exists():
            return False, f"waiting for absence of {rel}"
        details.append(f"absent:{rel}")

    if until_file_match:
        rel = str(until_file_match.get("path") or "").strip()
        if not rel:
            return False, "until_file_match.path required"
        alts = until_file_match.get("path_alternatives") or []
        if isinstance(alts, str):
            alts = [alts]
        candidates = [rel] + [str(a).strip() for a in alts if str(a).strip()]
        body = ""
        matched_rel = ""
        for candidate in candidates:
            path = (root / candidate).resolve()
            if not path.is_file():
                continue
            try:
                body = path.read_text(encoding="utf-8", errors="replace")
            except OSError:
                continue
            matched_rel = candidate
            break
        if not matched_rel:
            return False, f"waiting for file {rel}"
        contains = str(until_file_match.get("contains") or "")
        if contains and contains not in body:
            return False, f"file:{matched_rel} missing contains {contains!r}"
        ok, detail = check_text_patterns(
            body,
            any_match=until_file_match.get("any_match"),
            none_match=until_file_match.get("none_match"),
            label=f"file:{matched_rel}",
        )
        if ok:
            for needle in until_file_match.get("contains_all") or []:
                if str(needle) not in body:
                    ok = False
                    detail = f"file:{matched_rel} missing {needle!r}"
                    break
        if not ok:
            return False, detail
        details.append(f"match:{matched_rel}")

    return True, "; ".join(details)
