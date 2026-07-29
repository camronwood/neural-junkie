"""Claude Human Simulator for SUT episodes (Claude CLI preferred, hub DM fallback)."""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any

from lib.sut_claude import DEFAULT_CLAUDE_AGENT, DEFAULT_TIMEOUT_S, ask_claude

HUMAN_DM_USER = "SutHumanSim"

_USER_MSG_RE = re.compile(
    r"(?:^|\n)\s*(?:USER_MESSAGE|NEXT_USER_MESSAGE)\s*:\s*(.+?)(?=\n\s*(?:STOP|DONE|REASON)\s*:|\Z)",
    re.I | re.S,
)
_STOP_RE = re.compile(r"(?:^|\n)\s*(?:STOP|DONE)\s*:\s*(.+)", re.I | re.S)


@dataclass
class HumanTurn:
    kind: str  # "message" | "stop" | "error"
    text: str
    raw: str = ""


def format_transcript(turns: list[dict[str, str]]) -> str:
    lines: list[str] = []
    for t in turns:
        role = (t.get("role") or "?").strip()
        content = (t.get("content") or "").strip()
        lines.append(f"{role}: {content}")
    return "\n\n".join(lines) if lines else "(no turns yet)"


def build_human_prompt(
    *,
    persona: str,
    goal: str,
    stop_when: str,
    max_turns: int,
    turn_index: int,
    transcript: list[dict[str, str]],
) -> str:
    stop = (stop_when or "").strip() or "Goal is satisfied or SUT clearly fails the probe."
    return (
        "You are the Human Simulator for Neural Junkie release-engineering SUT eval.\n"
        "You play a realistic user talking to a local specialist agent (the Subject Under Test).\n"
        "Do NOT solve the task for the SUT. Probe weaknesses: forgetfulness, format failure, "
        "hallucination, shallow answers, ignoring constraints.\n\n"
        f"Persona:\n{persona.strip()}\n\n"
        f"Goal:\n{goal.strip()}\n\n"
        f"Stop when:\n{stop}\n\n"
        f"Turn {turn_index} of max {max_turns}.\n\n"
        "Transcript so far:\n"
        f"{format_transcript(transcript)}\n\n"
        "Respond with EXACTLY one of these formats (no markdown fences):\n"
        "USER_MESSAGE: <single user chat message to send to the SUT>\n"
        "or\n"
        "STOP: <short reason the episode should end>\n"
    )


def parse_human_response(text: str) -> HumanTurn:
    raw = (text or "").strip()
    if not raw:
        return HumanTurn(kind="error", text="empty human-sim response", raw=raw)

    stop_m = _STOP_RE.search(raw)
    user_m = _USER_MSG_RE.search(raw)

    first_line = raw.splitlines()[0].strip().upper()
    if first_line.startswith("STOP") or first_line.startswith("DONE"):
        reason = stop_m.group(1).strip() if stop_m else raw
        reason = reason.splitlines()[0].strip()
        return HumanTurn(kind="stop", text=reason or "stopped", raw=raw)

    if user_m:
        msg = user_m.group(1).strip()
        if "\nSTOP:" in msg.upper() or "\nDONE:" in msg.upper():
            msg = re.split(r"\n\s*(?:STOP|DONE)\s*:", msg, maxsplit=1, flags=re.I)[0].strip()
        if msg:
            return HumanTurn(kind="message", text=msg, raw=raw)

    if stop_m:
        return HumanTurn(kind="stop", text=stop_m.group(1).strip().splitlines()[0], raw=raw)

    if len(raw) <= 800 and not raw.upper().startswith("PASS") and not raw.upper().startswith("FAIL"):
        return HumanTurn(kind="message", text=raw.splitlines()[0].strip(), raw=raw)

    return HumanTurn(kind="error", text=f"unparseable human-sim response: {raw[:200]}", raw=raw)


def next_user_turn(
    *,
    hub_base: str,
    human: dict[str, Any],
    transcript: list[dict[str, str]],
    turn_index: int,
    agent_name: str = DEFAULT_CLAUDE_AGENT,
    timeout_s: float = DEFAULT_TIMEOUT_S,
) -> HumanTurn:
    """Ask Claude for the next USER_MESSAGE or STOP."""
    max_turns = int(human.get("max_turns") or 4)
    prompt = build_human_prompt(
        persona=str(human.get("persona") or ""),
        goal=str(human.get("goal") or ""),
        stop_when=str(human.get("stop_when") or ""),
        max_turns=max_turns,
        turn_index=turn_index,
        transcript=transcript,
    )
    ok, text = ask_claude(
        prompt=prompt,
        hub_base=hub_base,
        dm_user=HUMAN_DM_USER,
        agent_name=agent_name,
        timeout_s=timeout_s,
        meta_flag="sut_human_sim",
    )
    if not ok:
        return HumanTurn(kind="error", text=text, raw=text)
    return parse_human_response(text)
