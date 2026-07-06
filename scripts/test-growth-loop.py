#!/usr/bin/env python3
"""Test-growth loop: discover gaps → Cursor agent adds/strengthens tests → verify → commit → repeat."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
DEFAULT_TESTING_DIR = ROOT / "docs" / "testing"
PY = sys.executable

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.cursor_fix_agent import (  # noqa: E402
    DEFAULT_AGENT_TIMEOUT_S,
    RECOVERABLE_AGENT_EXITS,
    agent_binary,
    agent_exit_label,
    run_fix_agent,
)
from lib.fix_loop_git import (  # noqa: E402
    commit_test_growth_changes,
    default_test_growth_branch,
    list_commit_candidates,
    prepare_fix_loop_cwd,
)
from lib.release_prep_env import apply_release_prep_env, provision_hub_automation_key, release_prep_env  # noqa: E402
from lib.regression_boot import maybe_boot_regression  # noqa: E402
from lib.test_growth_candidates import (  # noqa: E402
    discover_candidates,
    format_candidate_list,
    pick_candidate,
)
from lib.test_growth_prompt import build_agent_invoke_message, build_test_growth_prompt  # noqa: E402
from lib.test_growth_verification import verify_iteration  # noqa: E402

LOOP_LOG_PREFIX = "test-growth"


def run_cmd(cmd: list[str], *, cwd: Path, env: dict | None = None) -> tuple[int, str]:
    merged = release_prep_env(ROOT)
    if env:
        merged.update(env)
    merged["PYTHONUNBUFFERED"] = "1"
    print(f"\n>>> {' '.join(cmd)}", flush=True)
    proc = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, env=merged)
    out = (proc.stdout or "") + (proc.stderr or "")
    if out:
        sys.stdout.write(out)
        if not out.endswith("\n"):
            sys.stdout.write("\n")
        sys.stdout.flush()
    return proc.returncode, out


def write_iteration_log(
    path: Path,
    *,
    iteration: int,
    stamp: str,
    candidate_id: str,
    candidate_title: str,
    agent_rc: int | None,
    agent_out: str,
    verify_result,
    git_commit: str = "",
    growth_branch: str = "",
    accepted: bool = False,
    repair_handoff: bool = False,
) -> None:
    lines = [
        f"# test growth loop — iteration {iteration} — {stamp} UTC",
        "",
        f"candidate_id={candidate_id}",
        f"candidate_title={candidate_title}",
        f"growth_branch={growth_branch or '(none)'}",
        f"git_commit={git_commit or '(none)'}",
        f"accepted={accepted}",
        f"repair_handoff={repair_handoff}",
        f"agent_rc={agent_rc}",
        "",
        "## Verification checks",
        "",
    ]
    for name, ok, detail in verify_result.checks:
        status = "OK" if ok else "FAIL"
        lines.append(f"- [{status}] {name}: {detail[:500]}")
    lines.append("")
    if verify_result.verify_runs:
        lines.append("## Commands run")
        lines.append("")
        for cmd, rc in verify_result.verify_runs:
            status = "OK" if rc == 0 else "FAIL"
            lines.append(f"- [{status}] {' '.join(cmd)} (exit {rc})")
        lines.append("")
    if verify_result.rejection_reason:
        lines.extend(["## Rejection", "", verify_result.rejection_reason, ""])
    if agent_out.strip():
        lines.extend(["## Cursor agent output", "", "```text", agent_out.strip()[-32000:], "```", ""])
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--max-iterations", type=int, default=3)
    p.add_argument("--candidate-limit", type=int, default=50)
    p.add_argument("--candidate-kind", help="Filter candidates: unit|scenario|layer_a|failure_repro")
    p.add_argument("--skip-agent", action="store_true")
    p.add_argument("--skip-verify", action="store_true")
    p.add_argument("--skip-live", action="store_true", help="Skip live scenario reruns during verify")
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--model", help="Cursor agent model")
    p.add_argument("--prefer-sdk", action="store_true")
    p.add_argument("--agent-timeout", type=int, default=int(os.environ.get("NJ_AGENT_TIMEOUT", str(DEFAULT_AGENT_TIMEOUT_S))))
    p.add_argument("--cwd", type=Path, default=ROOT)
    p.add_argument("--no-commit", action="store_true")
    p.add_argument("--growth-branch", help="Git branch for test-growth commits")
    p.add_argument("--base-branch", help="Base ref when creating growth branch")
    p.add_argument("--use-worktree", action="store_true", default=True)
    p.add_argument("--no-worktree", action="store_false", dest="use_worktree")
    p.add_argument("--stability-runs", type=int, default=2, help="Live scenario reruns before accept")
    p.add_argument("--list", action="store_true", help="List candidates and exit")
    p.add_argument("--once", action="store_true", help="Single iteration only")
    args = p.parse_args()

    if args.once:
        args.max_iterations = 1

    apply_release_prep_env(ROOT)
    provision_hub_automation_key(ROOT)
    testing_dir = Path(args.log_dir)
    testing_dir.mkdir(parents=True, exist_ok=True)
    hub_url = args.hub.rstrip("/")
    repo_root = args.cwd.resolve()

    candidates = discover_candidates(limit=args.candidate_limit)
    if args.list:
        print(format_candidate_list(candidates))
        list_path = testing_dir / f"{LOOP_LOG_PREFIX}-candidates.md"
        list_path.write_text(format_candidate_list(candidates), encoding="utf-8")
        print(f"Wrote: {list_path}")
        return 0

    if args.max_iterations < 1:
        print("max-iterations must be >= 1", file=sys.stderr)
        return 1

    if args.dry_run:
        args.skip_agent = True
        args.skip_verify = True

    loop_stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    growth_branch = args.growth_branch or default_test_growth_branch(loop_stamp)
    loop_cwd = repo_root

    if not args.no_commit and not args.dry_run:
        git_rc, loop_cwd, growth_branch = prepare_fix_loop_cwd(
            repo_root,
            branch=growth_branch,
            base_branch=args.base_branch,
            use_worktree=args.use_worktree,
            no_commit=args.no_commit,
            dry_run=args.dry_run,
        )
        if git_rc != 0:
            print("Git branch/worktree setup failed; use --no-commit to skip.", file=sys.stderr)
            return git_rc
    elif args.use_worktree:
        print(">>> [git] worktree skipped (--no-commit or --dry-run); using repo checkout", flush=True)

    if not args.skip_live and not args.dry_run:
        print("\n>>> Boot regression stack for live scenario verification")
        if not maybe_boot_regression(hub_url, root=loop_cwd, label="test-growth-loop"):
            print("WARN: regression boot failed — live scenario verify may fail", file=sys.stderr)

    skip_ids: set[str] = set()
    accepted_count = 0

    for iteration in range(1, args.max_iterations + 1):
        iter_stamp = f"{loop_stamp}-iter{iteration}"
        print(f"\n{'=' * 72}\n=== Test growth loop — iteration {iteration}/{args.max_iterations} ===\n{'=' * 72}")

        candidate = pick_candidate(candidates, skip_ids=skip_ids, kind=args.candidate_kind)
        if candidate is None:
            print("No more candidates — refreshing discovery")
            candidates = discover_candidates(limit=args.candidate_limit)
            candidate = pick_candidate(candidates, skip_ids=skip_ids, kind=args.candidate_kind)
        if candidate is None:
            print("No candidates available — loop complete")
            break

        print(f"\n=== Selected candidate ===")
        print(f"  id:    {candidate.id}")
        print(f"  kind:  {candidate.kind}")
        print(f"  title: {candidate.title}")
        print(f"  score: {candidate.score:.0f}")

        prompt = build_test_growth_prompt(candidate, iteration=iteration, max_iterations=args.max_iterations)
        prompt_path = testing_dir / f"{LOOP_LOG_PREFIX}-prompt-{iter_stamp}.md"
        prompt_path.write_text(prompt + "\n", encoding="utf-8")
        invoke_msg = build_agent_invoke_message(
            str(prompt_path.relative_to(loop_cwd) if prompt_path.is_relative_to(loop_cwd) else prompt_path),
            repo_root=str(loop_cwd),
        )
        print(f"\nAgent prompt: {prompt_path} ({len(prompt)} chars)")

        agent_rc: int | None = None
        agent_out = ""
        if args.skip_agent:
            print("\n--- Agent prompt preview (first 2000 chars) ---")
            print(prompt[:2000])
        else:
            if not agent_binary() and not args.prefer_sdk:
                print("Cursor CLI 'agent' not on PATH.", file=sys.stderr)
                return 127
            agent_log = testing_dir / f"{LOOP_LOG_PREFIX}-agent-{iter_stamp}.log"
            agent_rc, agent_out = run_fix_agent(
                invoke_msg,
                cwd=loop_cwd,
                model=args.model,
                prefer_sdk=args.prefer_sdk,
                timeout_s=args.agent_timeout,
                log_path=agent_log,
            )
            if agent_rc in RECOVERABLE_AGENT_EXITS:
                pending = list_commit_candidates(loop_cwd)
                label = agent_exit_label(agent_rc)
                if pending:
                    print(f">>> Agent {label} but {len(pending)} file(s) changed — continuing with verify.")
                else:
                    print(f">>> Agent {label} with no repo changes — skipping iteration.", file=sys.stderr)
                    skip_ids.add(candidate.id)
                    time.sleep(3)
                    continue

        verify_result = None
        accepted = False
        repair_handoff = False
        git_commit = ""

        if not args.skip_verify and not args.skip_agent:
            verify_result = verify_iteration(
                loop_cwd,
                candidate=candidate,
                hub_url=hub_url,
                stability_runs=args.stability_runs,
                skip_live=args.skip_live,
            )
            accepted = verify_result.accepted
            repair_handoff = verify_result.repair_handoff

            if repair_handoff:
                print("\n>>> REPAIR HANDOFF: new test exposed product defect")
                print("    Run: make layer-fix-loop LAYER=<layer> to fix product code")
                skip_ids.add(candidate.id)
            elif accepted:
                print("\n>>> ACCEPTED: meaningful test improvement verified")
                if not args.no_commit and not args.dry_run:
                    commit_rc, _ = commit_test_growth_changes(
                        loop_cwd,
                        branch=growth_branch,
                        iteration=iteration,
                        candidate_id=candidate.id,
                        dry_run=args.dry_run,
                    )
                    if commit_rc == 0:
                        proc = subprocess.run(
                            ["git", "rev-parse", "--short", "HEAD"],
                            cwd=loop_cwd,
                            capture_output=True,
                            text=True,
                        )
                        git_commit = (proc.stdout or "").strip()
                accepted_count += 1
            else:
                print(f"\n>>> REJECTED: {verify_result.rejection_reason}")
                skip_ids.add(candidate.id)
        elif args.skip_verify:
            from lib.test_growth_verification import VerifyResult

            verify_result = VerifyResult(accepted=False, rejection_reason="verify skipped")

        log_path = testing_dir / f"{LOOP_LOG_PREFIX}-{iter_stamp}.md"
        if verify_result is not None:
            write_iteration_log(
                log_path,
                iteration=iteration,
                stamp=iter_stamp,
                candidate_id=candidate.id,
                candidate_title=candidate.title,
                agent_rc=agent_rc,
                agent_out=agent_out,
                verify_result=verify_result,
                git_commit=git_commit,
                growth_branch=growth_branch,
                accepted=accepted,
                repair_handoff=repair_handoff,
            )
        print(f"Iteration log: {log_path}")

        if not accepted and not repair_handoff and not args.skip_verify:
            time.sleep(3)

    print(f"\n=== Test growth loop complete — {accepted_count} accepted iteration(s) ===")
    print(f"Branch: {growth_branch}")
    print(f"Worktree/cwd: {loop_cwd}")
    return 0 if accepted_count > 0 or args.dry_run or args.skip_agent else 1


if __name__ == "__main__":
    raise SystemExit(main())
