"""Shared Claude invocation for SUT human-sim and judge (CLI preferred, hub fallback)."""

from __future__ import annotations

import os
import shutil
import subprocess
import time
from typing import Callable

from lib import collab_hub as hub

DEFAULT_CLAUDE_AGENT = "Claude"
DEFAULT_TIMEOUT_S = 240.0


def _claude_cli(prompt: str, *, timeout_s: float, work_dir: str = "") -> tuple[bool, str]:
    binary = shutil.which("claude")
    if not binary:
        return False, "claude CLI not on PATH"
    try:
        from lib.claude_judge_auth import claude_subprocess_env
    except ImportError:
        from claude_judge_auth import claude_subprocess_env  # type: ignore[no-redef]
    run_env = claude_subprocess_env()
    cmd = [binary, "-p", prompt]
    try:
        proc = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=timeout_s,
            cwd=work_dir or None,
            env=run_env,
            check=False,
        )
    except subprocess.TimeoutExpired:
        return False, f"claude CLI timeout ({timeout_s}s)"
    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()[:400]
        return False, f"claude CLI exit {proc.returncode}: {err or 'no output'}"
    text = (proc.stdout or "").strip()
    if not text:
        return False, "claude CLI empty output"
    return True, text


def _claude_hub(
    *,
    hub_base: str,
    prompt: str,
    dm_user: str,
    agent_name: str,
    timeout_s: float,
    meta_flag: str,
) -> tuple[bool, str]:
    base = hub_base.rstrip("/")
    ok, missing = hub.verify_agents_online(base, [agent_name])
    if not ok:
        return False, f"claude hub agent offline: {', '.join(missing)}"

    channel = hub.ensure_dm_channel(base, dm_user, agent_name)
    if not channel:
        return False, f"could not open DM with {agent_name!r}"

    hub.clear_channel_history(base, channel)
    time.sleep(1.5)

    code, _ = hub.send_message(
        base,
        channel,
        prompt,
        from_name=dm_user,
        metadata={
            "editor_mode": "ask",
            "conversation_mode": "chat",
            meta_flag: True,
        },
    )
    if code != 200:
        return False, f"claude hub send failed ({code})"

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
                return True, text
        time.sleep(2.0)
    return False, f"timeout waiting for {agent_name} via hub ({timeout_s}s)"


def ask_claude(
    *,
    prompt: str,
    hub_base: str = "",
    dm_user: str = "SutClaude",
    agent_name: str = DEFAULT_CLAUDE_AGENT,
    timeout_s: float = DEFAULT_TIMEOUT_S,
    meta_flag: str = "sut_claude",
    work_dir: str = "",
) -> tuple[bool, str]:
    """Ask Claude via CLI (default) or hub DM.

    Mode: NJ_SUT_CLAUDE_MODE=cli|hub|auto (default auto = CLI then hub fallback).
    Regression hubs often pin @Claude to Ollama when ANTHROPIC_API_KEY is not sk-ant-…;
    CLI uses Claude Code OAuth and is the reliable cloud path.
    """
    mode = os.environ.get("NJ_SUT_CLAUDE_MODE", "auto").strip().lower() or "auto"
    errors: list[str] = []

    def try_cli() -> tuple[bool, str] | None:
        ok, text = _claude_cli(prompt, timeout_s=timeout_s, work_dir=work_dir)
        if ok:
            return True, text
        errors.append(f"cli: {text}")
        return None

    def try_hub() -> tuple[bool, str] | None:
        if not hub_base.strip():
            errors.append("hub: no hub_base")
            return None
        ok, text = _claude_hub(
            hub_base=hub_base,
            prompt=prompt,
            dm_user=dm_user,
            agent_name=agent_name,
            timeout_s=timeout_s,
            meta_flag=meta_flag,
        )
        if ok:
            return True, text
        errors.append(f"hub: {text}")
        return None

    order: list[Callable[[], tuple[bool, str] | None]]
    if mode == "hub":
        order = [try_hub, try_cli]
    elif mode == "cli":
        order = [try_cli]
    else:
        order = [try_cli, try_hub]

    for fn in order:
        result = fn()
        if result is not None:
            return result
    return False, "; ".join(errors) if errors else "claude unavailable"
