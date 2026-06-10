"""Unit tests for model benchmark report helpers."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.model_benchmark import (  # noqa: E402
    ModelBenchmarkResult,
    ScenarioResult,
    format_duration,
    model_is_installed,
    render_markdown_report,
    resolve_suite_scenarios,
)


class TestModelBenchmark(unittest.TestCase):
    def test_format_duration(self) -> None:
        self.assertEqual(format_duration(45), "45s")
        self.assertEqual(format_duration(125), "2m05s")

    def test_model_is_installed(self) -> None:
        installed = {"qwen2.5-coder:14b", "qwen3.5:9b"}
        self.assertTrue(model_is_installed(installed, "qwen2.5-coder:14b"))
        self.assertFalse(model_is_installed(installed, "codestral:22b"))

    def test_ollama_installed_tags_from_object(self) -> None:
        from lib.model_benchmark import ollama_installed_tags
        import unittest.mock as mock

        with mock.patch("lib.collab_hub.hub_request", return_value=(200, {"models": ["codestral:22b", "devstral:24b"]})):
            tags = ollama_installed_tags("http://127.0.0.1:18765")
        self.assertIn("codestral:22b", tags)
        self.assertIn("devstral:24b", tags)

    def test_resolve_quick_suite(self) -> None:
        suite = {
            "implement": ["go-handler", "theme-toggle"],
            "chat": ["dm-backend-workspace"],
        }
        impl, chat = resolve_suite_scenarios(suite)
        self.assertEqual(impl, ["go-handler", "theme-toggle"])
        self.assertEqual(chat, ["dm-backend-workspace"])

    def test_render_markdown_includes_winner(self) -> None:
        r = ModelBenchmarkResult(
            model_id="a",
            model_tag="qwen2.5-coder:14b",
            title="Qwen 2.5 Coder 14B",
            size_hint_gb=9,
            scenarios=[
                ScenarioResult("go-handler", "implement", True, 120.0),
                ScenarioResult("dm-backend-workspace", "chat", True, 30.0),
            ],
            total_duration_s=150.0,
        )
        md = render_markdown_report(
            suite_name="quick",
            suite_desc="smoke",
            hub_url="http://127.0.0.1:18765",
            results=[r],
            implement_names=["go-handler"],
            chat_names=["dm-backend-workspace"],
        )
        self.assertIn("qwen2.5-coder:14b", md)
        self.assertIn("winner", md)


if __name__ == "__main__":
    unittest.main()
