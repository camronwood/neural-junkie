"""Load hub automation settings from GET /api/settings when env vars are unset."""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from typing import Any


def hub_base_url() -> str:
    return os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")


def fetch_hub_settings() -> dict[str, Any]:
    url = f"{hub_base_url()}/api/settings"
    req = urllib.request.Request(url, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except (urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        return {}


def automation_config() -> dict[str, Any]:
    settings = fetch_hub_settings()
    auto = settings.get("automation") or {}
    if not isinstance(auto, dict):
        return {}
    return auto


def env_or_automation(key: str, field: str, default: str = "") -> str:
    env_val = (os.environ.get(key) or "").strip()
    if env_val:
        return env_val
    raw = automation_config().get(field)
    if raw is None:
        return default
    return str(raw).strip() or default


def env_or_automation_bool(key: str, field: str, default: bool = False) -> bool:
    env_val = (os.environ.get(key) or "").strip().lower()
    if env_val in ("1", "true", "yes"):
        return True
    if env_val in ("0", "false", "no"):
        return False
    auto = automation_config()
    if field in auto:
        return bool(auto[field])
    return default


def env_or_automation_int(key: str, field: str, default: int) -> int:
    env_val = (os.environ.get(key) or "").strip()
    if env_val:
        try:
            return int(env_val)
        except ValueError:
            return default
    auto = automation_config()
    raw = auto.get(field)
    if raw is None:
        return default
    try:
        return int(raw)
    except (TypeError, ValueError):
        return default


def apply_automation_to_env(env: dict[str, str]) -> None:
    """Fill unset automation env keys from hub config (env wins when already set)."""
    pairs: list[tuple[str, str, str]] = [
        ("NJ_DELIVERABLE_JUDGE_PROVIDER", "deliverable_judge_provider", "gemini"),
        ("NJ_DELIVERABLE_JUDGE_MODE", "deliverable_judge_mode", "hub"),
        ("NJ_DELIVERABLE_JUDGE_MODEL", "deliverable_judge_model", "qwen2.5-coder:14b"),
        ("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", "deliverable_judge_gemini_model", ""),
        ("NJ_DELIVERABLE_JUDGE_AGENT", "deliverable_judge_agent", ""),
        ("NEURAL_JUNKIE_SCENARIO_REPO", "scenario_repo", ""),
        ("NEURAL_JUNKIE_HUMAN_NAME", "human_name", ""),
    ]
    for key, field, default in pairs:
        if (env.get(key) or "").strip():
            continue
        val = env_or_automation(key, field, default)
        if val:
            env[key] = val
    int_pairs: list[tuple[str, str, int]] = [
        ("NJ_DELIVERABLE_JUDGE_TIMEOUT", "deliverable_judge_timeout", 180),
        ("NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S", "deliverable_judge_min_interval_s", 13),
    ]
    for key, field, default in int_pairs:
        if (env.get(key) or "").strip():
            continue
        env[key] = str(env_or_automation_int(key, field, default))
    bool_pairs: list[tuple[str, str, bool]] = [
        ("NJ_DELIVERABLE_JUDGE_SKIP", "deliverable_judge_skip", False),
        ("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA", "deliverable_judge_fallback_ollama", True),
        ("NJ_SCENARIO_ALLOW_FILE_FALLBACK", "scenario_allow_file_fallback", False),
        ("NEURAL_JUNKIE_AGENT_POLL", "agent_poll", False),
    ]
    for key, field, default in bool_pairs:
        if (env.get(key) or "").strip():
            continue
        if env_or_automation_bool(key, field, default):
            env[key] = "1"
        elif not default:
            env[key] = "0"
