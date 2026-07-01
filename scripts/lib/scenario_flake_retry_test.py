"""Unit tests for scenario_flake_retry."""

from __future__ import annotations

import os
import unittest
from unittest import mock

from lib import scenario_flake_retry as sfr


class ScenarioFlakeRetryTest(unittest.TestCase):
    def setUp(self) -> None:
        self._env = dict(os.environ)

    def tearDown(self) -> None:
        os.environ.clear()
        os.environ.update(self._env)

    def test_retryable_timeout(self) -> None:
        self.assertTrue(sfr.is_retryable_failure("timeout waiting for BackendEngineer"))

    def test_retryable_401(self) -> None:
        self.assertTrue(sfr.is_retryable_failure("send failed (401)"))

    def test_not_retryable_assertion(self) -> None:
        self.assertFalse(sfr.is_retryable_failure("metadata outcome: got 'proposals_submitted'"))

    def test_disabled_via_env(self) -> None:
        os.environ["NJ_SCENARIO_FLAKE_RETRY"] = "0"
        self.assertFalse(sfr.flake_retry_enabled())
        self.assertFalse(
            sfr.maybe_retry_after_failure(
                "http://127.0.0.1:18765",
                "go-handler",
                "timeout waiting for BackendEngineer",
                1,
            )
        )

    def test_maybe_retry_when_enabled(self) -> None:
        os.environ["NJ_SCENARIO_FLAKE_RETRY"] = "1"
        with mock.patch.object(sfr, "refresh_auth_for_retry"), mock.patch.object(
            sfr, "pause_before_retry"
        ):
            self.assertTrue(
                sfr.maybe_retry_after_failure(
                    "http://127.0.0.1:18765",
                    "go-handler",
                    "timeout waiting for BackendEngineer",
                    1,
                )
            )


if __name__ == "__main__":
    unittest.main()
