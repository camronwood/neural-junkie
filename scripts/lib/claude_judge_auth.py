"""Claude Code CLI auth smoke test for deliverable judging and collab preflight."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PROMPT = "Reply with exactly two lines:\nLine 1: PASS\nLine 2: ok"


@dataclass(frozen=True)
class ClaudeTestSelection:
    ok: bool
    detail: str


def claude_proxy_env_keys() -> tuple[str, ...]:
    return (
        "ANTHROPIC_BASE_URL",
        "ANTHROPIC_AUTH_TOKEN",
        "ANTHROPIC_API_KEY",
        "ANTHROPIC_MODEL",
        "ANTHROPIC_SMALL_FAST_MODEL",
    )


def claude_subprocess_env() -> dict[str, str]:
    """Drop env auth overrides so Claude Code uses OAuth / default Anthropic routing."""
    if os.environ.get("NEURAL_JUNKIE_CLAUDE_CUSTOM_ROUTING", "").strip().lower() in ("1", "true", "yes"):
        return os.environ.copy()
    env = os.environ.copy()
    for key in claude_proxy_env_keys():
        env.pop(key, None)
    return env


def strip_claude_proxy_env(env: dict[str, str]) -> dict[str, str]:
    """Return env copy without LiteLLM/proxy overrides (for hub boot)."""
    if os.environ.get("NEURAL_JUNKIE_CLAUDE_CUSTOM_ROUTING", "").strip().lower() in ("1", "true", "yes"):
        return env.copy()
    out = env.copy()
    for key in claude_proxy_env_keys():
        out.pop(key, None)
    return out


def _run_claude(cmd: list[str], **kwargs) -> subprocess.CompletedProcess[str]:
    kwargs["env"] = claude_subprocess_env()
    return subprocess.run(cmd, **kwargs)


def claude_binary() -> str:
    return (shutil.which("claude") or "").strip()


def check_claude_auth(*, timeout_s: float = 15.0) -> tuple[bool, str]:
    binary = claude_binary()
    if not binary:
        return False, "claude CLI not on PATH (npm install -g @anthropic-ai/claude-code)"
    try:
        proc = _run_claude(
            [binary, "auth", "status"],
            capture_output=True,
            text=True,
            timeout=timeout_s,
            check=False,
        )
    except subprocess.TimeoutExpired:
        return False, f"claude auth status timed out ({timeout_s}s)"
    except OSError as exc:
        return False, f"claude auth status failed: {exc}"

    out = (proc.stdout or proc.stderr or "").strip()
    if proc.returncode != 0:
        return False, f"claude not authenticated: {out[:300] or f'exit {proc.returncode}'}"

    try:
        data = json.loads(out)
    except json.JSONDecodeError:
        lower = out.lower()
        if "loggedin" in lower and "true" in lower:
            return True, "claude OAuth OK"
        return False, f"unparseable claude auth status: {out[:200]}"

    if data.get("loggedIn") is True:
        method = (data.get("authMethod") or "oauth").strip()
        return True, f"claude auth OK ({method})"
    return False, "claude not logged in (run: claude login)"


def check_claude_judge(*, timeout_s: float = 45.0, smoke: bool = True) -> tuple[bool, str]:
    binary = claude_binary()
    if not binary:
        return False, "claude CLI not on PATH (npm install -g @anthropic-ai/claude-code)"

    auth_ok, auth_detail = check_claude_auth(timeout_s=min(timeout_s, 15.0))
    if not auth_ok:
        return False, auth_detail
    if not smoke:
        return True, auth_detail

    try:
        proc = _run_claude(
            [binary, "-p", PROMPT],
            capture_output=True,
            text=True,
            timeout=timeout_s,
            check=False,
        )
    except subprocess.TimeoutExpired:
        return False, f"claude CLI smoke timed out ({timeout_s}s)"
    except OSError as exc:
        return False, f"claude CLI failed to start: {exc}"

    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()
        return False, f"claude exit {proc.returncode}: {err[:400]}"

    body = (proc.stdout or "").strip()
    if "PASS" in body.upper():
        return True, f"claude-cli auth OK ({auth_detail}, PASS smoke)"
    return False, f"claude smoke missing PASS: {body[:200]}"


def ensure_claude_for_testing(*, timeout_s: float = 45.0, smoke: bool | None = None) -> ClaudeTestSelection:
    if smoke is None:
        smoke = os.environ.get("NJ_CLAUDE_PROBE_SMOKE", "").strip().lower() in ("1", "true", "yes")
    ok, detail = check_claude_judge(timeout_s=timeout_s, smoke=smoke)
    return ClaudeTestSelection(ok=ok, detail=detail)
