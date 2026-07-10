"""Tests for collab_hub agent roster helpers."""

from __future__ import annotations

import os
import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.collab_hub import (  # noqa: E402
    _collect_bullet_findings,
    _is_turn_handoff_content,
    collaborate_agent_names,
    parse_agent_mentions,
)


class CollabHubAgentParseTest(unittest.TestCase):
    def test_parse_agent_mentions(self) -> None:
        names = parse_agent_mentions("@ChatModerator @Assistant @BackendEngineer")
        self.assertEqual(names, ["ChatModerator", "Assistant", "BackendEngineer"])

    def test_collaborate_agent_names_skips_goal_placeholders(self) -> None:
        scenario = {
            "required_agents": ["SoftwareArchitect"],
            "collaborate": {
                "goal": "Use - Task 1: @AgentName - description as an example line.",
            },
        }
        names = collaborate_agent_names(scenario, "@BackendEngineer @PlatformEngineer")
        self.assertEqual(names, ["BackendEngineer", "PlatformEngineer", "SoftwareArchitect"])

    def test_is_turn_handoff_content(self) -> None:
        self.assertTrue(_is_turn_handoff_content("Collaboration turn handoff: next participant"))
        self.assertTrue(_is_turn_handoff_content("@SA -- You're up first for: goal"))
        self.assertFalse(_is_turn_handoff_content("Grounding: I loaded README"))

    def test_collect_bullet_findings_skips_task_list(self) -> None:
        messages = [
            {
                "from": {"name": "BackendEngineer"},
                "type": "collaboration_discussion",
                "content": (
                    "- Task 1: @BackendEngineer - Document findings in collabs/<id>/findings.md\n"
                    "- README describes a minimal sample repo.\n"
                    "- main.go prints a greeting from core/sample.\n"
                ),
            }
        ]
        body = _collect_bullet_findings(messages)
        self.assertNotIn("Task 1:", body)
        self.assertIn("README describes", body)
        self.assertIn("main.go prints", body)


if __name__ == "__main__":
    unittest.main()
