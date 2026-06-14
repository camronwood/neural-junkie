"""Unit tests for scenario deliverable contract validation."""

from __future__ import annotations

import unittest

from scenario_contract import validate_deliverable_contract


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


if __name__ == "__main__":
    unittest.main()
