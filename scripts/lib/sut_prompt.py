"""Build Cursor fix-agent briefs for SUT self-improve failures."""

from __future__ import annotations

from pathlib import Path
from typing import Any

from lib.sut_episode import fix_allow_paths
from lib.sut_human import format_transcript
from lib.sut_judge import JudgeVerdict


def build_sut_agent_prompt(
    *,
    episode: dict[str, Any],
    episode_name: str,
    verdict: JudgeVerdict,
    transcript: list[dict[str, str]],
    report_path: Path | None = None,
) -> str:
    allow = fix_allow_paths(episode)
    allow_block = "\n".join(f"- `{p}`" for p in allow)
    report_line = f"Episode report: `{report_path}`\n" if report_path else ""
    score = f"{verdict.score:.2f}" if verdict.score is not None else "n/a"

    return (
        "# Neural Junkie SUT self-improve fix brief\n\n"
        "You are the Fix Engineer for an internal release-engineering loop.\n"
        "The Subject Under Test is a **local** Neural Junkie specialist (not Claude).\n"
        "Fix product/hub/agent behavior so this episode passes on re-run.\n\n"
        "## Hard rules\n"
        "- Do **not** weaken tests or scenario assertions.\n"
        "- Do **not** change the SUT episode JSON to make it easier unless the episode is clearly broken.\n"
        f"- Prefer edits under these paths:\n{allow_block}\n"
        "- Keep changes minimal and explain what you changed.\n"
        "- After edits, summarize verification you ran (or could not run).\n\n"
        f"## Episode\n"
        f"- name: `{episode_name}`\n"
        f"- target_agent: `@{episode.get('target_agent')}`\n"
        f"{report_line}\n"
        f"## Judge verdict\n"
        f"- result: FAIL\n"
        f"- score: {score}\n"
        f"- failure_kind: `{verdict.failure_kind}`\n"
        f"- reason: {verdict.reason}\n\n"
        f"## Gold output (if any)\n"
        f"{(verdict.gold_output or '(none)').strip()}\n\n"
        f"## Transcript\n"
        f"{format_transcript(transcript)}\n\n"
        "## Suggested fix tiers\n"
        "1. System prompt / routing / context trim for the target agent\n"
        "2. Runtime params / reply handling bugs\n"
        "3. If the model simply lacks knowledge, note that LoRA rows were emitted; "
        "still fix harness or prompt bugs if present\n"
    )
