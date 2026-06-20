"""Tests for release-prep environment bootstrap."""

from __future__ import annotations

import os
import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.release_prep_env import (  # noqa: E402
    load_gemini_api_key,
    parse_env_file,
    release_prep_env,
)


class TestReleasePrepEnv(unittest.TestCase):
    def test_parse_env_file(self) -> None:
        root = SCRIPTS_DIR.parent
        parsed = parse_env_file(root / "env.example")
        self.assertIn("SERVER_PORT", parsed)

    def test_release_prep_env_sets_rate_limit(self) -> None:
        env = release_prep_env(SCRIPTS_DIR.parent)
        self.assertEqual(env.get("NEURAL_JUNKIE_RATE_LIMIT"), "0")

    def test_release_prep_env_cloud_first_with_fallback(self) -> None:
        env = release_prep_env(SCRIPTS_DIR.parent)
        self.assertEqual(env.get("NJ_DELIVERABLE_JUDGE_PROVIDER"), "gemini")
        self.assertEqual(env.get("NJ_DELIVERABLE_JUDGE_MODE"), "hub")
        self.assertEqual(env.get("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA"), "1")
        self.assertEqual(env.get("NJ_DELIVERABLE_JUDGE_MODEL"), "qwen2.5-coder:14b")

    def test_load_gemini_from_file_when_env_empty(self) -> None:
        key_path = SCRIPTS_DIR.parent / ".gemini-api-key"
        if not key_path.is_file():
            self.skipTest(".gemini-api-key not present")
        old = os.environ.pop("GEMINI_API_KEY", None)
        try:
            key = load_gemini_api_key(SCRIPTS_DIR.parent)
            self.assertTrue(key)
        finally:
            if old:
                os.environ["GEMINI_API_KEY"] = old


if __name__ == "__main__":
    unittest.main()
