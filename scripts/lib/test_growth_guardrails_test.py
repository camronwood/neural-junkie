"""Unit tests for test_growth_guardrails."""

from __future__ import annotations

import json
import subprocess
import tempfile
import unittest
from pathlib import Path

from test_growth_guardrails import (
    check_edit_guardrails,
    detect_assertion_weakening,
    is_allowed_edit_path,
    is_product_code_path,
)


class AllowlistTest(unittest.TestCase):
    def test_allows_go_test_files(self) -> None:
        self.assertTrue(is_allowed_edit_path("internal/agent/foo_test.go"))

    def test_allows_scenario_json(self) -> None:
        self.assertTrue(is_allowed_edit_path("scenarios/chat/dm-greeting.json"))

    def test_blocks_product_go(self) -> None:
        self.assertFalse(is_allowed_edit_path("internal/agent/assistant_agent.go"))
        self.assertTrue(is_product_code_path("internal/agent/assistant_agent.go"))


class AssertionWeakeningTest(unittest.TestCase):
    def test_detects_removed_none_match(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            cwd = Path(tmp)
            subprocess.run(["git", "init"], cwd=cwd, capture_output=True, check=True)
            subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=cwd, check=True)
            subprocess.run(["git", "config", "user.name", "Test"], cwd=cwd, check=True)

            scenario_dir = cwd / "scenarios" / "chat"
            scenario_dir.mkdir(parents=True)
            before = {
                "name": "test-scenario",
                "steps": [
                    {
                        "action": "assert_messages",
                        "none_match": ["bad-pattern"],
                    }
                ],
            }
            path = scenario_dir / "test-scenario.json"
            path.write_text(json.dumps(before), encoding="utf-8")
            subprocess.run(["git", "add", "."], cwd=cwd, check=True)
            subprocess.run(["git", "commit", "-m", "init"], cwd=cwd, check=True)

            after = {"name": "test-scenario", "steps": [{"action": "assert_messages"}]}
            path.write_text(json.dumps(after), encoding="utf-8")

            rel = "scenarios/chat/test-scenario.json"
            violations = detect_assertion_weakening(cwd, [rel])
            self.assertTrue(any("none_match" in v for v in violations))

    def test_guardrails_fail_on_product_only_edit(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            cwd = Path(tmp)
            result = check_edit_guardrails(
                cwd,
                changed_paths=["internal/hub/server.go"],
            )
            self.assertFalse(result.ok)
            self.assertTrue(result.product_code_touched)


if __name__ == "__main__":
    unittest.main()
