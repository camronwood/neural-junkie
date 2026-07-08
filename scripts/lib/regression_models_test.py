"""Tests for regression model policy."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path
from unittest import mock

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.regression_models import (  # noqa: E402
    DEFAULT_REGRESSION_AGENT_MODEL,
    agents_over_regression_cap,
    filter_regression_models,
    is_regression_allowed_model,
    model_params_b,
    resolve_regression_agent_model,
)


class RegressionModelsTest(unittest.TestCase):
    def test_model_params_from_tag(self) -> None:
        self.assertEqual(model_params_b("qwen3.5:27b"), 27.0)
        self.assertEqual(model_params_b("qwen2.5-coder:14b"), 14.0)
        self.assertEqual(model_params_b("qwen3.5:9b"), 9.0)

    def test_regression_cap(self) -> None:
        self.assertFalse(is_regression_allowed_model("qwen3.5:27b"))
        self.assertFalse(is_regression_allowed_model("devstral:24b"))
        self.assertTrue(is_regression_allowed_model("qwen2.5-coder:14b"))
        self.assertTrue(is_regression_allowed_model("gemma3:12b"))
        self.assertTrue(is_regression_allowed_model("qwen3.5:9b"))

    def test_filter_regression_models(self) -> None:
        tags = filter_regression_models(
            ["qwen3.5:9b", "qwen3.5:27b", "qwen2.5-coder:14b", "devstral:24b"]
        )
        self.assertEqual(tags, ["qwen3.5:9b", "qwen2.5-coder:14b"])

    def test_resolve_regression_agent_model_prefers_allowed_env(self) -> None:
        root = SCRIPTS_DIR.parent
        with mock.patch.dict(
            "os.environ",
            {"NJ_REGRESSION_AGENT_MODEL": "qwen3.5:27b", "OLLAMA_CODE_MODEL": "qwen3.5:9b"},
            clear=False,
        ):
            self.assertEqual(resolve_regression_agent_model(root), "qwen3.5:9b")

    def test_resolve_regression_agent_model_default(self) -> None:
        root = SCRIPTS_DIR.parent
        with mock.patch("lib.release_prep_env.parse_env_file", return_value={}):
            with mock.patch.dict("os.environ", {}, clear=True):
                self.assertEqual(resolve_regression_agent_model(root), DEFAULT_REGRESSION_AGENT_MODEL)

    def test_agents_over_regression_cap(self) -> None:
        payload = [
            {"name": "BackendEngineer", "ai_provider": "ollama", "ai_model": "qwen3.5:27b"},
            {"name": "Assistant", "ai_provider": "ollama", "ai_model": "qwen3.5:9b"},
        ]
        with mock.patch("lib.collab_hub.hub_request", return_value=(200, payload)):
            over = agents_over_regression_cap("http://127.0.0.1:18765")
        self.assertEqual(len(over), 1)
        self.assertIn("BackendEngineer", over[0])


if __name__ == "__main__":
    unittest.main()
