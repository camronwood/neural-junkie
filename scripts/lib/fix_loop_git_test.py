#!/usr/bin/env python3
"""Unit tests for fix-loop git helpers."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.fix_loop_git import (  # noqa: E402
    default_fix_branch,
    is_artifact_path,
    list_commit_candidates,
)


class FixLoopGitTest(unittest.TestCase):
    def test_default_fix_branch(self) -> None:
        self.assertEqual(default_fix_branch("2026-06-28-1500"), "release-prep/fix-2026-06-28-1500")

    def test_is_artifact_path(self) -> None:
        self.assertTrue(is_artifact_path("docs/testing/release-prep-2026-06-27.md"))
        self.assertTrue(is_artifact_path("docs/testing/release-prep-fix-loop-iter1.md"))
        self.assertFalse(is_artifact_path("internal/hub/hub.go"))
        self.assertFalse(is_artifact_path("scripts/release-prep-fix-loop.py"))

    def test_list_commit_candidates_excludes_artifacts(self) -> None:
        root = SCRIPTS_DIR.parent
        if not (root / ".git").exists():
            self.skipTest("not a git checkout")
        candidates = list_commit_candidates(root)
        for path in candidates:
            self.assertFalse(is_artifact_path(path), path)


if __name__ == "__main__":
    raise SystemExit(unittest.main())
