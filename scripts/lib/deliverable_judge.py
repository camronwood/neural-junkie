"""Cloud/local judges for scenario deliverable quality (independent of producing agent)."""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import time
import urllib.error
import urllib.request
from typing import Any

DEFAULT_JUDGE_PROVIDER = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "gemini").strip().lower()
DEFAULT_JUDGE_AGENT = os.environ.get("NJ_DELIVERABLE_JUDGE_AGENT", "").strip()
DEFAULT_OLLAMA_URL = os.environ.get("OLLAMA_HOST", "http://127.0.0.1:11434").rstrip("/")
DEFAULT_OLLAMA_MODEL = os.environ.get("NJ_DELIVERABLE_JUDGE_MODEL", "qwen3.5:9b")
JUDGE_MAX_CHARS = int(os.environ.get("NJ_DELIVERABLE_JUDGE_MAX_CHARS", "12000"))
JUDGE_TIMEOUT_S = float(os.environ.get("NJ_DELIVERABLE_JUDGE_TIMEOUT", "180"))
JUDGE_DM_USER = os.environ.get("NJ_DELIVERABLE_JUDGE_DM_USER", "DeliverableJudge").strip()

_PROVIDER_DEFAULT_AGENT = {
    "gemini": "Gemini",
    "cursor": "Cursor",
    "ollama": "",
}


def judge_skip_enabled() -> bool:
    return os.environ.get("NJ_DELIVERABLE_JUDGE_SKIP", "").strip().lower() in ("1", "true", "yes")


def resolve_judge_agent(provider: str, spec_agent: str = "") -> str:
    if spec_agent.strip():
        return spec_agent.strip().lstrip("@")
    if DEFAULT_JUDGE_AGENT:
        return DEFAULT_JUDGE_AGENT.lstrip("@")
    return _PROVIDER_DEFAULT_AGENT.get(provider, "Gemini")


def build_judge_prompt(
    *,
    question: str,
    rel_path: str,
    file_body: str,
    criteria: str = "",
    max_chars: int = JUDGE_MAX_CHARS,
) -> str:
    body = file_body if len(file_body) <= max_chars else file_body[:max_chars] + "\n…[truncated]"
    extra = f"\nAdditional criteria:\n{criteria}\n" if criteria.strip() else ""
    return (
        "You are an independent automated test harness judge. You did NOT produce this file.\n"
        "Evaluate whether the deliverable substantively answers the user's request.\n\n"
        "The user asked:\n---\n"
        f"{question.strip()}\n"
        "---\n\n"
        f'The deliverable file is "{rel_path}" with this content:\n---\n'
        f"{body}\n"
        f"---{extra}\n\n"
        "Reject stubs, placeholders, unrelated boilerplate, or wrong-stack artifacts.\n\n"
        "Reply with exactly two lines:\n"
        "Line 1: PASS or FAIL\n"
        "Line 2: one short reason"
    )


def parse_judge_response(text: str) -> tuple[bool, str]:
    response = (text or "").strip()
    if not response:
        return False, "empty judge response"
    lines = [ln.strip() for ln in response.splitlines() if ln.strip()]
    first = (lines[0] if lines else response).upper()
    reason = lines[1] if len(lines) > 1 else response
    if first.startswith("PASS") or re.search(r"\bPASS\b", first):
        return True, reason or "pass"
    if first.startswith("FAIL") or re.search(r"\bFAIL\b", first):
        return False, reason or "fail"
    return False, f"unparseable judge response: {response[:200]}"


def _judge_spec_provider(spec: dict[str, Any] | bool | None) -> str:
    if isinstance(spec, dict):
        p = (spec.get("provider") or "").strip().lower()
        if p:
            return p
    mode = os.environ.get("NJ_DELIVERABLE_JUDGE_MODE", "").strip().lower()
    if mode in ("cli", "hub", "ollama"):
        if mode == "cli":
            return DEFAULT_JUDGE_PROVIDER if DEFAULT_JUDGE_PROVIDER in ("gemini", "cursor") else "gemini"
        if mode == "ollama":
            return "ollama"
    return DEFAULT_JUDGE_PROVIDER if DEFAULT_JUDGE_PROVIDER in ("gemini", "cursor", "ollama") else "gemini"


def _judge_spec_agent(spec: dict[str, Any] | bool | None, provider: str) -> str:
    if isinstance(spec, dict):
        agent = (spec.get("agent") or "").strip()
        if agent:
            return agent.lstrip("@")
    return resolve_judge_agent(provider)


def _judge_spec_criteria(spec: dict[str, Any] | bool | None) -> str:
    if isinstance(spec, dict):
        return (spec.get("criteria") or "").strip()
    return ""


def ollama_judge_deliverable(
    *,
    prompt: str,
    model: str = DEFAULT_OLLAMA_MODEL,
    ollama_base: str = DEFAULT_OLLAMA_URL,
    timeout_s: float = JUDGE_TIMEOUT_S,
) -> tuple[bool, str]:
    payload = {
        "model": model,
        "prompt": prompt,
        "stream": False,
        "options": {"temperature": 0},
    }
    url = f"{ollama_base.rstrip('/')}/api/generate"
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except urllib.error.URLError as exc:
        return False, f"ollama unreachable at {ollama_base}: {exc}"
    except TimeoutError:
        return False, f"ollama judge timeout ({timeout_s}s)"
    return parse_judge_response((data.get("response") or "").strip())


