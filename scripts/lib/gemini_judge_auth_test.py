"""Tests for Gemini judge model probing."""

from __future__ import annotations

import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.gemini_judge_auth import (  # noqa: E402
    GEMINI_JUDGE_PROBE_CANDIDATES,
    ensure_gemini_for_testing,
    list_gemini_api_keys,
    select_gemini_judge_model,
)


class TestListGeminiAPIKeys(unittest.TestCase):
    def test_file_lines_and_env(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            key_file = root / ".gemini-api-key"
            key_file.write_text("line-one\n# comment\n\nline-two\n", encoding="utf-8")
            with patch.dict(os.environ, {"GEMINI_API_KEY": "from-env"}, clear=False):
                keys = list_gemini_api_keys(root)
            self.assertEqual([label for label, _ in keys], ["env", "key-1", "key-2"])
            self.assertEqual(keys[0][1], "from-env")
            self.assertEqual(keys[1][1], "line-one")
            self.assertEqual(keys[2][1], "line-two")


class TestSelectGeminiJudgeModel(unittest.TestCase):
    def setUp(self) -> None:
        self._saved = os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL")
        os.environ.pop("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", None)
        self._tmpdir = tempfile.TemporaryDirectory()
        self.root = Path(self._tmpdir.name)
        (self.root / ".gemini-api-key").write_text("test-key\n", encoding="utf-8")
        os.environ.pop("GEMINI_API_KEY", None)

    def tearDown(self) -> None:
        self._tmpdir.cleanup()
        if self._saved is None:
            os.environ.pop("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", None)
        else:
            os.environ["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = self._saved

    @patch("lib.gemini_judge_auth.check_gemini_judge")
    def test_probe_tries_fast_then_pro(self, mock_check) -> None:
        mock_check.side_effect = [
            (False, "quota [gemini-2.5-flash]"),
            (True, "gemini-api-key auth OK (gemini-2.5-pro, PASS smoke)"),
        ]
        model, ok, detail = select_gemini_judge_model(
            timeout_s=5.0, retry_quota=False, root=self.root
        )
        self.assertTrue(ok)
        self.assertEqual(model, "gemini-2.5-pro")
        self.assertIn("pro", detail)
        self.assertEqual(mock_check.call_count, 2)
        mock_check.assert_any_call(
            timeout_s=5.0,
            model="gemini-2.5-flash",
            api_key="test-key",
            retry_quota=False,
        )
        mock_check.assert_any_call(
            timeout_s=5.0,
            model="gemini-2.5-pro",
            api_key="test-key",
            retry_quota=False,
        )

    @patch("lib.gemini_judge_auth.check_gemini_judge")
    def test_probe_all_fail(self, mock_check) -> None:
        mock_check.return_value = (False, "quota")
        model, ok, detail = select_gemini_judge_model(
            timeout_s=5.0, retry_quota=False, root=self.root
        )
        self.assertFalse(ok)
        self.assertIsNone(model)
        self.assertEqual(mock_check.call_count, len(GEMINI_JUDGE_PROBE_CANDIDATES))

    @patch("lib.gemini_judge_auth.check_gemini_judge")
    def test_explicit_model_only(self, mock_check) -> None:
        mock_check.return_value = (True, "ok")
        model, ok, _detail = select_gemini_judge_model(
            timeout_s=5.0,
            explicit_model="gemini-2.5-flash-lite",
            root=self.root,
        )
        self.assertTrue(ok)
        self.assertEqual(model, "gemini-2.5-flash-lite")
        mock_check.assert_called_once_with(
            timeout_s=5.0,
            model="gemini-2.5-flash-lite",
            api_key="test-key",
            retry_quota=False,
        )

    @patch("lib.gemini_judge_auth.check_gemini_judge")
    def test_ensure_tries_second_key(self, mock_check) -> None:
        (self.root / ".gemini-api-key").write_text("bad-key\ngood-key\n", encoding="utf-8")
        mock_check.side_effect = [
            (False, "quota key-1"),
            (False, "quota key-1"),
            (False, "quota key-1"),
            (True, "gemini-api-key auth OK (gemini-2.5-flash, PASS smoke)"),
        ]
        sel = ensure_gemini_for_testing(root=self.root, timeout_s=5.0, retry_quota=False)
        self.assertTrue(sel.ok)
        self.assertEqual(sel.api_key_label, "key-2")
        self.assertEqual(sel.model, "gemini-2.5-flash")
        self.assertEqual(os.environ.get("GEMINI_API_KEY"), "good-key")


if __name__ == "__main__":
    unittest.main()
