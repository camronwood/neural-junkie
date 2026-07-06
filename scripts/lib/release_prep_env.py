"""Environment bootstrap for release-prep, test-everything, and live regression scripts."""

from __future__ import annotations

import os
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GEMINI_KEY_FILE = ROOT / ".gemini-api-key"
CURSOR_KEY_FILE = ROOT / ".cursor-api-key"
ENV_LOCAL = ROOT / "env.local"
HEADLESS_GEMINI_HOME = ROOT / "scripts" / "gemini-headless-home"

# Ollama fallback judge — independent coder model vs typical qwen3.5:9b implement agents.
DEFAULT_OLLAMA_JUDGE_MODEL = "qwen2.5-coder:14b"
# Free-tier Gemini: 5 RPM → ~12s between calls; release-prep probes fast → pro → fast-light.
DEFAULT_GEMINI_JUDGE_MIN_INTERVAL_S = "13"
# Cap impl-session wall clock during live regression (aligns with 600–900s scenario wait_reply).
DEFAULT_AGENT_TIMEOUT_MINUTES = "10"

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


def load_gemini_api_keys(root: Path = ROOT) -> list[str]:
    """All configured Gemini API keys (env first, then one per line in .gemini-api-key)."""
    from lib.gemini_judge_auth import list_gemini_api_keys

    return [key for _label, key in list_gemini_api_keys(root)]


def load_gemini_api_key(root: Path = ROOT) -> str:
    keys = load_gemini_api_keys(root)
    return keys[0] if keys else ""


def load_cursor_api_key(root: Path = ROOT) -> str:
    key = (os.environ.get("CURSOR_API_KEY") or "").strip()
    if key:
        return key
    path = root / ".cursor-api-key"
    if path.is_file():
        return path.read_text(encoding="utf-8").strip()
    return ""


def explicit_gemini_judge_model(root: Path = ROOT) -> str:
    """User-configured judge model (env.local or shell); empty means probe at release-prep start."""
    local = parse_env_file(root / "env.local")
    return (
        local.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", "").strip()
        or (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
    )


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

    cursor_key = load_cursor_api_key(root)
    if cursor_key:
        env["CURSOR_API_KEY"] = cursor_key

    env.setdefault("NEURAL_JUNKIE_RATE_LIMIT", "0")
    env.setdefault("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765")
    env.setdefault("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
    # Regression collab/implement scenarios default to the minimal fixture, not the NJ repo root.
    env.setdefault(
        "NEURAL_JUNKIE_SCENARIO_REPO",
        str((root / "scenarios" / "fixtures" / "minimal-repo").resolve()),
    )

    headless = root / "scripts" / "gemini-headless-home"
    if headless.is_dir() and not env.get("NEURAL_JUNKIE_GEMINI_CLI_HOME"):
        env["NEURAL_JUNKIE_GEMINI_CLI_HOME"] = str(headless)

    # Local Ollama judge for live regression (Gemini CLI quota is too flaky for sweeps).
    env.setdefault("NJ_DELIVERABLE_JUDGE_PROVIDER", "ollama")
    env.setdefault("NJ_DELIVERABLE_JUDGE_MODE", "ollama")
    env.setdefault("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA", "1")
    env.setdefault("NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S", DEFAULT_GEMINI_JUDGE_MIN_INTERVAL_S)
    shell_gemini = (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
    explicit_gemini = local.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", "").strip() or shell_gemini
    if explicit_gemini:
        env["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = explicit_gemini
    else:
        env.pop("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", None)
    if not env.get("NJ_DELIVERABLE_JUDGE_MODEL"):
        env["NJ_DELIVERABLE_JUDGE_MODEL"] = DEFAULT_OLLAMA_JUDGE_MODEL

    env.setdefault("NJ_AGENT_TIMEOUT_MINUTES", DEFAULT_AGENT_TIMEOUT_MINUTES)

    try:
        from lib.hub_config import apply_automation_to_env

        apply_automation_to_env(env)
    except ImportError:
        pass

    return env


def apply_release_prep_env(root: Path = ROOT) -> dict[str, str]:
    """Load release-prep env into os.environ and return the merged dict."""
    env = release_prep_env(root)
    os.environ.update(env)
    return env


def provision_hub_automation_key(root: Path = ROOT) -> bool:
    """Ensure ~/.neural-junkie/automation.key exists when bootstrap token is configured."""
    from lib.hub_auth import try_provision_automation_api_key

    merged = apply_release_prep_env(root)
    hub = (merged.get("NEURAL_JUNKIE_HUB_URL") or "http://127.0.0.1:18765").strip()
    return try_provision_automation_api_key(hub.rstrip("/"))


def summarize_release_prep_env(env: dict[str, str]) -> list[str]:
    from lib.gemini_judge_auth import GEMINI_JUDGE_PROBE_CANDIDATES

    lines: list[str] = []
    provider = (env.get("NJ_DELIVERABLE_JUDGE_PROVIDER") or "claude").strip().lower()
    mode = (env.get("NJ_DELIVERABLE_JUDGE_MODE") or "hub").strip().lower()
    fallback = env.get("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA") == "1"
    ollama_model = (env.get("NJ_DELIVERABLE_JUDGE_MODEL") or DEFAULT_OLLAMA_JUDGE_MODEL).strip()
    lines.append(f"deliverable judge: {provider} ({mode})")
    if fallback:
        lines.append(f"NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA=1 → ollama/{ollama_model} on cloud errors")
    if env.get("NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S"):
        lines.append(f"NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S={env['NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S']}")
    if provider == "gemini":
        gemini_model = (env.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
        if gemini_model:
            lines.append(f"NJ_DELIVERABLE_JUDGE_GEMINI_MODEL={gemini_model}")
        else:
            probe_labels = " → ".join(label for label, _ in GEMINI_JUDGE_PROBE_CANDIDATES)
            lines.append(f"NJ_DELIVERABLE_JUDGE_GEMINI_MODEL=probe ({probe_labels})")
        if env.get("GEMINI_API_KEY"):
            lines.append("GEMINI_API_KEY loaded")
        else:
            lines.append("GEMINI_API_KEY missing — cloud judge will use Ollama fallback only")
    elif provider == "claude":
        agent = (env.get("NJ_DELIVERABLE_JUDGE_AGENT") or "Claude").strip()
        lines.append(f"NJ_DELIVERABLE_JUDGE_AGENT={agent}")
        lines.append("Claude Code CLI on PATH + hub @Claude for cloud judging")
    if env.get("NEURAL_JUNKIE_AUTH_REQUIRED") == "1":
        lines.append("NEURAL_JUNKIE_AUTH_REQUIRED=1 (scripts use API key or hub_auth session)")
    if env.get("NJ_AGENT_TIMEOUT_MINUTES"):
        lines.append(f"NJ_AGENT_TIMEOUT_MINUTES={env['NJ_AGENT_TIMEOUT_MINUTES']}")
    return lines
