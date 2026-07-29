"""Claude Judge for SUT episodes (Claude CLI preferred, hub DM fallback)."""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Any

from lib.sut_claude import DEFAULT_CLAUDE_AGENT, DEFAULT_TIMEOUT_S, ask_claude
from lib.sut_human import format_transcript

JUDGE_DM_USER = "SutJudge"

_SCORE_RE = re.compile(r"^SCORE:\s*([0-9]*\.?[0-9]+)\s*$", re.I)
_KIND_RE = re.compile(r"^FAILURE_KIND:\s*(.+)$", re.I)
_GOLD_RE = re.compile(
    r"(?:^|\n)\s*GOLD_OUTPUT\s*:\s*(.+?)(?=\n\s*(?:REASON|PASS|FAIL|SCORE|FAILURE_KIND)\s*:|\Z)",
    re.I | re.S,
)
_REASON_RE = re.compile(r"(?:^|\n)\s*REASON\s*:\s*(.+)", re.I | re.S)


@dataclass
class JudgeVerdict:
    passed: bool
    reason: str
    score: float | None = None
    failure_kind: str = ""
    gold_output: str = ""
    raw: str = ""
    failed_turn_indices: list[int] = field(default_factory=list)


def build_judge_prompt(
    *,
    rubric: str,
    require_gold_on_fail: bool,
    target_agent: str,
    transcript: list[dict[str, str]],
    metrics_note: str = "",
) -> str:
    gold_line = (
        "If FAIL, include GOLD_OUTPUT: <corrected assistant reply the SUT should have given>\n"
        if require_gold_on_fail
        else "If FAIL, optionally include GOLD_OUTPUT: <corrected assistant reply>\n"
    )
    metrics = (metrics_note or "").strip() or "(none)"
    return (
        "You are the Judge for Neural Junkie release-engineering SUT eval.\n"
        f"Subject Under Test agent: @{target_agent} (local model — not you).\n"
        "Evaluate correctness, constraint following, multi-turn memory, and usefulness.\n"
        "Do not grade yourself; grade only the SUT assistant turns.\n\n"
        f"Rubric:\n{rubric.strip()}\n\n"
        f"Inference / telemetry notes:\n{metrics}\n\n"
        "Transcript:\n"
        f"{format_transcript(transcript)}\n\n"
        "Respond with EXACTLY this format (no markdown fences):\n"
        "PASS or FAIL\n"
        "SCORE: <0.00-1.00>\n"
        "FAILURE_KIND: <none|correctness|memory|format|hallucination|performance|other>\n"
        f"{gold_line}"
        "REASON: <one short paragraph>\n"
    )


def parse_judge_response(text: str) -> JudgeVerdict:
    raw = (text or "").strip()
    if not raw:
        return JudgeVerdict(passed=False, reason="empty judge response", raw=raw)

    lines = [ln.strip() for ln in raw.splitlines() if ln.strip()]
    first = (lines[0] if lines else raw).upper()
    passed = bool(re.search(r"\bPASS\b", first)) and not first.startswith("FAIL")
    if first.startswith("FAIL") or re.search(r"\bFAIL\b", first):
        passed = False
    elif first.startswith("PASS"):
        passed = True
    else:
        return JudgeVerdict(passed=False, reason=f"unparseable judge response: {raw[:200]}", raw=raw)

    score: float | None = None
    failure_kind = "none" if passed else "other"
    for ln in lines[1:6]:
        m = _SCORE_RE.match(ln)
        if m:
            try:
                score = max(0.0, min(1.0, float(m.group(1))))
            except ValueError:
                score = None
            continue
        km = _KIND_RE.match(ln)
        if km:
            failure_kind = km.group(1).strip().lower() or failure_kind

    gold = ""
    gm = _GOLD_RE.search(raw)
    if gm:
        gold = gm.group(1).strip()

    reason = ""
    rm = _REASON_RE.search(raw)
    if rm:
        reason = rm.group(1).strip().splitlines()[0]
    elif len(lines) > 1:
        reason = lines[-1]

    return JudgeVerdict(
        passed=passed,
        reason=reason or ("pass" if passed else "fail"),
        score=score,
        failure_kind=failure_kind if not passed else "none",
        gold_output=gold,
        raw=raw,
    )


def judge_episode(
    *,
    hub_base: str,
    episode: dict[str, Any],
    transcript: list[dict[str, str]],
    metrics_note: str = "",
    agent_name: str = DEFAULT_CLAUDE_AGENT,
    timeout_s: float = DEFAULT_TIMEOUT_S,
) -> JudgeVerdict:
    judge = episode.get("judge") if isinstance(episode.get("judge"), dict) else {}
    prompt = build_judge_prompt(
        rubric=str(judge.get("rubric") or ""),
        require_gold_on_fail=bool(judge.get("require_gold_on_fail", True)),
        target_agent=str(episode.get("target_agent") or "SUT"),
        transcript=transcript,
        metrics_note=metrics_note,
    )
    ok, text = ask_claude(
        prompt=prompt,
        hub_base=hub_base,
        dm_user=JUDGE_DM_USER,
        agent_name=agent_name,
        timeout_s=timeout_s,
        meta_flag="sut_judge",
    )
    if not ok:
        return JudgeVerdict(passed=False, reason=text, failure_kind="infra", raw=text)
    return parse_judge_response(text)
