"""Tests for collab_hub agent roster helpers."""

from __future__ import annotations

import os
import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.collab_hub import collaborate_agent_names, parse_agent_mentions  # noqa: E402


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


if __name__ == "__main__":
    unittest.main()
