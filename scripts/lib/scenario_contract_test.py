"""Unit tests for scenario deliverable contract validation."""

from __future__ import annotations

import unittest

from scenario_contract import (
    validate_deliverable_contract,
    validate_implement_wait_gates,
    validate_scenario_shape,
)


class DeliverableContractTest(unittest.TestCase):
    def test_implement_agent_requires_expect_deliverables(self) -> None:
        scenario = {
            "steps": [
                {
                    "action": "send",
                    "content": "implement foo",
                    "metadata": {"editor_mode": "agent", "implementation_session": True},
                }
            ]
        }
        errors = validate_deliverable_contract("implement/example.json", scenario)
        self.assertTrue(errors)
        self.assertIn("expect_deliverables", errors[0])

    def test_implement_ask_mode_opt_out(self) -> None:
        scenario = {
            "expect_no_deliverables": True,
            "steps": [
                {
                    "action": "send",
                    "content": "implement foo",
                    "metadata": {"editor_mode": "ask", "implementation_session": True},
                },
                {"action": "assert_no_file_change"},
            ],
        }
        errors = validate_deliverable_contract("implement/ask-mode.json", scenario)
        self.assertEqual(errors, [])

    def test_implement_valid_contract(self) -> None:
        scenario = {
            "steps": [
                {
                    "action": "send",
                    "content": "implement HelloWorld",
                    "metadata": {"editor_mode": "agent", "implementation_session": True},
                }
            ],
            "expect_deliverables": [
                {
                    "path": "core/sample/main.go",
                    "for_question": {"contains_all": ["HelloWorld"]},
                    "llm_judge": True,
                }
            ],
        }
        errors = validate_deliverable_contract("implement/go-handler.json", scenario)
        self.assertEqual(errors, [])

    def test_collab_with_assert_files_requires_contract(self) -> None:
        scenario = {
            "steps": [{"action": "assert_files", "path": "collabs/<collab-id>/findings.md"}],
        }
        errors = validate_deliverable_contract("collab/execute-deliverable.json", scenario)
        self.assertTrue(errors)

    def test_collab_valid_contract(self) -> None:
        scenario = {
            "steps": [{"action": "assert_files", "path": "collabs/<collab-id>/findings.md"}],
            "expect_deliverables": [
                {
                    "path": "collabs/<collab-id>/findings.md",
                    "min_bytes": 40,
                    "for_question": {"any_match": ["main\\.go"]},
                    "llm_judge": True,
                }
            ],
        }
        errors = validate_deliverable_contract("collab/execute-deliverable.json", scenario)
        self.assertEqual(errors, [])

    def test_must_exist_requires_quality_bar(self) -> None:
        scenario = {
            "steps": [{"action": "assert_files", "path": "collabs/<collab-id>/x.md"}],
            "expect_deliverables": [{"path": "collabs/<collab-id>/x.md"}],
        }
        errors = validate_deliverable_contract("collab/bad.json", scenario)
        self.assertTrue(any("for_question" in e or "llm_judge" in e for e in errors))

    def test_long_horizon_chat_requires_four_turns_and_metrics(self) -> None:
        scenario = {
            "name": "short",
            "evaluation": {"long_horizon": True, "min_user_turns": 4},
            "steps": [{"action": "send", "content": "one"}],
        }
        errors = validate_scenario_shape("chat/short.json", scenario)
        self.assertTrue(any("at least 4" in error for error in errors))
        self.assertTrue(any("assert_transcript_metrics" in error for error in errors))

    def test_dialogue_tag_requires_three_turns_and_continuity_assert(self) -> None:
        scenario = {
            "name": "thin",
            "tags": ["dm", "dialogue"],
            "steps": [
                {"action": "send", "content": "one"},
                {"action": "send", "content": "two"},
            ],
        }
        errors = validate_scenario_shape("chat/thin.json", scenario)
        self.assertTrue(any("at least 3 send" in error for error in errors))
        self.assertTrue(any("assert_transcript_metrics" in error or "any_match" in error for error in errors))

    def test_dialogue_tag_accepts_metrics_continuity(self) -> None:
        scenario = {
            "name": "ok",
            "tags": ["dialogue"],
            "steps": [
                {"action": "send", "content": "one"},
                {"action": "send", "content": "two"},
                {"action": "send", "content": "three"},
                {"action": "assert_transcript_metrics", "cases": []},
            ],
        }
        errors = validate_scenario_shape("chat/ok.json", scenario)
        self.assertEqual(errors, [])

    def test_phrase_only_wait_rejected(self) -> None:
        scenario = {
            "expect_deliverables": [
                {"path": "x.go", "for_question": {"contains_all": ["Hello"]}}
            ],
            "steps": [
                {
                    "action": "send",
                    "content": "implement",
                    "metadata": {"editor_mode": "agent", "implementation_session": True},
                },
                {
                    "action": "wait_reply",
                    "until_any_match": ["implementation session complete"],
                },
            ],
        }
        errors = validate_implement_wait_gates("implement/phrase.json", scenario)
        self.assertTrue(any("until_any_match alone" in e for e in errors))

    def test_disk_meta_wait_ok(self) -> None:
        scenario = {
            "expect_deliverables": [
                {"path": "x.go", "for_question": {"contains_all": ["Hello"]}}
            ],
            "steps": [
                {
                    "action": "send",
                    "content": "implement",
                    "metadata": {"editor_mode": "agent", "implementation_session": True},
                },
                {
                    "action": "wait_reply",
                    "until_file_match": {"path": "x.go", "contains_all": ["Hello"]},
                    "until_metadata_keys": ["implementation_session_outcome"],
                },
            ],
        }
        self.assertEqual(validate_implement_wait_gates("implement/ok.json", scenario), [])


if __name__ == "__main__":
    unittest.main()
