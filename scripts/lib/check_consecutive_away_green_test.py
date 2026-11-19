#!/usr/bin/env python3
"""Unit tests for check-consecutive-away-green helpers."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "check-consecutive-away-green.py"


def load_mod():
    spec = importlib.util.spec_from_file_location("check_consecutive_away_green", SCRIPT)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


class TestConsecutiveAwayGreen(unittest.TestCase):
    def test_green_and_red(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as td:
            p = Path(td)
            green = p / "away-morning-2026-11-17.md"
            green.write_text(
                "| Gate | Result | Notes |\n|------|--------|-------|\n| `make test-all` | **PASS** | — |\n",
                encoding="utf-8",
            )
            red = p / "away-morning-2026-11-16.md"
            red.write_text(
                "| Gate | Result | Notes |\n|------|--------|-------|\n| `make test-all` | **FAIL** | boom |\n",
                encoding="utf-8",
            )
            self.assertTrue(mod.report_is_green(green))
            self.assertFalse(mod.report_is_green(red))


if __name__ == "__main__":
    unittest.main()
