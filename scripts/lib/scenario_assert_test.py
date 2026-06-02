"""Unit tests for scenario_assert helpers."""

from __future__ import annotations

import unittest

from scenario_assert import looks_like_stack_tool_command


class StackToolCommandTest(unittest.TestCase):
    def test_detects_stack_tools(self) -> None:
        for cmd in ("docker-compose up -d", "npm install", "kubectl get pods", "make build"):
            self.assertTrue(looks_like_stack_tool_command(cmd), cmd)

    def test_allows_read_only(self) -> None:
        for cmd in ("cat README.md", "grep schema resource-api/", "find . -name '*.md'"):
            self.assertFalse(looks_like_stack_tool_command(cmd), cmd)


if __name__ == "__main__":
    unittest.main()
