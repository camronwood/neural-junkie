"""Tests for scripts/lib/proc_timeout.py."""

from __future__ import annotations

import subprocess
import sys
import unittest

from lib.proc_timeout import wait_with_timeout


class ProcTimeoutTest(unittest.TestCase):
    def test_wait_with_timeout_kills_hung_process(self) -> None:
        proc = subprocess.Popen(
            [sys.executable, "-c", "import time; time.sleep(30)"],
            start_new_session=True,
        )
        rc, timed_out = wait_with_timeout(proc, 1.0, tick_s=0.2)
        self.assertTrue(timed_out)
        self.assertIsNotNone(proc.poll())
        self.assertNotEqual(rc, 0)

    def test_wait_with_timeout_returns_quick_exit(self) -> None:
        proc = subprocess.Popen(
            [sys.executable, "-c", "raise SystemExit(7)"],
            start_new_session=True,
        )
        rc, timed_out = wait_with_timeout(proc, 10.0)
        self.assertFalse(timed_out)
        self.assertEqual(rc, 7)


if __name__ == "__main__":
    unittest.main()
