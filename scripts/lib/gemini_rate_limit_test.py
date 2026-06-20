"""Tests for Gemini API pacing."""

from __future__ import annotations

import os
import sys
import time
import unittest
from pathlib import Path
from unittest import mock

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.gemini_rate_limit import gemini_min_interval_s, throttle_gemini_api_call  # noqa: E402


class TestGeminiRateLimit(unittest.TestCase):
    def test_min_interval_unset(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            self.assertIsNone(gemini_min_interval_s())

    def test_throttle_waits(self) -> None:
        with mock.patch.dict(os.environ, {"NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S": "0.2"}, clear=False):
            throttle_gemini_api_call()
            t0 = time.monotonic()
            throttle_gemini_api_call()
            elapsed = time.monotonic() - t0
            self.assertGreaterEqual(elapsed, 0.15)


if __name__ == "__main__":
    unittest.main()
