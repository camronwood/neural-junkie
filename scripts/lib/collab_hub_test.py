"""Tests for collab_hub agent roster helpers."""

from __future__ import annotations

import os
import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.collab_hub import (  # noqa: E402
    CHAT_REPLY_TYPES,
    _collect_bullet_findings,
    _is_turn_handoff_content,
    collaborate_agent_names,
    parse_agent_mentions,
)


class CollabHubAgentParseTest(unittest.TestCase):
    def test_chat_reply_types_include_user_question(self) -> None:
        self.assertIn("user_question", CHAT_REPLY_TYPES)
        self.assertIn("chat", CHAT_REPLY_TYPES)
        self.assertIn("answer", CHAT_REPLY_TYPES)

    def test_agent_messages_honors_explicit_types(self) -> None:
        from lib.collab_hub import agent_messages

        msgs = [
            {"type": "user_question", "from": {"name": "BackendEngineer"}, "content": "Which theme?"},
            {"type": "file_change", "from": {"name": "FrontendEngineer"}, "content": "edit SidebarFooter"},
            {"type": "system_info", "from": {"name": "System"}, "content": "noise"},
        ]
        chatish = agent_messages(msgs, types=CHAT_REPLY_TYPES)
        self.assertEqual(len(chatish), 1)
        self.assertEqual(chatish[0]["type"], "user_question")
        files = agent_messages(msgs, types=frozenset({"file_change"}))
        self.assertEqual(len(files), 1)
        self.assertEqual(files[0]["from"]["name"], "FrontendEngineer")
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
