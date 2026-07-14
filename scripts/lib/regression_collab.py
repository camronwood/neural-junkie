"""Collab regression tuning — slim roster, cloud Claude, excess-agent pause."""

from __future__ import annotations

import os

from lib import collab_hub as hub
from lib.collab_core_scenarios import collab_core_scenarios
from lib.collab_edge_scenarios import collab_edge_scenarios

# Agents required by collab-core scenarios (keep unpaused during core gates).
COLLAB_CORE_KEEP_AGENTS: tuple[str, ...] = (
    "SoftwareArchitect",
    "BackendEngineer",
    "Claude",
)

# Agents required by make collab-scenario-regression / layer-gate LAYER=collab.
# Slim roster still pauses everyone else to reduce Ollama contention.
COLLAB_EDGE_KEEP_AGENTS: tuple[str, ...] = (
    "SoftwareArchitect",
    "BackendEngineer",
    "Claude",
    "PlatformEngineer",
    "FrontendEngineer",
    "SecurityReviewer",
)

CORE_WAIT_DISCUSSION_DEFAULTS: dict[str, object] = {
    "retry_on_generation_error": True,
    "nudge_silent_agents": True,
}

CORE_ASSERT_MESSAGES_DEFAULTS: dict[str, object] = {
    "deny_generation_errors": True,
    "allow_recovered_generation_errors": True,
}


def _env_truthy(name: str) -> bool:
    return os.environ.get(name, "").strip().lower() in ("1", "true", "yes")


def apply_core_scenario_defaults(scenario: dict) -> dict:
    """Merge participation-friendly defaults into collab-core scenario steps."""
    steps = scenario.get("steps")
    if not isinstance(steps, list):
        return scenario
    out_steps: list[dict] = []
    for step in steps:
        if not isinstance(step, dict):
            out_steps.append(step)
            continue
        merged = dict(step)
        action = (merged.get("action") or "").strip()
        if action == "wait_discussion":
            for key, val in CORE_WAIT_DISCUSSION_DEFAULTS.items():
                merged.setdefault(key, val)
        elif action == "assert_messages":
            for key, val in CORE_ASSERT_MESSAGES_DEFAULTS.items():
                merged.setdefault(key, val)
        out_steps.append(merged)
    scenario = dict(scenario)
    scenario["steps"] = out_steps
    return scenario


def unpause_keep_agents(hub_url: str, keep: tuple[str, ...] | list[str]) -> tuple[bool, str]:
    """Ensure core regression agents are not left paused from a prior slim-roster run."""
    base = hub_url.rstrip("/")
    keep_set = {n.strip() for n in keep if n.strip()}
    code, data = hub.hub_request(base, "GET", "/api/agents")
    if code != 200 or not isinstance(data, list):
        return False, f"list agents HTTP {code}"
    unpaused: list[str] = []
    for item in data:
        if not isinstance(item, dict):
            continue
        name = (item.get("name") or "").strip()
        if not name or name not in keep_set:
            continue
        if not item.get("is_paused") and (item.get("status") or "").strip().lower() != "paused":
            continue
        send_code, _ = hub.send_message(base, "general", f"/unpause-agent {name}")
        if send_code == 200:
            unpaused.append(name)
    if not unpaused:
        return True, "core agents already unpaused"
    return True, f"unpaused {len(unpaused)} agent(s): {', '.join(unpaused)}"


def pause_excess_agents(hub_url: str, keep: tuple[str, ...] | list[str]) -> tuple[bool, str]:
    """Pause in-process agents not in keep list to reduce Ollama contention."""
    base = hub_url.rstrip("/")
    keep_set = {n.strip() for n in keep if n.strip()}
    code, data = hub.hub_request(base, "GET", "/api/agents")
    if code != 200 or not isinstance(data, list):
        return False, f"list agents HTTP {code}"
    paused: list[str] = []
    for item in data:
        if not isinstance(item, dict):
            continue
        name = (item.get("name") or "").strip()
        if not name or name in keep_set:
            continue
        if item.get("is_paused") or (item.get("status") or "").strip().lower() == "paused":
            continue
        send_code, _ = hub.send_message(base, "general", f"/pause-agent {name}")
        if send_code == 200:
            paused.append(name)
    if not paused:
        return True, "no excess agents to pause"
    return True, f"paused {len(paused)} agent(s): {', '.join(paused[:8])}{'…' if len(paused) > 8 else ''}"


