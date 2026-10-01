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
    IMPLEMENT_REPLY_TYPES,
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

    def test_implement_reply_types_exclude_user_question(self) -> None:
        self.assertNotIn("user_question", IMPLEMENT_REPLY_TYPES)
        self.assertIn("chat", IMPLEMENT_REPLY_TYPES)
        self.assertIn("answer", IMPLEMENT_REPLY_TYPES)

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

    def test_approve_pending_tool_approvals_and_busy(self) -> None:
        from unittest import mock
        from lib import collab_hub as hub

        pending = [
            {"id": "a1", "channel": "user-flow-scenarios"},
            {"id": "a2", "channel": "other"},
        ]
        with mock.patch.object(hub, "list_pending_tool_approvals", return_value=pending), mock.patch.object(
            hub, "approve_tool_approval", return_value=(200, {})
        ) as approve:
            n = hub.approve_pending_tool_approvals("http://hub", channel="user-flow-scenarios")
            self.assertEqual(n, 1)
            approve.assert_called_once_with("http://hub", "a1", scope="once")

        with mock.patch.object(hub, "list_pending_tool_approvals", return_value=[]), mock.patch.object(
            hub, "list_pending_file_changes", return_value=[]
        ):
            self.assertFalse(hub.channel_agent_busy("http://hub", "user-flow-scenarios"))

        with mock.patch.object(hub, "list_pending_tool_approvals", return_value=pending), mock.patch.object(
            hub, "list_pending_file_changes", return_value=[]
        ):
            self.assertTrue(hub.channel_agent_busy("http://hub", "user-flow-scenarios"))

        with mock.patch.object(hub, "list_pending_tool_approvals", return_value=[]), mock.patch.object(
            hub, "list_pending_file_changes", return_value=[{"id": "fc1", "channel": "user-flow-scenarios"}]
        ):
            self.assertTrue(hub.channel_agent_busy("http://hub", "user-flow-scenarios"))

        # Historical registered_change_id must NOT count as busy.
        with mock.patch.object(hub, "list_pending_tool_approvals", return_value=[]), mock.patch.object(
            hub, "list_pending_file_changes", return_value=[]
        ), mock.patch.object(hub, "pending_change_ids_for_channel", return_value=["stale-approved"]):
            self.assertFalse(hub.channel_agent_busy("http://hub", "user-flow-scenarios"))


if __name__ == "__main__":
    unittest.main()
