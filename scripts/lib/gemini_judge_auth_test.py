"""Tests for Gemini judge model probing."""

from __future__ import annotations

import os
import sys
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.gemini_judge_auth import (  # noqa: E402
    GEMINI_JUDGE_PROBE_CANDIDATES,
    select_gemini_judge_model,
)


class TestSelectGeminiJudgeModel(unittest.TestCase):
    def setUp(self) -> None:
        self._saved = os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL")
        os.environ.pop("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", None)

    def tearDown(self) -> None:
        if self._saved is None:
            os.environ.pop("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", None)
        else:
            os.environ["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = self._saved

    @patch.dict(os.environ, {"GEMINI_API_KEY": "test-key"}, clear=False)
    @patch("lib.gemini_judge_auth.check_gemini_judge")
    def test_probe_tries_fast_then_pro(self, mock_check) -> None:
        mock_check.side_effect = [
            (False, "quota [gemini-2.5-flash]"),
            (True, "gemini-api-key auth OK (gemini-2.5-pro, PASS smoke)"),
        ]
        model, ok, detail = select_gemini_judge_model(timeout_s=5.0, retry_quota=False)
        self.assertTrue(ok)
        self.assertEqual(model, "gemini-2.5-pro")
        self.assertIn("pro", detail)
        self.assertEqual(mock_check.call_count, 2)
        mock_check.assert_any_call(
            timeout_s=5.0,
            model="gemini-2.5-flash",
            retry_quota=False,
        )
        mock_check.assert_any_call(
            timeout_s=5.0,
            model="gemini-2.5-pro",
            retry_quota=False,
        )

    @patch.dict(os.environ, {"GEMINI_API_KEY": "test-key"}, clear=False)
    @patch("lib.gemini_judge_auth.check_gemini_judge")
    def test_probe_all_fail(self, mock_check) -> None:
        mock_check.return_value = (False, "quota")
        model, ok, detail = select_gemini_judge_model(timeout_s=5.0, retry_quota=False)
        self.assertFalse(ok)
        self.assertIsNone(model)
        self.assertEqual(mock_check.call_count, len(GEMINI_JUDGE_PROBE_CANDIDATES))

    @patch.dict(os.environ, {"GEMINI_API_KEY": "test-key"}, clear=False)
    @patch("lib.gemini_judge_auth.check_gemini_judge")
    def test_explicit_model_only(self, mock_check) -> None:
        mock_check.return_value = (True, "ok")
        model, ok, _detail = select_gemini_judge_model(
            timeout_s=5.0,
            explicit_model="gemini-2.5-flash-lite",
        )
        self.assertTrue(ok)
        self.assertEqual(model, "gemini-2.5-flash-lite")
        mock_check.assert_called_once_with(
            timeout_s=5.0,
            model="gemini-2.5-flash-lite",
            retry_quota=True,
        )


if __name__ == "__main__":
    unittest.main()