def switch_claude_to_cloud(hub_url: str) -> tuple[bool, str]:
    """Route @Claude to cloud API during regression so collab gates do not queue on Ollama."""
    base = hub_url.rstrip("/")
    agent_id = hub.resolve_agent_id(base, "Claude")
    if not agent_id:
        return True, "Claude not configured (skip cloud switch)"
    code, data = hub.hub_request(
        base,
        "POST",
        f"/api/agents/{agent_id}/provider",
        {"provider": "claude", "model": ""},
    )
    if code != 200:
        detail = data if isinstance(data, str) else str(data)
        return False, f"Claude cloud switch HTTP {code}: {detail}"
    return True, "Claude → cloud (claude provider)"


def switch_claude_to_ollama(hub_url: str, model: str) -> tuple[bool, str]:
    """Route @Claude to Ollama during regression so collab gates avoid flaky cloud turns."""
    base = hub_url.rstrip("/")
    agent_id = hub.resolve_agent_id(base, "Claude")
    if not agent_id:
        return True, "Claude not configured (skip ollama switch)"
    tag = (model or "").strip()
    if not tag:
        return False, "regression agent model unset"
    code, data = hub.hub_request(
        base,
        "POST",
        f"/api/agents/{agent_id}/provider",
        {"provider": "ollama", "model": tag},
    )
    if code != 200:
        detail = data if isinstance(data, str) else str(data)
        return False, f"Claude ollama switch HTTP {code}: {detail}"
    return True, f"Claude → ollama ({tag})"


def slim_roster_keep_agents() -> list[str]:
    """Agents to leave online when NJ_REGRESSION_SLIM_ROSTER is enabled."""
    if _env_truthy("NJ_REGRESSION_COLLAB_EDGE"):
        keep = list(COLLAB_EDGE_KEEP_AGENTS)
    else:
        keep = list(COLLAB_CORE_KEEP_AGENTS)
    extra = os.environ.get("NJ_REGRESSION_KEEP_AGENTS", "").strip()
    if extra:
        keep.extend(n.strip() for n in extra.split(",") if n.strip())
    # Preserve order; drop dupes.
    seen: set[str] = set()
    out: list[str] = []
    for name in keep:
        if name in seen:
            continue
        seen.add(name)
        out.append(name)
    return out


def apply_collab_regression_tuning(hub_url: str) -> tuple[bool, str]:
    """Apply slim roster + Claude routing when regression collab env flags are set."""
    parts: list[str] = []
    if _env_truthy("NJ_REGRESSION_CLAUDE_CLOUD"):
        ok, detail = switch_claude_to_cloud(hub_url)
        if not ok:
            return False, detail
        parts.append(detail)
    elif _env_truthy("NJ_REGRESSION_SLIM_ROSTER"):
        from pathlib import Path

        from lib.regression_models import resolve_regression_agent_model

        root = Path(__file__).resolve().parents[2]
        model = resolve_regression_agent_model(root)
        ok, detail = switch_claude_to_ollama(hub_url, model)
        if not ok:
            return False, detail
        parts.append(detail)
    if _env_truthy("NJ_REGRESSION_SLIM_ROSTER"):
        keep = slim_roster_keep_agents()
        ok, detail = unpause_keep_agents(hub_url, keep)
        if not ok:
            return False, detail
        parts.append(detail)
        ok, detail = pause_excess_agents(hub_url, keep)
        if not ok:
            return False, detail
        parts.append(detail)
    if not parts:
        return True, "collab regression tuning skipped (flags off)"
    return True, "; ".join(parts)


def collab_core_keep_agents() -> list[str]:
    return list(COLLAB_CORE_KEEP_AGENTS)


def resolve_preflight_roster() -> list[str]:
    """Agents that must be online before collab sweeps (slim roster when enabled)."""
    if _env_truthy("NJ_REGRESSION_SLIM_ROSTER"):
        return slim_roster_keep_agents()
    return [
        "BackendEngineer",
        "SoftwareArchitect",
        "PlatformEngineer",
        "FrontendEngineer",
        "SecurityReviewer",
    ]


def is_collab_core_scenario(name: str) -> bool:
    return name in collab_core_scenarios()


def is_collab_edge_scenario(name: str) -> bool:
    return name in collab_edge_scenarios()
