#!/usr/bin/env python3
"""Claim one agent-ready GitHub issue, run Cursor agent, open agent-pr with auto-merge."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
PY = sys.executable

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.cursor_fix_agent import (  # noqa: E402
    DEFAULT_AGENT_TIMEOUT_S,
    run_fix_agent,
)
from lib.fix_loop_git import (  # noqa: E402
    commit_iteration_changes,
    list_commit_candidates,
    prepare_fix_loop_cwd,
)
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402

LABEL_READY = "agent-ready"
LABEL_WORKING = "agent-working"
LABEL_PR = "agent-pr"
LABEL_BLOCKED = "agent-blocked"


def run(cmd: list[str], *, cwd: Path = ROOT, check: bool = False) -> subprocess.CompletedProcess[str]:
    env = release_prep_env(ROOT)
    env["PYTHONUNBUFFERED"] = "1"
    print(f"\n>>> {' '.join(cmd)}", flush=True)
    proc = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, env=env)
    if proc.stdout:
        sys.stdout.write(proc.stdout)
    if proc.stderr:
        sys.stderr.write(proc.stderr)
    if check and proc.returncode != 0:
        raise SystemExit(proc.returncode)
    return proc


def gh(args: list[str], *, cwd: Path = ROOT) -> subprocess.CompletedProcess[str]:
    return run(["gh", *args], cwd=cwd)


def list_ready_issues(limit: int = 5) -> list[dict]:
    proc = run(
        [
            "gh",
            "issue",
            "list",
            "--label",
            LABEL_READY,
            "--state",
            "open",
            "--limit",
            str(limit),
            "--json",
            "number,title,body,labels",
        ]
    )
    if proc.returncode != 0:
        return []
    try:
        return json.loads(proc.stdout or "[]")
    except json.JSONDecodeError:
        return []


def open_agent_prs() -> list[dict]:
    proc = run(
        [
            "gh",
            "pr",
            "list",
            "--label",
            LABEL_PR,
            "--state",
            "open",
            "--json",
            "number,title,url",
        ]
    )
    if proc.returncode != 0:
        return []
    try:
        return json.loads(proc.stdout or "[]")
    except json.JSONDecodeError:
        return []


def slugify(text: str, max_len: int = 40) -> str:
    s = re.sub(r"[^a-zA-Z0-9]+", "-", text.strip().lower()).strip("-")
    return (s or "issue")[:max_len]


def write_brief(issue: dict, path: Path) -> None:
    agents = ROOT / "AGENTS.md"
    agents_txt = agents.read_text(encoding="utf-8") if agents.is_file() else "(AGENTS.md missing)"
    body = f"""# Away agent brief — issue #{issue['number']}

## Issue
**Title:** {issue.get('title', '')}

{issue.get('body') or '(no body)'}

## Rules
{agents_txt}

## Required verification
1. `make test-all`
2. If Go touched: `make test-race`
3. Do not weaken tests, skip scenarios, or commit secrets.
4. Prefer a small PR that closes #{issue['number']}.

