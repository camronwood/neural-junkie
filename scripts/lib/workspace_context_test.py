"""Tests for scenario send metadata enrichment."""

from __future__ import annotations

from lib.workspace_context import enrich_send_metadata, ide_route_for_target_agent


def test_ide_route_for_target_agent():
    assert ide_route_for_target_agent("BackendEngineer") == "backend"
    assert ide_route_for_target_agent("@FrontendEngineer") == "frontend"
    assert ide_route_for_target_agent("Unknown") == ""


def test_enrich_send_metadata_injects_route_from_target_agent():
    meta = enrich_send_metadata(
        {"editor_mode": "plan", "conversation_mode": "code"},
        {"target_agent": "BackendEngineer", "workspace": {"fixture": "minimal-repo"}},
        content="plan a fix",
    )
    assert meta is not None
    assert meta["ide_route_agent_type"] == "backend"


def test_enrich_send_metadata_keeps_explicit_route():
    meta = enrich_send_metadata(
        {"ide_route_agent_type": "frontend", "editor_mode": "agent"},
        {"target_agent": "BackendEngineer", "workspace": {"fixture": "minimal-repo"}},
        content="fix it",
    )
    assert meta is not None
    assert meta["ide_route_agent_type"] == "frontend"
