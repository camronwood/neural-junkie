"""Gemini CLI auth smoke test for deliverable judging."""

from __future__ import annotations

import os
import re
import shutil
import subprocess
import time
from pathlib import Path

try:
    from lib.gemini_rate_limit import throttle_gemini_api_call
except ImportError:
    from gemini_rate_limit import throttle_gemini_api_call  # type: ignore[no-redef]

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_HEADLESS_HOME = ROOT / "scripts" / "gemini-headless-home"
PROMPT = "Reply with exactly two lines:\nLine 1: PASS\nLine 2: ok"


def headless_home() -> Path:
    override = (os.environ.get("NEURAL_JUNKIE_GEMINI_CLI_HOME") or "").strip()
    if override:
        return Path(override).expanduser()
    return DEFAULT_HEADLESS_HOME


def resolve_gemini_api_key() -> str:
    key = (os.environ.get("GEMINI_API_KEY") or "").strip()
    if key:
        return key
    path = ROOT / ".gemini-api-key"
    if path.is_file():
        return path.read_text(encoding="utf-8").strip()
    return ""


def resolve_judge_gemini_model() -> str:
    """Model for deliverable judge CLI smokes and direct CLI judging."""
    judge = (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
    if judge:
        return judge
    return (os.environ.get("GEMINI_MODEL") or "gemini-2.5-flash").strip()


def gemini_cli_env() -> dict[str, str]:
    env = os.environ.copy()
    api_key = resolve_gemini_api_key()
    if api_key:
        env["GEMINI_API_KEY"] = api_key
    model = resolve_judge_gemini_model()
    if model:
        env["GEMINI_MODEL"] = model
    if not api_key:
        return env
    home = headless_home()
    settings = home / ".gemini" / "settings.json"
    if settings.is_file():
        env["GEMINI_CLI_HOME"] = str(home)
    return env


def check_gemini_judge(*, timeout_s: float = 60.0) -> tuple[bool, str]:
    binary = shutil.which("gemini")
    if not binary:
        return False, "gemini CLI not on PATH (npm install -g @google/gemini-cli)"

    api_key = resolve_gemini_api_key()
    if not api_key:
        return False, (
            "GEMINI_API_KEY is not set. Google ended Code Assist OAuth for individuals on "
            "2026-06-18. Add a key to .gemini-api-key (gitignored) or GEMINI_API_KEY in env.local "
            "from https://aistudio.google.com/app/apikey and restart the hub."
        )

    cmd = [binary, "--output-format", "text", "-p", PROMPT]
    throttle_gemini_api_call()
    try:
        proc = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=timeout_s,
            check=False,
            env=gemini_cli_env(),
        )
    except subprocess.TimeoutExpired:
        return False, f"gemini CLI smoke timed out ({timeout_s}s)"
    except OSError as exc:
        return False, f"gemini CLI failed to start: {exc}"

    combined = (proc.stdout or "") + (proc.stderr or "")
    lower = combined.lower()
    if "ineligibletier" in lower or "no longer supported for gemini code assist" in lower:
        return False, (
            "Gemini OAuth tier is deprecated. Ensure GEMINI_API_KEY is set and "
            f"GEMINI_CLI_HOME points at {DEFAULT_HEADLESS_HOME} (or run from repo root)."
        )
    if proc.returncode != 0:
        detail = (proc.stderr or proc.stdout or "").strip().replace("\n", " ")
        lower = detail.lower()
        if "quota" in lower or "429" in lower or "resource_exhausted" in lower:
            m = re.search(r"retry in ([\d.]+)s", detail, re.I)
            if m:
                try:
                    delay = float(m.group(1))
                    if 0 < delay < 120:
                        time.sleep(delay + 0.5)
                        throttle_gemini_api_call()
                        proc = subprocess.run(
                            cmd,
                            capture_output=True,
                            text=True,
                            timeout=timeout_s,
                            check=False,
                            env=gemini_cli_env(),
                        )
                        if proc.returncode == 0:
                            first = (proc.stdout or "").strip().splitlines()
                            if first and first[0].strip().upper().startswith("PASS"):
                                return True, "gemini-api-key auth OK (PASS smoke, after quota retry)"
                except ValueError:
                    pass
        return False, f"gemini exit {proc.returncode}: {detail[:400]}"

    first = (proc.stdout or "").strip().splitlines()
    if not first or not first[0].strip().upper().startswith("PASS"):
        snippet = (proc.stdout or proc.stderr or "").strip()[:200]
        return False, f"unexpected judge smoke response: {snippet!r}"
    return True, "gemini-api-key auth OK (PASS smoke)"
