"""Unit tests for scenario flake retry helpers."""

from __future__ import annotations

import unittest

from lib.scenario_flake_retry import detail_is_flake, output_is_flake


class ScenarioFlakeRetryTest(unittest.TestCase):
    def test_implement_timeout_is_flake(self) -> None:
        self.assertTrue(
            detail_is_flake("timeout waiting for FrontendEngineer", kind="implement")
        )

    def test_collab_silence_is_flake(self) -> None:
        self.assertTrue(
            output_is_flake(
                "FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)",
                kind="collab",
            )
        )

    def test_assertion_fail_is_not_flake(self) -> None:
        self.assertFalse(
            detail_is_flake("metadata 'implementation_session_outcome.outcome': got None", kind="implement")
        )


if __name__ == "__main__":
    unittest.main()
