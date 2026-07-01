"""Git helpers for release-prep fix loop (branch + iteration commits)."""

from __future__ import annotations

import re
import subprocess
from datetime import datetime, timezone
from pathlib import Path

# Generated run artifacts — never auto-commit these.
TESTING_ARTIFACT_PREFIXES = (
    "docs/testing/release-prep-",
    "docs/testing/release-prep-fix-loop-",
    "docs/testing/test-everything-",
    "docs/testing/hub-recovery-",
    "docs/testing/parity-stable",
    "docs/testing/model-benchmark-",
)

BRANCH_SAFE_RE = re.compile(r"[^a-zA-Z0-9._/-]+")


def default_fix_branch(stamp: str | None = None) -> str:
    ts = stamp or datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    return f"release-prep/fix-{ts}"


def _run_git(args: list[str], *, cwd: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=cwd,
        capture_output=True,
        text=True,
        check=False,
    )


def git_available(cwd: Path) -> bool:
    return _run_git(["rev-parse", "--git-dir"], cwd=cwd).returncode == 0


def current_branch(cwd: Path) -> str:
    proc = _run_git(["branch", "--show-current"], cwd=cwd)
    return (proc.stdout or "").strip()


def is_artifact_path(path: str) -> bool:
    normalized = path.replace("\\", "/")
    return any(normalized.startswith(prefix) for prefix in TESTING_ARTIFACT_PREFIXES)


def list_commit_candidates(cwd: Path) -> list[str]:
    """Return changed paths suitable for auto-commit (exclude testing run artifacts)."""
    proc = _run_git(["status", "--porcelain"], cwd=cwd)
    if proc.returncode != 0:
        return []
    paths: list[str] = []
    for line in (proc.stdout or "").splitlines():
        if len(line) < 4:
            continue
        path = line[3:].strip()
        if " -> " in path:
            path = path.split(" -> ", 1)[1]
        if not path or is_artifact_path(path):
            continue
        paths.append(path)
    return paths


def ensure_fix_branch(
    cwd: Path,
    *,
    branch: str | None = None,
    base_branch: str | None = None,
) -> tuple[int, str]:
    """Create or checkout fix branch. Returns (exit_code, branch_name)."""
    if not git_available(cwd):
        return 1, branch or ""

    name = branch or default_fix_branch()
    name = BRANCH_SAFE_RE.sub("-", name).strip("-/")

    if current_branch(cwd) == name:
        print(f">>> [git] already on branch {name}")
        return 0, name

    exists = _run_git(["rev-parse", "--verify", name], cwd=cwd).returncode == 0
    if exists:
        proc = _run_git(["checkout", name], cwd=cwd)
        if proc.returncode != 0:
            err = (proc.stderr or proc.stdout or "").strip()
            return proc.returncode, name
        print(f">>> [git] checked out existing branch {name}")
        return 0, name

    if base_branch:
        proc = _run_git(["checkout", "-b", name, base_branch], cwd=cwd)
    else:
        proc = _run_git(["checkout", "-b", name], cwd=cwd)
    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()
        print(f">>> [git] failed to create branch {name}: {err}", flush=True)
        return proc.returncode, name
    print(f">>> [git] created branch {name}")
    return 0, name


def commit_iteration_changes(
    cwd: Path,
    *,
    branch: str,
    iteration: int,
    summary_path: Path | None = None,
    dry_run: bool = False,
) -> tuple[int, str]:
    """Stage non-artifact changes and commit. Returns (exit_code, message)."""
    if not git_available(cwd):
        return 1, "not a git repository"

    paths = list_commit_candidates(cwd)
    if not paths:
        print(">>> [git] no commit candidates (only testing artifacts or clean tree)")
        return 0, ""

    summary_note = summary_path.name if summary_path else "release-prep"
    message = (
        f"fix(release-prep): iteration {iteration} from {summary_note}\n\n"
        "Auto-commit from release-prep-fix-loop Cursor agent pass."
    )

    if dry_run:
        print(f">>> [git] dry-run would commit {len(paths)} path(s) on {branch}:")
        for p in paths[:20]:
            print(f"  - {p}")
        if len(paths) > 20:
            print(f"  ... and {len(paths) - 20} more")
        return 0, message

    add = _run_git(["add", "--", *paths], cwd=cwd)
    if add.returncode != 0:
        err = (add.stderr or add.stdout or "").strip()
        print(f">>> [git] add failed: {err}", flush=True)
        return add.returncode, message

    commit = _run_git(["commit", "-m", message], cwd=cwd)
    if commit.returncode != 0:
        err = (commit.stderr or commit.stdout or "").strip()
        print(f">>> [git] commit failed: {err}", flush=True)
        return commit.returncode, message

    sha = _run_git(["rev-parse", "--short", "HEAD"], cwd=cwd).stdout.strip()
    print(f">>> [git] committed {sha} on {branch} ({len(paths)} files)")
    return 0, message
