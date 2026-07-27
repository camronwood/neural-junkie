"""Unit tests for scenario_wait disk conditions."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

_LIB_DIR = Path(__file__).resolve().parent
if str(_LIB_DIR) not in sys.path:
    sys.path.insert(0, str(_LIB_DIR))

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

    def test_until_file_match_path_alternatives(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "src").mkdir()
            (root / "src" / "lib.rs").write_text(
                "fn hit() {}\nfn stand() {}\nfn dealer() {}\n",
                encoding="utf-8",
            )
            step = {
                "until_file_match": {
                    "path": "src/main.rs",
                    "path_alternatives": ["src/lib.rs"],
                    "contains_all": ["hit|stand", "dealer|Dealer"],
                }
            }
            ok, detail = disk_wait_satisfied(root, step)
            self.assertTrue(ok, detail)

    def test_until_file_match_contains_all_regex(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "index.html").write_text(
                "<h1>Neural Junkie</h1><p>Local multi-agent hub</p>",
                encoding="utf-8",
            )
            step = {
                "until_file_match": {
                    "path": "index.html",
                    "contains_all": ["Neural Junkie", "Local multi-agent hub"],
                    "none_match": ["Brightest Bio"],
                }
            }
            ok, detail = disk_wait_satisfied(root, step)
            self.assertTrue(ok, detail)

            step_bad = {
                "until_file_match": {
                    "path": "index.html",
                    "contains_all": [r"README\.md", r"core/sample/main\.go"],
                }
            }
            ok, _ = disk_wait_satisfied(root, step_bad)
            self.assertFalse(ok)

    def test_until_file_match_min_bytes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "tiny.txt").write_text("x", encoding="utf-8")
            step = {
                "until_file_match": {
                    "path": "tiny.txt",
                    "min_bytes": 10,
                }
            }
            ok, _ = disk_wait_satisfied(root, step)
            self.assertFalse(ok)
            (root / "tiny.txt").write_text("x" * 20, encoding="utf-8")
            ok, detail = disk_wait_satisfied(root, step)
            self.assertTrue(ok, detail)


if __name__ == "__main__":
    unittest.main()