def cli_judge_deliverable(
    *,
    provider: str,
    prompt: str,
    work_dir: str = "",
    timeout_s: float = JUDGE_TIMEOUT_S,
) -> tuple[bool, str]:
    if provider == "gemini":
        binary = shutil.which("gemini")
        if not binary:
            return False, "gemini CLI not on PATH"
        cmd = [binary, "--output-format", "text", "-p", prompt]
    elif provider == "cursor":
        binary = shutil.which("agent")
        if not binary:
            return False, "cursor agent CLI not on PATH"
        cmd = [binary, "-p", "--output-format", "text", prompt]
    else:
        return False, f"unsupported CLI judge provider {provider!r}"

    try:
        proc = subprocess.run(
            cmd,
            cwd=work_dir or None,
            capture_output=True,
            text=True,
            timeout=timeout_s,
            check=False,
        )
    except subprocess.TimeoutExpired:
        return False, f"{provider} CLI judge timeout ({timeout_s}s)"
    except OSError as exc:
        return False, f"{provider} CLI judge failed: {exc}"

    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()
        return False, f"{provider} CLI exit {proc.returncode}: {err[:300]}"
    return parse_judge_response(proc.stdout or "")


def hub_judge_deliverable(
    *,
    hub_base: str,
    agent_name: str,
    prompt: str,
    timeout_s: float = JUDGE_TIMEOUT_S,
) -> tuple[bool, str]:
    try:
        from lib import collab_hub as hub
    except ImportError:  # unittest cwd scripts/lib
        import collab_hub as hub  # type: ignore[no-redef]

    base = hub_base.rstrip("/")
    ok, missing = hub.verify_agents_online(base, [agent_name])
    if not ok:
        return False, f"judge agent offline: {', '.join(missing)}"

    channel = hub.ensure_dm_channel(base, JUDGE_DM_USER, agent_name)
    if not channel:
        return False, f"could not open DM with judge agent {agent_name!r}"

    hub.clear_channel_history(base, channel)
    time.sleep(2.0)

    code, _ = hub.send_message(
        base,
        channel,
        prompt,
        from_name=JUDGE_DM_USER,
        metadata={
            "editor_mode": "ask",
            "conversation_mode": "chat",
            "deliverable_judge": True,
        },
    )
    if code != 200:
        return False, f"judge send failed ({code})"

    deadline = time.time() + timeout_s
    while time.time() < deadline:
        msgs = hub.list_messages(base, channel, 30)
        for msg in reversed(msgs):
            sender = msg.get("from") if isinstance(msg.get("from"), dict) else {}
            if (sender.get("name") or "").strip() != agent_name:
                continue
            if (sender.get("type") or "").strip().lower() == "human":
                continue
            text = (msg.get("content") or "").strip()
            if text:
                return parse_judge_response(text)
        time.sleep(2.0)
    return False, f"timeout waiting for {agent_name} judge ({timeout_s}s)"


def judge_deliverable(
    *,
    question: str,
    rel_path: str,
    file_body: str,
    criteria: str = "",
    llm_judge_spec: dict[str, Any] | bool | None = True,
    hub_base: str = "",
    work_dir: str = "",
) -> tuple[bool, str]:
    """Route deliverable judging to an independent cloud agent (default: hub Gemini)."""
    if judge_skip_enabled():
        return True, "skipped (NJ_DELIVERABLE_JUDGE_SKIP)"

    provider = _judge_spec_provider(llm_judge_spec)
    agent = _judge_spec_agent(llm_judge_spec, provider)
    criteria = criteria or _judge_spec_criteria(llm_judge_spec)
    prompt = build_judge_prompt(
        question=question,
        rel_path=rel_path,
        file_body=file_body,
        criteria=criteria,
    )

    mode = os.environ.get("NJ_DELIVERABLE_JUDGE_MODE", "hub").strip().lower()
    if mode not in ("hub", "cli", "ollama"):
        mode = "hub"

    if provider == "ollama" or mode == "ollama":
        ok, detail = ollama_judge_deliverable(prompt=prompt)
        return ok, f"ollama/{DEFAULT_OLLAMA_MODEL}: {detail}"

    if mode == "cli" or not hub_base.strip():
        ok, detail = cli_judge_deliverable(provider=provider, prompt=prompt, work_dir=work_dir)
        if ok or hub_base.strip():
            prefix = f"{provider}-cli"
            return ok, f"{prefix}: {detail}"
        # fall through to hub when CLI fails and hub is available

    if hub_base.strip():
        ok, detail = hub_judge_deliverable(
            hub_base=hub_base,
            agent_name=agent,
            prompt=prompt,
        )
        return ok, f"{agent}@{hub_base}: {detail}"

    ok, detail = cli_judge_deliverable(provider=provider, prompt=prompt, work_dir=work_dir)
    return ok, f"{provider}-cli: {detail}"
