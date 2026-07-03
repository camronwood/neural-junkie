#!/usr/bin/env python3
"""Unit tests for fix-loop git helpers."""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.fix_loop_git import (  # noqa: E402
    WORKTREES_DIRNAME,
    current_branch,
    default_fix_branch,
    ensure_fix_worktree,
    is_artifact_path,
    list_commit_candidates,
    normalize_branch_name,
    planned_fix_worktree_path,
    prepare_fix_loop_cwd,
    sanitize_branch_for_path,
)


def _run_git(args: list[str], *, cwd: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["git", *args], cwd=cwd, capture_output=True, text=True, check=False)


def _init_repo(root: Path, *, branch: str = "main") -> None:
    _run_git(["init", "-b", branch], cwd=root)
    _run_git(["config", "user.email", "test@example.com"], cwd=root)
    _run_git(["config", "user.name", "Test"], cwd=root)
    (root / "README.md").write_text("initial\n", encoding="utf-8")
    _run_git(["add", "README.md"], cwd=root)
    _run_git(["commit", "-m", "init"], cwd=root)


class FixLoopGitTest(unittest.TestCase):
    def test_default_fix_branch(self) -> None:
        self.assertEqual(default_fix_branch("2026-06-28-1500"), "release-prep/fix-2026-06-28-1500")

    def test_sanitize_branch_for_path(self) -> None:
        self.assertEqual(
            sanitize_branch_for_path("release-prep/layer-implement-2026-07-02"),
            "release-prep-layer-implement-2026-07-02",
        )

    def test_normalize_branch_name_keeps_slashes(self) -> None:
        self.assertEqual(
            normalize_branch_name("release-prep/layer-implement-2026-07-02"),
            "release-prep/layer-implement-2026-07-02",
        )

    def test_planned_fix_worktree_path(self) -> None:
        root = Path("/tmp/repo")
        branch = "release-prep/layer-chat-2026-07-02"
        expected = root.resolve() / WORKTREES_DIRNAME / "release-prep-layer-chat-2026-07-02"
        self.assertEqual(planned_fix_worktree_path(root, branch), expected)

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

    def test_ensure_fix_worktree_leaves_main_checkout_alone(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / "repo"
            repo.mkdir()
            _init_repo(repo)
            self.assertEqual(current_branch(repo), "main")

            branch = "release-prep/layer-implement-test"
            rc, worktree_path, fix_branch = ensure_fix_worktree(repo, branch=branch, base_branch="main")

            self.assertEqual(rc, 0)
            self.assertEqual(fix_branch, branch)
            self.assertTrue(worktree_path.is_dir())
            self.assertEqual(current_branch(repo), "main")
            self.assertEqual(current_branch(worktree_path), branch)
            self.assertEqual(
                worktree_path,
                planned_fix_worktree_path(repo, branch),
            )

    def test_ensure_fix_worktree_reuses_existing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / "repo"
            repo.mkdir()
            _init_repo(repo)

            branch = "release-prep/fix-test"
            rc1, path1, _ = ensure_fix_worktree(repo, branch=branch)
            rc2, path2, _ = ensure_fix_worktree(repo, branch=branch)

            self.assertEqual(rc1, 0)
            self.assertEqual(rc2, 0)
            self.assertEqual(path1, path2)
            self.assertEqual(current_branch(repo), "main")

    def test_prepare_fix_loop_cwd_with_worktree(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / "repo"
            repo.mkdir()
            _init_repo(repo)

            rc, loop_cwd, fix_branch = prepare_fix_loop_cwd(
                repo,
                branch="release-prep/layer-chat-test",
                base_branch="main",
                use_worktree=True,
                no_commit=False,
                dry_run=False,
            )

            self.assertEqual(rc, 0)
            self.assertNotEqual(loop_cwd.resolve(), repo.resolve())
            self.assertEqual(fix_branch, "release-prep/layer-chat-test")
            self.assertEqual(current_branch(repo), "main")
            self.assertEqual(current_branch(loop_cwd), fix_branch)

    def test_prepare_fix_loop_cwd_without_worktree_checks_out_branch(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / "repo"
            repo.mkdir()
            _init_repo(repo)

            rc, loop_cwd, fix_branch = prepare_fix_loop_cwd(
                repo,
                branch="release-prep/fix-inline",
                base_branch="main",
                use_worktree=False,
                no_commit=False,
                dry_run=False,
            )

            self.assertEqual(rc, 0)
            self.assertEqual(loop_cwd.resolve(), repo.resolve())
            self.assertEqual(fix_branch, "release-prep/fix-inline")
            self.assertEqual(current_branch(repo), fix_branch)


if __name__ == "__main__":
    raise SystemExit(unittest.main())
