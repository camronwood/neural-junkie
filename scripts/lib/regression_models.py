"""Regression model policy — live gates never run models above 14B."""

from __future__ import annotations

import json
import os
import re
import urllib.request
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS_DIR.parent
MODELS_CONFIG = SCRIPTS_DIR / "config" / "model-benchmark-models.json"

MAX_REGRESSION_PARAMS_B = 14.0
DEFAULT_REGRESSION_AGENT_MODEL = "qwen2.5-coder:14b"
DEFAULT_REGRESSION_UTILITY_MODEL = "qwen3.5:9b"

_PARAM_SUFFIX_RE = re.compile(r":(\d+(?:\.\d+)?)b\b", re.I)
_OVERSIZED_HINT_RE = re.compile(r":(?:35|27|24|22|122)b\b", re.I)

_CATALOG_PARAMS: dict[str, float] | None = None


def _load_catalog_params() -> dict[str, float]:
    global _CATALOG_PARAMS
    if _CATALOG_PARAMS is not None:
        return _CATALOG_PARAMS
    out: dict[str, float] = {}
    if MODELS_CONFIG.is_file():
        try:
            data = json.loads(MODELS_CONFIG.read_text(encoding="utf-8"))
            for item in data.get("models") or []:
                if not isinstance(item, dict):
                    continue
                tag = str(item.get("tag") or "").strip()
                params = item.get("params_b")
                if tag and isinstance(params, (int, float)):
                    out[tag] = float(params)
        except (OSError, json.JSONDecodeError):
            pass
    _CATALOG_PARAMS = out
    return out


def model_params_b(tag: str) -> float | None:
    """Best-effort parameter count from tag suffix or benchmark catalog."""
    tag = (tag or "").strip()
    if not tag:
        return None
    m = _PARAM_SUFFIX_RE.search(tag.lower())
    if m:
        return float(m.group(1))
    catalog = _load_catalog_params()
    if tag in catalog:
        return catalog[tag]
    base = tag.split(":", 1)[0]
    for name, params in catalog.items():
        if name.split(":", 1)[0] == base:
            return params
    return None


def is_regression_allowed_model(tag: str, *, max_b: float = MAX_REGRESSION_PARAMS_B) -> bool:
    tag = (tag or "").strip()
    if not tag:
        return False
    if _OVERSIZED_HINT_RE.search(tag.lower()):
        return False
    params = model_params_b(tag)
    if params is not None:
        return params <= max_b
    # Unknown tags without an explicit size hint — allow (e.g. small HF pulls).
    return True


def filter_regression_models(tags: list[str], *, max_b: float = MAX_REGRESSION_PARAMS_B) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for tag in tags:
        t = (tag or "").strip()
        if not t or t in seen:
            continue
        if is_regression_allowed_model(t, max_b=max_b):
            seen.add(t)
            out.append(t)
    return out


def resolve_regression_agent_model(root: Path = ROOT, env: dict[str, str] | None = None) -> str:
    """Primary Ollama tag for specialists during live regression gates."""
    merged = dict(os.environ)
    if env:
        merged.update(env)
    explicit = (merged.get("NJ_REGRESSION_AGENT_MODEL") or "").strip()
    if explicit and is_regression_allowed_model(explicit):
        return explicit

    from lib.release_prep_env import parse_env_file

    local = parse_env_file(root / "env.local")
    for key in ("OLLAMA_CODE_MODEL", "OLLAMA_MODEL"):
        candidate = (merged.get(key) or local.get(key) or "").strip()
        if candidate and is_regression_allowed_model(candidate):
            return candidate

    return DEFAULT_REGRESSION_AGENT_MODEL


def resolve_regression_utility_model(root: Path = ROOT, env: dict[str, str] | None = None) -> str:
    merged = dict(os.environ)
    if env:
        merged.update(env)
    from lib.release_prep_env import parse_env_file

    local = parse_env_file(root / "env.local")
    candidate = (merged.get("OLLAMA_MODEL") or local.get("OLLAMA_MODEL") or "").strip()
    if candidate and is_regression_allowed_model(candidate):
        return candidate
    return DEFAULT_REGRESSION_UTILITY_MODEL


def regression_ollama_warm_tags(root: Path = ROOT, env: dict[str, str] | None = None) -> list[str]:
    """Models to pull/warm before live regression — never above 14B."""
    merged = dict(os.environ)
    if env:
        merged.update(env)

    tags: list[str] = []
    agent = resolve_regression_agent_model(root, merged)
    utility = resolve_regression_utility_model(root, merged)
    tags.extend([agent, utility])

    judge = (merged.get("NJ_DELIVERABLE_JUDGE_MODEL") or DEFAULT_REGRESSION_AGENT_MODEL).strip()
    if judge:
        tags.append(judge)

    summary = (merged.get("NJ_SESSION_SUMMARY_MODEL") or "qwen2.5:3b").strip()
    if summary:
        tags.append(summary)

    cfg_path = Path.home() / ".neural-junkie" / "config.json"
    if cfg_path.is_file():
        try:
            cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
            impl = cfg.get("implementation") if isinstance(cfg.get("implementation"), dict) else {}
            tool = (impl.get("local_tool_model") or "").strip()
            if tool:
                tags.append(tool)
        except (OSError, json.JSONDecodeError):
            pass

    return filter_regression_models(tags)


