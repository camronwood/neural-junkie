"""Gemini CLI auth smoke test for deliverable judging and collab preflight."""

from __future__ import annotations

import os
import re
import shutil
import subprocess
import time
from dataclasses import dataclass
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

# Collab @Gemini turns burn quota fast — probe lite first when flash is exhausted.
GEMINI_COLLAB_PROBE_CANDIDATES: list[tuple[str, str]] = [
    ("fast-light", "gemini-2.5-flash-lite"),
    ("fast", "gemini-2.5-flash"),
    ("pro", "gemini-2.5-pro"),
]


@dataclass(frozen=True)
class GeminiTestSelection:
    ok: bool
    api_key_label: str
    model: str
    detail: str


def headless_home() -> Path:
    override = (os.environ.get("NEURAL_JUNKIE_GEMINI_CLI_HOME") or "").strip()
    if override:
        return Path(override).expanduser()
    return DEFAULT_HEADLESS_HOME


def _parse_key_lines(raw: str) -> list[str]:
    keys: list[str] = []
    seen: set[str] = set()
    for line in raw.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line not in seen:
            keys.append(line)
            seen.add(line)
    return keys


def list_gemini_api_keys(root: Path | None = None) -> list[tuple[str, str]]:
    """Return (label, api_key) pairs to probe — env first, then .gemini-api-key lines."""
    root = root or ROOT
    out: list[tuple[str, str]] = []
    seen: set[str] = set()

    env_key = (os.environ.get("GEMINI_API_KEY") or "").strip()
    if env_key and env_key not in seen:
        out.append(("env", env_key))
        seen.add(env_key)

    path = root / ".gemini-api-key"
    if path.is_file():
        for idx, key in enumerate(_parse_key_lines(path.read_text(encoding="utf-8")), start=1):
            if key not in seen:
                out.append((f"key-{idx}", key))
                seen.add(key)
    return out


def resolve_gemini_api_key(root: Path | None = None) -> str:
    keys = list_gemini_api_keys(root)
    if keys:
        return keys[0][1]
    return ""


def _key_hint(label: str, api_key: str) -> str:
    tail = api_key[-4:] if len(api_key) >= 4 else "****"
    return f"{label} (...{tail})"


def gemini_cli_env(*, model: str | None = None, api_key: str | None = None) -> dict[str, str]:
    env = os.environ.copy()
    chosen_key = (api_key or resolve_gemini_api_key()).strip()
    if chosen_key:
        env["GEMINI_API_KEY"] = chosen_key
    chosen = (model or resolve_judge_gemini_model()).strip()
    if chosen:
        env["GEMINI_MODEL"] = chosen
        env["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = chosen
    if not chosen_key:
        return env
    home = headless_home()
    settings = home / ".gemini" / "settings.json"
    if settings.is_file():
        env["GEMINI_CLI_HOME"] = str(home)
    return env


def resolve_judge_gemini_model() -> str:
    """Model for deliverable judge CLI smokes and direct CLI judging."""
    judge = (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
    if judge:
        return judge
    return (os.environ.get("GEMINI_MODEL") or "gemini-2.5-flash").strip()


def _quota_error(detail: str) -> bool:
    lower = detail.lower()
    return "quota" in lower or "429" in lower or "resource_exhausted" in lower


def apply_gemini_test_selection(*, api_key: str, model: str) -> None:
    os.environ["GEMINI_API_KEY"] = api_key
    os.environ["GEMINI_MODEL"] = model
    os.environ["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = model


def check_gemini_judge(
    *,
    timeout_s: float = 60.0,
    model: str | None = None,
    api_key: str | None = None,
    retry_quota: bool = True,
) -> tuple[bool, str]:
    binary = shutil.which("gemini")
    if not binary:
        return False, "gemini CLI not on PATH (npm install -g @google/gemini-cli)"

    chosen_key = (api_key or resolve_gemini_api_key()).strip()
    if not chosen_key:
        return False, (
            "GEMINI_API_KEY is not set. Google ended Code Assist OAuth for individuals on "
            "2026-06-18. Add one key per line to .gemini-api-key (gitignored) or GEMINI_API_KEY "
            "in env.local from https://aistudio.google.com/app/apikey and restart the hub."
        )

    chosen = (model or resolve_judge_gemini_model()).strip()
    cmd = [binary, "--output-format", "text", "-p", PROMPT]
    cli_env = gemini_cli_env(model=chosen or None, api_key=chosen_key)
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


def ensure_gemini_for_testing(
    *,
    root: Path | None = None,
    timeout_s: float = 45.0,
    explicit_model: str = "",
    retry_quota: bool = False,
    collab: bool = False,
    skip_api_keys: set[str] | None = None,
) -> GeminiTestSelection:
    """Probe each API key against model candidates; apply the first working pair."""
    root = root or ROOT
    skip = skip_api_keys or set()
    keys = [(label, key) for label, key in list_gemini_api_keys(root) if key not in skip]
    if not keys:
        return GeminiTestSelection(
            ok=False,
            api_key_label="",
            model="",
            detail="no Gemini API keys configured (.gemini-api-key or GEMINI_API_KEY)",
        )

    explicit = explicit_model.strip()
    models: list[tuple[str, str]] = []
    if explicit:
        models = [("explicit", explicit)]
    else:
        pinned = (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
        if pinned:
            models = [("pinned", pinned)]
        else:
            models = list(GEMINI_COLLAB_PROBE_CANDIDATES if collab else GEMINI_JUDGE_PROBE_CANDIDATES)

    failures: list[str] = []
    for key_label, api_key in keys:
        for model_label, candidate in models:
            ok, detail = check_gemini_judge(
                timeout_s=timeout_s,
                model=candidate,
                api_key=api_key,
                retry_quota=retry_quota,
            )
            if ok:
                apply_gemini_test_selection(api_key=api_key, model=candidate)
                hint = _key_hint(key_label, api_key)
                return GeminiTestSelection(
                    ok=True,
                    api_key_label=key_label,
                    model=candidate,
                    detail=f"{model_label} ({candidate}) via {hint}: {detail}",
                )
            failures.append(f"{key_label}/{model_label}: {detail[:120]}")

    return GeminiTestSelection(
        ok=False,
        api_key_label="",
        model="",
        detail="no Gemini key+model available — " + "; ".join(failures[:12]),
    )


def select_gemini_judge_model(
    *,
    timeout_s: float = 30.0,
    explicit_model: str = "",
    retry_quota: bool = False,
    root: Path | None = None,
) -> tuple[str | None, bool, str]:
    """Try keys × models and return the first working judge model."""
    sel = ensure_gemini_for_testing(
        root=root,
        timeout_s=timeout_s,
        explicit_model=explicit_model,
        retry_quota=retry_quota,
    )
    if sel.ok:
        return sel.model, True, sel.detail
    return None, False, sel.detail