Summarize what you changed and which commands you ran.
"""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(body, encoding="utf-8")


def claim_issue(number: int) -> None:
    gh(["issue", "edit", str(number), "--add-label", LABEL_WORKING, "--remove-label", LABEL_READY])
    gh(
        [
            "issue",
            "comment",
            str(number),
            "--body",
            "Claimed by `away-agent-loop` (overnight box).",
        ]
    )


def block_issue(number: int, reason: str) -> None:
    gh(["issue", "edit", str(number), "--add-label", LABEL_BLOCKED, "--remove-label", LABEL_WORKING])
    gh(["issue", "edit", str(number), "--remove-label", LABEL_READY])
    gh(["issue", "comment", str(number), "--body", f"Marked `{LABEL_BLOCKED}`:\n\n{reason}"])


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--max-iter", type=int, default=2)
    p.add_argument("--model", default=os.environ.get("MODEL") or "")
    p.add_argument("--prefer-sdk", action="store_true")
    p.add_argument("--agent-timeout", type=int, default=DEFAULT_AGENT_TIMEOUT_S)
    p.add_argument("--no-worktree", action="store_true")
    p.add_argument("--skip-agent", action="store_true")
    args = p.parse_args()

    apply_release_prep_env(ROOT)

    open_prs = open_agent_prs()
    if open_prs:
        print(f"Open {LABEL_PR} PR already in flight: {open_prs[0].get('url')}")
        print("Serialize: skipping new issue claim.")
        return 0

    issues = list_ready_issues()
    if not issues:
        print(f"No open issues with label {LABEL_READY}")
        return 0

    issue = issues[0]
    number = int(issue["number"])
    stamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M")
    branch = f"agent/issue-{number}-{slugify(issue.get('title') or '')}"
    print(f"Claiming issue #{number}: {issue.get('title')}")

    if args.dry_run:
        print(f"DRY_RUN would claim #{number} on branch {branch}")
        return 0

    claim_issue(number)

    rc, cwd, fix_branch = prepare_fix_loop_cwd(
        ROOT,
        branch=branch,
        base_branch="main",
        use_worktree=not args.no_worktree,
        no_commit=False,
        dry_run=False,
    )
    if rc != 0:
        block_issue(number, f"Failed to prepare branch/worktree (rc={rc}).")
        return rc

    brief = ROOT / "docs" / "testing" / f"away-agent-issue-{number}-{stamp}.md"
    write_brief(issue, brief)

    if not args.skip_agent:
        for i in range(1, args.max_iter + 1):
            prompt = (
                f"Neural Junkie away-agent issue loop (iteration {i}/{args.max_iter}).\n\n"
                f"Read `{brief.relative_to(ROOT)}` and implement the issue. "
                "Run verification commands listed there."
            )
            agent_rc, _ = run_fix_agent(
                prompt,
                cwd=cwd,
                model=args.model or None,
                timeout_s=args.agent_timeout,
                prefer_sdk=args.prefer_sdk,
                log_path=ROOT / "docs" / "testing" / f"away-agent-issue-{number}-{stamp}-iter{i}.log",
            )
            print(f"Agent exit: {agent_rc}")
            test_rc = run(["make", "test-all"], cwd=cwd).returncode
            if test_rc == 0:
                break
            if i == args.max_iter:
                block_issue(number, f"`make test-all` still failing after {args.max_iter} agent iterations.")
                return test_rc
    else:
        test_rc = run(["make", "test-all"], cwd=cwd).returncode
        if test_rc != 0:
            block_issue(number, "`make test-all` failed (skip-agent).")
            return test_rc

    candidates = list_commit_candidates(cwd)
    if not candidates:
        block_issue(number, "Agent produced no commit candidates.")
        return 1

    crc, _ = commit_iteration_changes(
        cwd,
        branch=fix_branch,
        iteration=1,
        summary_path=brief,
        dry_run=False,
    )
    if crc != 0:
        block_issue(number, f"git commit failed (rc={crc}).")
        return crc
    run(["git", "push", "-u", "origin", "HEAD"], cwd=cwd, check=True)

    pr = gh(
        [
            "pr",
            "create",
            "--title",
            f"away: #{number} {issue.get('title', '')[:72]}",
            "--body",
            f"Closes #{number}\n\nAutomated by `away-agent-loop`.\n\nBrief: `{brief.relative_to(ROOT)}`",
            "--label",
            LABEL_PR,
            "--base",
            "main",
            "--head",
            fix_branch,
        ],
        cwd=cwd,
    )
    if pr.returncode != 0:
        # head may already be current branch name
        pr = gh(
            [
                "pr",
                "create",
                "--title",
                f"away: #{number} {issue.get('title', '')[:72]}",
                "--body",
                f"Closes #{number}\n\nAutomated by `away-agent-loop`.",
                "--label",
                LABEL_PR,
                "--base",
                "main",
            ],
            cwd=cwd,
        )
    if pr.returncode != 0:
        block_issue(number, f"gh pr create failed:\n```\n{pr.stderr or pr.stdout}\n```")
        return pr.returncode

    gh(["pr", "merge", "--auto", "--squash"], cwd=cwd)
    gh(["issue", "edit", str(number), "--remove-label", LABEL_WORKING])
    gh(
        [
            "issue",
            "comment",
            str(number),
            "--body",
            "Opened `agent-pr` and enabled auto-merge (squash) pending green unit CI.",
        ]
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
