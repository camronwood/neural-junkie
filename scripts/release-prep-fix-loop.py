#!/usr/bin/env python3
"""Release-prep fix loop: run gate → Cursor agent fixes → commit → targeted rerun → repeat."""

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
    build_agent_invoke_message,
    run_fix_agent,
)
from lib.fix_loop_git import commit_iteration_changes, ensure_fix_branch, list_commit_candidates  # noqa: E402
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402
from lib.release_prep_failures import (  # noqa: E402
    FailureKind,
    build_agent_prompt,
    parse_release_prep_report,
)

LOOP_LOG_PREFIX = "release-prep-fix-loop"


def run_cmd(cmd: list[str], *, env: dict | None = None, cwd: Path = ROOT) -> tuple[int, str]:
    merged = release_prep_env(ROOT)
    if env:
        merged.update(env)
    merged["PYTHONUNBUFFERED"] = "1"
    print(f"\n>>> {' '.join(cmd)}", flush=True)
    proc = subprocess.run(
        cmd,
        cwd=cwd,
        capture_output=True,
        text=True,
        env=merged,
    )
    out = (proc.stdout or "") + (proc.stderr or "")
    if out:
        sys.stdout.write(out)
        if not out.endswith("\n"):
            sys.stdout.write("\n")
        sys.stdout.flush()
    return proc.returncode, out


def run_release_prep(
    *,
    hub_url: str,
    testing_dir: Path,
    stamp: str,
    skip_benchmark: bool,
    no_full: bool,
    skip_parity: bool,
    skip_live: bool,
    verbose: bool,
) -> tuple[int, Path]:
    cmd = [
        PY,
        str(SCRIPTS_DIR / "release-prep.py"),
        "--hub",
        hub_url,
        "--stamp",
        stamp,
        "--log-dir",
        str(testing_dir),
    ]
    if skip_benchmark:
        cmd.append("--skip-benchmark")
    if no_full:
        cmd.append("--no-full")
    if skip_parity:
        cmd.append("--skip-parity")
    if skip_live:
        cmd.append("--skip-live")
    if verbose:
        cmd.append("--verbose")
    rc, _ = run_cmd(cmd)
    summary = testing_dir / f"release-prep-{stamp}.md"
    return rc, summary


def run_targeted_verification(cmds: list[list[str]], *, hub_url: str) -> tuple[bool, list[tuple[list[str], int]]]:
    env = {"NEURAL_JUNKIE_HUB_URL": hub_url}
    results: list[tuple[list[str], int]] = []
    all_ok = True
    for cmd in cmds:
        rc, _ = run_cmd(cmd, env=env)
        results.append((cmd, rc))
        if rc != 0:
            all_ok = False
    return all_ok, results


def write_iteration_log(
    path: Path,
    *,
    iteration: int,
    stamp: str,
    summary_path: Path,
    agent_rc: int | None,
    agent_out: str,
    verify_results: list[tuple[list[str], int]],
    release_prep_rc: int | None,
    git_commit: str = "",
    fix_branch: str = "",
) -> None:
    lines = [
        f"# release-prep fix loop — iteration {iteration} — {stamp} UTC",
        "",
        f"summary={summary_path}",
        f"fix_branch={fix_branch or '(none)'}",
        f"git_commit={git_commit or '(none)'}",
        f"release_prep_rc={release_prep_rc}",
        f"agent_rc={agent_rc}",
        "",
    ]
    if verify_results:
        lines.append("## Targeted verification")
        for cmd, rc in verify_results:
            status = "OK" if rc == 0 else "FAIL"
            lines.append(f"- [{status}] {' '.join(cmd)} (exit {rc})")
        lines.append("")
    if agent_out.strip():
        lines.extend(["## Cursor agent output", "", "```text", agent_out.strip()[-32000:], "```", ""])
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def print_report_summary(summary_path: Path, hub_url: str) -> None:
    report = parse_release_prep_report(summary_path, hub_url=hub_url)
    print("\n=== Parsed failures ===")
    print(f"Summary: {summary_path}")
    print(f"Phases:  {', '.join(report.failed_phases) or '(none)'}")
    print(f"Items:   {len(report.failures)}")
    for f in report.failures:
        rerun = " ".join(f.rerun_cmd) if f.rerun_cmd else "—"
        print(f"  - {f.name} [{f.kind.value}] → {rerun}")
    print(f"Agent candidates: {len(report.agent_candidates)}")
    print(f"Retry-only:       {len(report.retry_only)}")