def apply_regression_model_env(env: dict[str, str], root: Path = ROOT) -> None:
    """Pin subprocess/hub env to ≤14B regression models (overrides oversized hub config)."""
    env.setdefault("NJ_REGRESSION", "1")
    env.setdefault("NJ_REGRESSION_MAX_PARAMS_B", str(int(MAX_REGRESSION_PARAMS_B)))
    agent = resolve_regression_agent_model(root, env)
    utility = resolve_regression_utility_model(root, env)
    env["NJ_REGRESSION_AGENT_MODEL"] = agent
    env["OLLAMA_CODE_MODEL"] = agent
    env["OLLAMA_MODEL"] = utility
    if not (env.get("NJ_DELIVERABLE_JUDGE_MODEL") or "").strip():
        env["NJ_DELIVERABLE_JUDGE_MODEL"] = DEFAULT_REGRESSION_AGENT_MODEL
    judge = (env.get("NJ_DELIVERABLE_JUDGE_MODEL") or "").strip()
    if judge and not is_regression_allowed_model(judge):
        env["NJ_DELIVERABLE_JUDGE_MODEL"] = DEFAULT_REGRESSION_AGENT_MODEL


def list_hub_ollama_agent_models(hub_url: str) -> list[tuple[str, str]]:
    """Return (agent_name, ai_model) for in-process Ollama agents."""
    from lib import collab_hub as hub

    code, data = hub.hub_request(hub_url.rstrip("/"), "GET", "/api/agents")
    if code != 200 or not isinstance(data, list):
        return []
    out: list[tuple[str, str]] = []
    for item in data:
        if not isinstance(item, dict):
            continue
        provider = (item.get("ai_provider") or "").strip().lower()
        if provider != "ollama":
            continue
        name = (item.get("name") or "").strip()
        model = (item.get("ai_model") or "").strip()
        if name and model:
            out.append((name, model))
    return out


def agents_over_regression_cap(hub_url: str, *, max_b: float = MAX_REGRESSION_PARAMS_B) -> list[str]:
    over: list[str] = []
    for name, model in list_hub_ollama_agent_models(hub_url):
        if not is_regression_allowed_model(model, max_b=max_b):
            params = model_params_b(model)
            label = f"{params:.0f}B" if params is not None else "?B"
            over.append(f"{name}={model} ({label})")
    return over


def switch_all_agents_regression_model(hub_url: str, model: str | None = None) -> tuple[bool, str]:
    from lib.model_benchmark import switch_all_ollama

    tag = (model or resolve_regression_agent_model()).strip()
    if not is_regression_allowed_model(tag):
        return False, f"refusing to switch agents to oversized model {tag}"
    return switch_all_ollama(hub_url, tag)


def unload_ollama_model(tag: str) -> None:
    """Best-effort unload via keep_alive=0."""
    tag = (tag or "").strip()
    if not tag:
        return
    body = json.dumps({"model": tag, "prompt": "", "keep_alive": 0}).encode()
    req = urllib.request.Request(
        "http://127.0.0.1:11434/api/generate",
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=30.0):
            pass
    except OSError:
        pass


def unload_loaded_models_over_cap(*, max_b: float = MAX_REGRESSION_PARAMS_B) -> list[str]:
    """Unload any loaded Ollama model above the regression cap."""
    unloaded: list[str] = []
    try:
        with urllib.request.urlopen("http://127.0.0.1:11434/api/ps", timeout=5.0) as resp:
            data = json.loads(resp.read().decode())
    except OSError:
        return unloaded
    for item in data.get("models") or []:
        if not isinstance(item, dict):
            continue
        name = (item.get("name") or item.get("model") or "").strip()
        if name and not is_regression_allowed_model(name, max_b=max_b):
            unload_ollama_model(name)
            unloaded.append(name)
    return unloaded


def enforce_regression_agent_models(hub_url: str, root: Path = ROOT) -> tuple[bool, str]:
    """Switch all agents to the regression model and verify none exceed 14B."""
    model = resolve_regression_agent_model(root)
    ok, detail = switch_all_agents_regression_model(hub_url, model)
    if not ok:
        return False, detail

    over = agents_over_regression_cap(hub_url)
    if over:
        return False, f"agents still above {int(MAX_REGRESSION_PARAMS_B)}B: {', '.join(over)}"

    unloaded = unload_loaded_models_over_cap()
    if unloaded:
        detail = f"{detail}; unloaded {', '.join(unloaded)}"
    return True, f"switched all agents → {model} ({detail})"
