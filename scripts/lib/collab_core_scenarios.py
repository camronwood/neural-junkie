"""Scenarios for collab-core layer — fast participation/planning gate (~45–90m).

Heavy website/phoenix/execution sweeps stay in collab-full; use this layer for
fix-loop convergence and infra validation under batch Ollama load.
"""

from __future__ import annotations

COLLAB_CORE_SCENARIOS: tuple[str, ...] = (
    "collab-participation-two-agent-strict",
    "collab-participation-three-agent",
    "collab-human-planning-interject",
    "collab-generation-error-resilience",
    "planning-two-agent",
    "plan-dependency-prose-regression",
    "collab-minimal-completion-regression",
    "document-findings-execution",
)


def collab_core_scenarios() -> list[str]:
    return list(COLLAB_CORE_SCENARIOS)
