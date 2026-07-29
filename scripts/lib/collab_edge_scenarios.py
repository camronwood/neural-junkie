"""Scenarios for collab edge layer — make collab-scenario-regression / LAYER=collab.

Thinned website/findings twins: one website + one findings execute.
Website-sa / make-me-a-website / execute-deliverable / execution-no-stack stay in collab-full.
"""

from __future__ import annotations

COLLAB_EDGE_SCENARIOS: tuple[str, ...] = (
    "collab-minimal-completion-regression",
    "plan-dependency-prose-regression",
    "plan-findings-task-regression",
    "plan-distinct-deliverables-same-agent",
    "document-findings-execution",
    "collab-conversation-quality-regression",
    "collab-no-edit-after-cancel",
    "collaboration-station-website",
)


def collab_edge_scenarios() -> list[str]:
    return list(COLLAB_EDGE_SCENARIOS)
