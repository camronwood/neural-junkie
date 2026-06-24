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

# Release-prep probes these in order when NJ_DELIVERABLE_JUDGE_GEMINI_MODEL is unset.
GEMINI_JUDGE_PROBE_CANDIDATES: list[tuple[str, str]] = [
    ("fast", "gemini-2.5-flash"),
    ("pro", "gemini-2.5-pro"),
    ("fast-light", "gemini-2.5-flash-lite"),
]


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


def gemini_cli_env(*, model: str | None = None) -> dict[str, str]:
    env = os.environ.copy()
    api_key = resolve_gemini_api_key()
    if api_key:
        env["GEMINI_API_KEY"] = api_key
    chosen = (model or resolve_judge_gemini_model()).strip()
    if chosen:
        env["GEMINI_MODEL"] = chosen
        env["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = chosen
    if not api_key:
        return env
    home = headless_home()
    settings = home / ".gemini" / "settings.json"
    if settings.is_file():
        env["GEMINI_CLI_HOME"] = str(home)
    return env


def _quota_error(detail: str) -> bool:
    lower = detail.lower()
    return "quota" in lower or "429" in lower or "resource_exhausted" in lower


def check_gemini_judge(
    *,
    timeout_s: float = 60.0,
    model: str | None = None,
    retry_quota: bool = True,
) -> tuple[bool, str]:
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

    chosen = (model or resolve_judge_gemini_model()).strip()
    cmd = [binary, "--output-format", "text", "-p", PROMPT]
    cli_env = gemini_cli_env(model=chosen or None)
    throttle_gemini_api_call()
    try:
        proc = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=timeout_s,
            check=False,
            env=cli_env,
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
        if retry_quota and _quota_error(detail):
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
                            env=cli_env,
                        )
                        if proc.returncode == 0:
                            first = (proc.stdout or "").strip().splitlines()
                            if first and first[0].strip().upper().startswith("PASS"):
                                label = chosen or "default"
                                return True, f"gemini-api-key auth OK ({label}, after quota retry)"
                except ValueError:
                    pass
        suffix = f" [{chosen}]" if chosen else ""
        return False, f"gemini exit {proc.returncode}{suffix}: {detail[:400]}"

    first = (proc.stdout or "").strip().splitlines()
    if not first or not first[0].strip().upper().startswith("PASS"):
        snippet = (proc.stdout or proc.stderr or "").strip()[:200]
        suffix = f" [{chosen}]" if chosen else ""
        return False, f"unexpected judge smoke response{suffix}: {snippet!r}"
    label = chosen or "default"
    return True, f"gemini-api-key auth OK ({label}, PASS smoke)"


def select_gemini_judge_model(
    *,
    timeout_s: float = 30.0,
    explicit_model: str = "",
    retry_quota: bool = False,
) -> tuple[str | None, bool, str]:
    """Try fast → pro → fast-light (or only explicit_model) and return the first that works."""
    api_key = resolve_gemini_api_key()
    if not api_key:
        return None, False, "GEMINI_API_KEY missing — cannot probe Gemini judge models"

    explicit = explicit_model.strip()
    if explicit:
        ok, detail = check_gemini_judge(timeout_s=timeout_s, model=explicit, retry_quota=True)
        return explicit if ok else None, ok, detail

    already = (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
    if already:
        ok, detail = check_gemini_judge(timeout_s=timeout_s, model=already, retry_quota=True)
        return already if ok else None, ok, detail

    failures: list[str] = []
    for label, candidate in GEMINI_JUDGE_PROBE_CANDIDATES:
        ok, detail = check_gemini_judge(
            timeout_s=timeout_s,
            model=candidate,
            retry_quota=retry_quota,
        )
        if ok:
            return candidate, True, f"{label} ({candidate}): {detail}"
        failures.append(f"{label}: {detail[:100]}")
    return None, False, "no Gemini judge model available — " + "; ".join(failures)