def _handle_agent_interrupted(
    *,
    agent_rc: int,
    agent_timeout: int,
    cwd: Path,
    iteration: int,
    max_iterations: int,
) -> str:
    """Return 'continue' | 'retry' | 'abort'."""
    pending = list_commit_candidates(cwd)
    label = agent_exit_label(agent_rc)
    if pending:
        print(
            f">>> Agent {label} but {len(pending)} file(s) changed — continuing with verify.",
            flush=True,
        )
        return "continue"
    print(f">>> Agent {label} with no repo changes.", file=sys.stderr)
    if agent_rc == 124:
        print(f"    (limit was {agent_timeout}s — set AGENT_TIMEOUT=14400 or higher)", file=sys.stderr)
    if agent_rc == 143:
        print(
            "    Common causes: Mac sleep, closed terminal, tmux session killed, or Ctrl+C.",
            file=sys.stderr,
        )
    if iteration >= max_iterations:
        return "abort"
    return "retry"


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--max-iterations", type=int, default=3)
    p.add_argument("--report", type=Path, help="Parse existing release-prep-*.md instead of running gate first")
    p.add_argument("--skip-release-prep", action="store_true", help="Only parse/fix/rerun (requires --report)")
    p.add_argument("--skip-agent", action="store_true", help="Parse and print prompt; do not invoke Cursor")
    p.add_argument("--skip-verify", action="store_true", help="Skip targeted verification after agent")
    p.add_argument("--dry-run", action="store_true", help="Print plan only; no agent, no reruns")
    p.add_argument("--skip-benchmark", action="store_true", help="Pass through to release-prep (faster loops)")
    p.add_argument("--no-full", action="store_true", help="Skip collab-scenarios-all in release-prep")
    p.add_argument("--skip-parity", action="store_true", help="Skip test-parity-stable-restart in release-prep")
    p.add_argument("--skip-live", action="store_true", help="Skip live harness phases in release-prep")
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--model", help="Cursor agent model (CLI --model)")
    p.add_argument("--prefer-sdk", action="store_true", help="Use cursor_sdk when installed")
    p.add_argument(
        "--agent-timeout",
        type=int,
        default=int(os.environ.get("NJ_AGENT_TIMEOUT", str(DEFAULT_AGENT_TIMEOUT_S))),
        help=f"Cursor agent timeout seconds (default {DEFAULT_AGENT_TIMEOUT_S}, env NJ_AGENT_TIMEOUT)",
    )
    p.add_argument("--cwd", type=Path, default=ROOT, help="Repo root for Cursor agent")
    p.add_argument("--no-commit", action="store_true", help="Do not auto-commit agent changes to a fix branch")
    p.add_argument("--fix-branch", help="Git branch for fixes (default: release-prep/fix-<stamp>)")
    p.add_argument("--base-branch", help="Base ref when creating a new fix branch")
    args = p.parse_args()

    if args.max_iterations < 1:
        print("max-iterations must be >= 1", file=sys.stderr)
        return 1

    apply_release_prep_env(ROOT)
    from lib.release_prep_env import provision_hub_automation_key

    provision_hub_automation_key(ROOT)
    testing_dir = Path(args.log_dir)
    testing_dir.mkdir(parents=True, exist_ok=True)
    hub_url = args.hub.rstrip("/")

    if args.dry_run:
        args.skip_agent = True
        args.skip_verify = True

    summary_path = args.report
    loop_stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    fix_branch = args.fix_branch or f"release-prep/fix-{loop_stamp}"

    if not args.no_commit and not args.dry_run:
        git_rc, fix_branch = ensure_fix_branch(
            args.cwd.resolve(),
            branch=fix_branch,
            base_branch=args.base_branch,
        )
        if git_rc != 0:
            print("Git branch setup failed; use --no-commit to skip.", file=sys.stderr)
            return git_rc

    for iteration in range(1, args.max_iterations + 1):
        iter_stamp = f"{loop_stamp}-iter{iteration}"
        print(f"\n{'=' * 72}\n=== Fix loop iteration {iteration}/{args.max_iterations} ===\n{'=' * 72}")

        release_prep_rc: int | None = None
        if summary_path is None and not args.skip_release_prep:
            rp_stamp = iter_stamp
            release_prep_rc, summary_path = run_release_prep(
                hub_url=hub_url,
                testing_dir=testing_dir,
                stamp=rp_stamp,
                skip_benchmark=args.skip_benchmark,
                no_full=args.no_full,
                skip_parity=args.skip_parity,
                skip_live=args.skip_live,
                verbose=args.verbose,
            )
            if release_prep_rc == 0:
                print("\n=== Release prep PASS — fix loop complete ===")
                print(f"Review: {summary_path}")
                return 0
        elif summary_path is None:
            print("Need --report or allow release-prep run", file=sys.stderr)
            return 1

        if not summary_path.is_file():
            print(f"Missing summary: {summary_path}", file=sys.stderr)
            return 1

        report = parse_release_prep_report(summary_path, hub_url=hub_url)
        print_report_summary(summary_path, hub_url)

        if not report.overall_fail and release_prep_rc == 0:
            print("\n=== Report shows PASS — fix loop complete ===")
            return 0

        code_fixes = [f for f in report.agent_candidates if f.kind == FailureKind.CODE]
        if not code_fixes and report.retry_only and not args.skip_agent:
            print("\n>>> No code-fix candidates; only flakes/infra/model issues detected.")
            if report.rerun_cmds and not args.skip_verify:
                print(">>> Retrying targeted commands once...")
                ok, verify_results = run_targeted_verification(report.rerun_cmds, hub_url=hub_url)
                log_path = testing_dir / f"{LOOP_LOG_PREFIX}-{iter_stamp}.md"
                write_iteration_log(
                    log_path,
                    iteration=iteration,
                    stamp=iter_stamp,
                    summary_path=summary_path,
                    agent_rc=None,
                    agent_out="(skipped — retry-only failures)",
                    verify_results=verify_results,
                    release_prep_rc=release_prep_rc,
                    fix_branch=fix_branch,
                )
                if ok:
                    summary_path = None
                    continue
            if iteration >= args.max_iterations:
                print("Max iterations reached with retry-only failures.", file=sys.stderr)
                return 1
            summary_path = None
            time.sleep(5)
            continue

        prompt = build_agent_prompt(report)
        prompt_path = testing_dir / f"{LOOP_LOG_PREFIX}-prompt-{iter_stamp}.md"
        prompt_path.write_text(prompt + "\n", encoding="utf-8")
        invoke_msg = build_agent_invoke_message(prompt_path, repo_root=args.cwd.resolve())
        print(
            f"\nAgent prompt: {prompt_path} "
            f"({len(prompt)} chars brief, {len(invoke_msg)} chars argv)"
        )

        agent_rc: int | None = None
        agent_out = ""
        if args.skip_agent:
            print("\n--- Agent prompt preview (first 2000 chars) ---")
            print(prompt[:2000])
            if len(prompt) > 2000:
                print(f"\n... ({len(prompt) - 2000} more chars in {prompt_path})")
        else:
            if not agent_binary() and not args.prefer_sdk:
                print(
                    "Cursor CLI 'agent' not on PATH. Install or use --skip-agent to inspect prompt.",
                    file=sys.stderr,
                )
                return 127
            agent_log = testing_dir / f"{LOOP_LOG_PREFIX}-agent-{iter_stamp}.log"
            agent_rc, agent_out = run_fix_agent(
                invoke_msg,
                cwd=args.cwd.resolve(),
                model=args.model,
                prefer_sdk=args.prefer_sdk,
                timeout_s=args.agent_timeout,
                log_path=agent_log,
            )
            if not agent_log.is_file() or agent_log.stat().st_size == 0:
                agent_log.write_text(agent_out, encoding="utf-8")
            print(f"Agent log: {agent_log} (exit {agent_rc})")
            if agent_rc in RECOVERABLE_AGENT_EXITS:
                action = _handle_agent_interrupted(
                    agent_rc=agent_rc,
                    agent_timeout=args.agent_timeout,
                    cwd=args.cwd.resolve(),
                    iteration=iteration,
                    max_iterations=args.max_iterations,
                )
                if action == "abort":
                    return agent_rc
                if action == "retry":
                    summary_path = None
                    time.sleep(5)
                    continue
            elif agent_rc not in (0, 2):
                print("Cursor agent failed to run; stopping loop.", file=sys.stderr)
                return agent_rc if agent_rc is not None else 1

        git_commit = ""
        if not args.skip_agent and not args.no_commit and not args.dry_run:
            commit_rc, _ = commit_iteration_changes(
                args.cwd.resolve(),
                branch=fix_branch,
                iteration=iteration,
                summary_path=summary_path,
            )
            if commit_rc != 0:
                print("Git commit failed; continuing with verification.", file=sys.stderr)
            else:
                sha_proc = subprocess.run(
                    ["git", "rev-parse", "--short", "HEAD"],
                    cwd=args.cwd.resolve(),
                    capture_output=True,
                    text=True,
                )
                git_commit = (sha_proc.stdout or "").strip()

        verify_results: list[tuple[list[str], int]] = []
        verify_ok = True
        if not args.skip_verify and report.rerun_cmds:
            verify_ok, verify_results = run_targeted_verification(report.rerun_cmds, hub_url=hub_url)

        log_path = testing_dir / f"{LOOP_LOG_PREFIX}-{iter_stamp}.md"
        write_iteration_log(
            log_path,
            iteration=iteration,
            stamp=iter_stamp,
            summary_path=summary_path,
            agent_rc=agent_rc,
            agent_out=agent_out,
            verify_results=verify_results,
            release_prep_rc=release_prep_rc,
            git_commit=git_commit,
            fix_branch=fix_branch,
        )
        print(f"Iteration log: {log_path}")

        if args.dry_run or args.skip_agent:
            return 0

        if verify_ok and iteration < args.max_iterations:
            print("\n>>> Targeted verification PASS — running full release-prep confirm...")
            summary_path = None
            continue

        if verify_ok and iteration == args.max_iterations:
            print("\n>>> Targeted verification PASS on final iteration.")
            return 0

        if iteration >= args.max_iterations:
            print("\n=== Max iterations reached; still failing ===", file=sys.stderr)
            return 1

        summary_path = None
        time.sleep(5)

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
