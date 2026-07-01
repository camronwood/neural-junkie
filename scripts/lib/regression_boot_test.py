"""Tests for regression boot helpers."""

from __future__ import annotations

import os
import unittest
from unittest.mock import patch

from lib.regression_boot import BOOT_DONE_ENV, boot_regression_stack, should_skip_boot
from pathlib import Path


class RegressionBootTest(unittest.TestCase):
    def tearDown(self) -> None:
        for key in ("SKIP_BOOT", BOOT_DONE_ENV):
            os.environ.pop(key, None)

    def test_should_skip_boot_when_flag_set(self) -> None:
        os.environ["SKIP_BOOT"] = "1"
        self.assertTrue(should_skip_boot())

    def test_boot_skips_when_already_done(self) -> None:
        os.environ[BOOT_DONE_ENV] = "1"
        with patch("lib.regression_boot.clean_environment") as clean:
            self.assertTrue(boot_regression_stack(Path("."), "http://127.0.0.1:18765"))
            clean.assert_not_called()


if __name__ == "__main__":
    unittest.main()
