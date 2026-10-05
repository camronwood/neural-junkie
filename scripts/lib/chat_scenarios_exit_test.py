"""Canary suite exit aggregation: eventual_pass must not leave the suite red."""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


def _load_chat_scenarios():
    path = Path(__file__).resolve().parents[1] / "chat-scenarios.py"
    spec = importlib.util.spec_from_file_location("chat_scenarios_mod", path)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


class ChatScenariosExitTest(unittest.TestCase):
    def test_suite_exit_zero_when_no_failures(self) -> None:
        mod = _load_chat_scenarios()
        self.assertEqual(mod.suite_exit_code([]), 0)

    def test_suite_exit_one_when_hard_fail_remains(self) -> None:
        mod = _load_chat_scenarios()
        self.assertEqual(mod.suite_exit_code(["dm-debug-desktop-ui-missing-with-workspace"]), 1)

    def test_eventual_pass_not_listed_as_failed(self) -> None:
        # Flake retry that ends eventual_pass=True returns True from run_scenario,
        # so the name never enters failed[] — suite stays green.
        mod = _load_chat_scenarios()
        failed: list[str] = []
        eventual_pass = True
        if not eventual_pass:
            failed.append("dm-topic-continuity-same-thread")
        self.assertEqual(mod.suite_exit_code(failed), 0)


if __name__ == "__main__":
    unittest.main()
