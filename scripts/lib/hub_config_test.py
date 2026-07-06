"""Tests for hub_config automation env bridging."""

from __future__ import annotations

import os
import unittest
from unittest import mock

from lib.hub_config import apply_automation_to_env, env_or_automation


class HubConfigTest(unittest.TestCase):
    def test_env_or_automation_prefers_env(self) -> None:
        with mock.patch.dict(os.environ, {"NJ_DELIVERABLE_JUDGE_PROVIDER": "ollama"}, clear=False):
            with mock.patch("lib.hub_config.automation_config", return_value={"deliverable_judge_provider": "claude"}):
                self.assertEqual(env_or_automation("NJ_DELIVERABLE_JUDGE_PROVIDER", "deliverable_judge_provider", "claude"), "ollama")

    def test_apply_automation_to_env_fills_unset(self) -> None:
        env: dict[str, str] = {}
        auto = {
            "deliverable_judge_provider": "claude",
            "deliverable_judge_mode": "hub",
            "deliverable_judge_fallback_ollama": True,
        }
        with mock.patch("lib.hub_config.automation_config", return_value=auto):
            apply_automation_to_env(env)
        self.assertEqual(env.get("NJ_DELIVERABLE_JUDGE_PROVIDER"), "claude")
        self.assertEqual(env.get("NJ_DELIVERABLE_JUDGE_MODE"), "hub")
        self.assertEqual(env.get("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA"), "1")


if __name__ == "__main__":
    unittest.main()
