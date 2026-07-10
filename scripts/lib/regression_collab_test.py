"""Tests for collab regression tuning helpers."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.regression_collab import (  # noqa: E402
    apply_core_scenario_defaults,
    is_collab_core_scenario,
)


class TestRegressionCollab(unittest.TestCase):
    def test_is_collab_core_scenario(self) -> None:
        self.assertTrue(is_collab_core_scenario("planning-two-agent"))
        self.assertFalse(is_collab_core_scenario("make-me-a-website"))

    def test_apply_core_scenario_defaults(self) -> None:
        scenario = {
            "steps": [
                {"action": "wait_discussion", "timeout": "180s"},
                {"action": "assert_messages"},
            ]
        }
        out = apply_core_scenario_defaults(scenario)
        wait = out["steps"][0]
        assert_msg = out["steps"][1]
        self.assertTrue(wait.get("retry_on_generation_error"))
        self.assertTrue(assert_msg.get("allow_recovered_generation_errors"))
        self.assertTrue(assert_msg.get("deny_generation_errors"))


if __name__ == "__main__":
    unittest.main()
