"""Scenarios for collab edge layer — make collab-scenario-regression / LAYER=collab.

Canonical list for --suite edge (one process, soft between-scenario reset).
"""

from __future__ import annotations

COLLAB_EDGE_SCENARIOS: tuple[str, ...] = (
    "collab-minimal-completion-regression",
    "plan-dependency-prose-regression",
    "plan-findings-task-regression",
    "plan-distinct-deliverables-same-agent",
    "execute-deliverable",
    "document-findings-execution",
    "execution-no-stack-commands",
    "collab-conversation-quality-regression",
    "collab-no-edit-after-cancel",
    "collaboration-station-website",
    "collaboration-station-website-sa",
    "make-me-a-website",
)


def collab_edge_scenarios() -> list[str]:
    return list(COLLAB_EDGE_SCENARIOS)
