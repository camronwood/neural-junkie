"""Ollama smoke test for local deliverable judging."""

from __future__ import annotations

import os

PROMPT = (
    "Reply with exactly three lines:\n"
    "Line 1: PASS\n"
    "Line 2: SCORE: 1.00\n"
    "Line 3: ok"
)


def resolve_ollama_judge_model() -> str:
    return (os.environ.get("NJ_DELIVERABLE_JUDGE_MODEL") or "qwen2.5-coder:14b").strip()


def check_ollama_judge(*, timeout_s: float = 90.0) -> tuple[bool, str]:
    try:
        from lib.deliverable_judge import ollama_judge_deliverable
    except ImportError:
        from deliverable_judge import ollama_judge_deliverable  # type: ignore[no-redef]

    model = resolve_ollama_judge_model()
    ok, detail, _score = ollama_judge_deliverable(
        prompt=PROMPT,
        model=model,
        timeout_s=timeout_s,
    )
    if ok:
        return True, f"ollama/{model} judge OK ({detail})"
    return False, f"ollama/{model}: {detail}"
