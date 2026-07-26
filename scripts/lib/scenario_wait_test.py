"""Unit tests for scenario_wait disk conditions."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from scenario_wait import disk_wait_satisfied, step_has_disk_wait


class DiskWaitTest(unittest.TestCase):
    def test_and_semantics_file_exists_and_match(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "a.txt").write_text("hello world\n", encoding="utf-8")
            step = {
                "until_file_exists": "a.txt",
                "until_file_match": {"path": "a.txt", "contains": "hello"},
            }
            self.assertTrue(step_has_disk_wait(step))
            ok, detail = disk_wait_satisfied(root, step)
            self.assertTrue(ok, detail)

            step_bad = {
                "until_file_exists": "a.txt",
                "until_file_match": {"path": "a.txt", "contains": "missing"},
            }
            ok, _ = disk_wait_satisfied(root, step_bad)
            self.assertFalse(ok)

    def test_file_absent(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "gone.txt").write_text("x", encoding="utf-8")
            step = {"until_file_absent": "gone.txt"}
            ok, _ = disk_wait_satisfied(root, step)
            self.assertFalse(ok)
            (root / "gone.txt").unlink()
            ok, detail = disk_wait_satisfied(root, step)
            self.assertTrue(ok, detail)


if __name__ == "__main__":
    unittest.main()
