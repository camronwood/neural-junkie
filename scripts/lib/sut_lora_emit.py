"""Emit Alpaca-style LoRA rows from SUT eval failures (source_kind=sut_eval)."""

from __future__ import annotations

import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


SOURCE_KIND = "sut_eval"


def _row_id(*parts: str) -> str:
    blob = "|".join(p.strip() for p in parts)
    return hashlib.sha256(blob.encode("utf-8")).hexdigest()[:8]


def make_row(
    *,
    instruction: str,
    output: str,
    input_text: str = "",
    source_ref: str = "",
    message_at: str | None = None,
) -> dict[str, Any]:
    ts = message_at or datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    return {
        "row_id": _row_id(instruction, input_text, output, source_ref),
        "instruction": instruction.strip(),
        "input": (input_text or "").strip(),
        "output": output.strip(),
        "source_kind": SOURCE_KIND,
        "source_ref": source_ref.strip(),
        "message_at": ts,
        "included": True,
    }


def rows_from_failure(
    *,
    episode_name: str,
    transcript: list[dict[str, str]],
    gold_output: str,
    target_agent: str = "",
) -> list[dict[str, Any]]:
    """Build training rows from the last user turn + gold assistant output."""
    gold = (gold_output or "").strip()
    if not gold:
        return []

    last_user = ""
    prior: list[str] = []
    for t in transcript:
        role = (t.get("role") or "").strip().lower()
        content = (t.get("content") or "").strip()
        if role in ("user", "human"):
            last_user = content
        elif role in ("assistant", "sut", "agent") and content:
            prior.append(f"assistant: {content[:500]}")

    if not last_user:
        return []

    context = "\n".join(prior[-3:]) if prior else ""
    instruction = last_user
    if target_agent:
        instruction = f"[as @{target_agent}] {last_user}"

    return [
        make_row(
            instruction=instruction,
            input_text=context,
            output=gold,
            source_ref=f"sut:{episode_name}",
        )
    ]


def append_jsonl(path: Path, rows: list[dict[str, Any]]) -> int:
    if not rows:
        return 0
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")
    return len(rows)


def write_jsonl(path: Path, rows: list[dict[str, Any]]) -> int:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")
    return len(rows)
