"""Environment bootstrap for release-prep, test-everything, and live regression scripts."""

from __future__ import annotations

import os
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GEMINI_KEY_FILE = ROOT / ".gemini-api-key"
ENV_LOCAL = ROOT / "env.local"
HEADLESS_GEMINI_HOME = ROOT / "scripts" / "gemini-headless-home"

# Ollama fallback judge — independent coder model vs typical qwen3.5:9b implement agents.
DEFAULT_OLLAMA_JUDGE_MODEL = "qwen2.5-coder:14b"
# Free-tier Gemini: 5 RPM → ~12s between calls; flash-lite has more headroom than flash.
DEFAULT_GEMINI_JUDGE_MODEL = "gemini-2.5-flash-lite"
DEFAULT_GEMINI_JUDGE_MIN_INTERVAL_S = "13"

_ENV_LINE = re.compile(r"^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$")


def _strip_quotes(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in ("'", '"'):
        return value[1:-1]
    return value


def parse_env_file(path: Path) -> dict[str, str]:
    out: dict[str, str] = {}
    if not path.is_file():
        return out
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        m = _ENV_LINE.match(line)
        if not m:
            continue
        key, raw = m.group(1), _strip_quotes(m.group(2))
        if raw:
            out[key] = raw
    return out


def load_gemini_api_key(root: Path = ROOT) -> str:
    key = (os.environ.get("GEMINI_API_KEY") or "").strip()
    if key:
        return key
    path = root / ".gemini-api-key"
    if path.is_file():
        return path.read_text(encoding="utf-8").strip()
    return ""


def release_prep_env(root: Path = ROOT) -> dict[str, str]:
    """Build subprocess environment for a successful live release gate."""
    env = os.environ.copy()
    local = parse_env_file(root / "env.local")
    for key, val in local.items():
        if val:
            env[key] = val

    gemini_key = load_gemini_api_key(root)
    if gemini_key:
        env["GEMINI_API_KEY"] = gemini_key

    env.setdefault("NEURAL_JUNKIE_RATE_LIMIT", "0")
    env.setdefault("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765")

    headless = root / "scripts" / "gemini-headless-home"
    if headless.is_dir() and not env.get("NEURAL_JUNKIE_GEMINI_CLI_HOME"):
        env["NEURAL_JUNKIE_GEMINI_CLI_HOME"] = str(headless)

    # Cloud-first judge (hub Gemini); Ollama fallback so quota outages do not block sweeps.
    env.setdefault("NJ_DELIVERABLE_JUDGE_PROVIDER", "gemini")
    env.setdefault("NJ_DELIVERABLE_JUDGE_MODE", "hub")
    env.setdefault("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA", "1")
    env.setdefault("NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S", DEFAULT_GEMINI_JUDGE_MIN_INTERVAL_S)
    env.setdefault("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", DEFAULT_GEMINI_JUDGE_MODEL)
    if not env.get("NJ_DELIVERABLE_JUDGE_MODEL"):
        env["NJ_DELIVERABLE_JUDGE_MODEL"] = DEFAULT_OLLAMA_JUDGE_MODEL

    return env


def apply_release_prep_env(root: Path = ROOT) -> dict[str, str]:
    """Load release-prep env into os.environ and return the merged dict."""
    env = release_prep_env(root)
    os.environ.update(env)
    return env


def summarize_release_prep_env(env: dict[str, str]) -> list[str]:
    lines: list[str] = []
    provider = (env.get("NJ_DELIVERABLE_JUDGE_PROVIDER") or "gemini").strip().lower()
    mode = (env.get("NJ_DELIVERABLE_JUDGE_MODE") or "hub").strip().lower()
    fallback = env.get("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA") == "1"
    ollama_model = (env.get("NJ_DELIVERABLE_JUDGE_MODEL") or DEFAULT_OLLAMA_JUDGE_MODEL).strip()
    lines.append(f"deliverable judge: {provider} ({mode})")
    if fallback:
        lines.append(f"NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA=1 → ollama/{ollama_model} on cloud errors")
    if env.get("NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S"):
        lines.append(f"NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S={env['NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S']}")
    gemini_model = (env.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
    if gemini_model:
        lines.append(f"NJ_DELIVERABLE_JUDGE_GEMINI_MODEL={gemini_model}")
    if env.get("GEMINI_API_KEY"):
        lines.append("GEMINI_API_KEY loaded")
    else:
        lines.append("GEMINI_API_KEY missing — cloud judge will use Ollama fallback only")
    return lines
