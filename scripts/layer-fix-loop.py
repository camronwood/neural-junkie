#!/usr/bin/env python3
"""Layer fix loop: run one layer gate → Cursor agent fixes → commit → targeted rerun → repeat."""

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
from lib.fix_loop_git import commit_iteration_changes, prepare_fix_loop_cwd, list_commit_candidates  # noqa: E402
from lib.release_prep_env import apply_release_prep_env, provision_hub_automation_key  # noqa: E402
from lib.regression_boot import restart_hub_for_live_run  # noqa: E402
from lib.release_prep_failures import FailureKind  # noqa: E402
from lib.release_prep_layers import (  # noqa: E402
    LAYER_ORDER,
    build_layer_agent_prompt,
    get_layer,
    parse_layer_gate_report,
)

LOOP_LOG_PREFIX = "layer-fix-loop"


def run_cmd(cmd: list[str], *, env: dict | None = None, cwd: Path = ROOT) -> tuple[int, str]:
    from lib.release_prep_env import release_prep_env

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


def run_layer_gate(
    *,
    layer: str,
    hub_url: str,
    testing_dir: Path,
    stamp: str,
    no_restart_hub: bool,
    verbose: bool,
    cwd: Path,
) -> tuple[int, Path]:
    cmd = [
        PY,
        str(SCRIPTS_DIR / "layer-gate.py"),
        "--layer",
        layer,
        "--hub",
        hub_url,
        "--stamp",
        stamp,
        "--log-dir",
        str(testing_dir),
    ]
    if no_restart_hub:
        cmd.append("--no-restart-hub")
    if verbose:
        cmd.append("--verbose")
    rc, _ = run_cmd(cmd, cwd=cwd)
    summary = testing_dir / f"layer-gate-{layer}-{stamp}.md"
    return rc, summary


def prepare_next_iteration_hub(hub_url: str, *, layer: str, next_iter: int, cwd: Path, no_restart_hub: bool) -> bool:
    if no_restart_hub:
        return True
    return restart_hub_for_live_run(cwd.resolve(), hub_url, label=f"layer-fix-loop-{layer}-iter{next_iter}")


def run_targeted_verification(
    cmds: list[list[str]], *, hub_url: str, cwd: Path, max_scenarios: int = 0
) -> tuple[bool, list[tuple[list[str], int]]]:
    env = {"NEURAL_JUNKIE_HUB_URL": hub_url}
    results: list[tuple[list[str], int]] = []
    all_ok = True
    to_run = cmds
    if max_scenarios > 0 and len(cmds) > max_scenarios:
        to_run = cmds[:max_scenarios]
        print(
            f"\n>>> Targeted verify capped to {max_scenarios}/{len(cmds)} scenarios "
            f"(set MAX_VERIFY_SCENARIOS=0 to rerun all)",
            flush=True,
        )
    for cmd in to_run:
        rc, _ = run_cmd(cmd, env=env, cwd=cwd)
        results.append((cmd, rc))
        if rc != 0:
            all_ok = False
    return all_ok, results


def effective_max_verify_scenarios(*, layer: str, agent_rc: int | None, configured: int) -> int:
    """Cap post-agent reruns so a stalled agent does not replay the full gate."""
    if configured == 0:
        return 0
    if configured > 0:
        return configured
    if layer == "collab-full":
        return 5
    if agent_rc is not None and agent_rc != 0:
        return 3
    return 0


def write_iteration_log(
    path: Path,
    *,
    layer: str,
    iteration: int,
    stamp: str,
    summary_path: Path,
    gate_rc: int | None,
    agent_rc: int | None,
    agent_out: str,
    verify_results: list[tuple[list[str], int]],
    git_commit: str = "",
    fix_branch: str = "",
) -> None:
    lines = [
        f"# layer fix loop — {layer} — iteration {iteration} — {stamp} UTC",
        "",
        f"layer={layer}",
        f"summary={summary_path}",
        f"fix_branch={fix_branch or '(none)'}",
        f"git_commit={git_commit or '(none)'}",
        f"layer_gate_rc={gate_rc}",
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


def _handle_agent_interrupted(
    *,
    agent_rc: int,
    agent_timeout: int,
    cwd: Path,
    iteration: int,
    max_iterations: int,
) -> str:
    pending = list_commit_candidates(cwd)
    label = agent_exit_label(agent_rc)
    if pending:
        print(f">>> Agent {label} but {len(pending)} file(s) changed — continuing with verify.", flush=True)
        return "continue"
    print(f">>> Agent {label} with no repo changes.", file=sys.stderr)
    if agent_rc == 124:
        print(f"    (limit was {agent_timeout}s)", file=sys.stderr)
    if iteration >= max_iterations:
        return "abort"
    return "retry"


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--layer", required=True, help=f"Layer: {', '.join(LAYER_ORDER)}")
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--max-iterations", type=int, default=3)
    p.add_argument("--report", type=Path, help="Parse existing layer-gate-*.md instead of running gate")
    p.add_argument("--skip-gate", action="store_true", help="Only parse/fix/rerun (requires --report)")
    p.add_argument("--skip-agent", action="store_true")
    p.add_argument("--skip-verify", action="store_true")
    p.add_argument(
        "--max-verify-scenarios",
        type=int,
        default=int(os.environ.get("MAX_VERIFY_SCENARIOS", "-1")),
        help="Cap targeted reruns after agent (0=unlimited; default 5 for collab-full)",
    )
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--no-restart-hub", action="store_true")
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--model", help="Cursor agent model")
    p.add_argument("--prefer-sdk", action="store_true")
    p.add_argument("--agent-timeout", type=int, default=int(os.environ.get("NJ_AGENT_TIMEOUT", str(DEFAULT_AGENT_TIMEOUT_S))))
    p.add_argument("--cwd", type=Path, default=ROOT, help="Repo root (worktree created here when --use-worktree)")
    p.add_argument("--no-commit", action="store_true")
    p.add_argument("--fix-branch", help="Git branch for fixes")
    p.add_argument("--base-branch", help="Base ref when creating fix branch")
    p.add_argument("--use-worktree", action="store_true", default=True, help="Run fixes in .worktrees/<branch> (default)")
    p.add_argument("--no-worktree", action="store_false", dest="use_worktree", help="Checkout fix branch in --cwd instead")
    p.add_argument("--list", action="store_true")
    args = p.parse_args()

    if args.list:
        from lib.release_prep_layers import list_layers

        for spec in list_layers():
            print(f"{spec.name}: {spec.description}")
        return 0

    try:
        spec = get_layer(args.layer)
    except ValueError as err:
        print(err, file=sys.stderr)
        return 2

    if args.max_iterations < 1:
        print("max-iterations must be >= 1", file=sys.stderr)
        return 1

    apply_release_prep_env(ROOT)
    provision_hub_automation_key(ROOT)
    testing_dir = Path(args.log_dir)
    testing_dir.mkdir(parents=True, exist_ok=True)
    hub_url = args.hub.rstrip("/")
    layer = spec.name

    if args.dry_run:
        args.skip_agent = True
        args.skip_verify = True

    summary_path = args.report
    loop_stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    repo_root = args.cwd.resolve()
    fix_branch = args.fix_branch or f"release-prep/layer-{layer}-{loop_stamp}"
    loop_cwd = repo_root

    if not args.no_commit and not args.dry_run:
        git_rc, loop_cwd, fix_branch = prepare_fix_loop_cwd(
            repo_root,
            branch=fix_branch,
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

    for iteration in range(1, args.max_iterations + 1):
        iter_stamp = f"{loop_stamp}-iter{iteration}"
        print(f"\n{'=' * 72}\n=== Layer fix loop {layer} — iteration {iteration}/{args.max_iterations} ===\n{'=' * 72}")

        gate_rc: int | None = None
        if summary_path is None and not args.skip_gate:
            gate_rc, summary_path = run_layer_gate(
                layer=layer,
                hub_url=hub_url,
                testing_dir=testing_dir,
                stamp=iter_stamp,
                no_restart_hub=args.no_restart_hub,
                verbose=args.verbose,
                cwd=loop_cwd,
            )
            if gate_rc == 0:
                print(f"\n=== Layer {layer} PASS — fix loop complete ===")
                print(f"Review: {summary_path}")
                if spec.next_layer:
                    print(f"Next: make layer-gate LAYER={spec.next_layer}")
                return 0
        elif summary_path is None:
            print("Need --report or allow layer gate run", file=sys.stderr)
            return 1

        if not summary_path.is_file():
            print(f"Missing summary: {summary_path}", file=sys.stderr)
            return 1

        report = parse_layer_gate_report(summary_path, hub_url=hub_url)
        print(f"\n=== Parsed failures (layer={layer}) ===")
        print(f"Summary: {summary_path}")
        print(f"Items:   {len(report.failures)}")
        for f in report.failures:
            rerun = " ".join(f.rerun_cmd) if f.rerun_cmd else "—"
            print(f"  - {f.name} [{f.kind.value}] → {rerun}")

        if not report.overall_fail and gate_rc == 0:
            print(f"\n=== Layer {layer} PASS — fix loop complete ===")
            return 0

        code_fixes = [f for f in report.agent_candidates if f.kind == FailureKind.CODE]
        if not code_fixes and report.retry_only and not args.skip_agent:
            print("\n>>> No code-fix candidates; flake/infra only.")
            if report.rerun_cmds and not args.skip_verify:
                max_verify = effective_max_verify_scenarios(
                    layer=layer, agent_rc=None, configured=args.max_verify_scenarios
                )
                ok, verify_results = run_targeted_verification(
                    report.rerun_cmds, hub_url=hub_url, cwd=loop_cwd, max_scenarios=max_verify
                )
                log_path = testing_dir / f"{LOOP_LOG_PREFIX}-{layer}-{iter_stamp}.md"
                write_iteration_log(
                    log_path,
                    layer=layer,
                    iteration=iteration,
                    stamp=iter_stamp,
                    summary_path=summary_path,
                    gate_rc=gate_rc,
                    agent_rc=None,
                    agent_out="(skipped — retry-only failures)",
                    verify_results=verify_results,
                    fix_branch=fix_branch,
                )
                if ok:
                    summary_path = None
                    if not prepare_next_iteration_hub(
                        hub_url, layer=layer, next_iter=iteration + 1, cwd=loop_cwd, no_restart_hub=args.no_restart_hub
                    ):
                        return 1
                    continue
            if iteration >= args.max_iterations:
                return 1
            summary_path = None
            if not prepare_next_iteration_hub(
                hub_url, layer=layer, next_iter=iteration + 1, cwd=loop_cwd, no_restart_hub=args.no_restart_hub
            ):
                return 1
            time.sleep(5)
            continue

        prompt = build_layer_agent_prompt(report, layer=layer)
        prompt_path = testing_dir / f"{LOOP_LOG_PREFIX}-prompt-{layer}-{iter_stamp}.md"
        prompt_path.write_text(prompt + "\n", encoding="utf-8")
        invoke_msg = build_agent_invoke_message(prompt_path, repo_root=loop_cwd)
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
            agent_log = testing_dir / f"{LOOP_LOG_PREFIX}-agent-{layer}-{iter_stamp}.log"
            agent_rc, agent_out = run_fix_agent(
                invoke_msg,
                cwd=loop_cwd,
                model=args.model,
                prefer_sdk=args.prefer_sdk,
                timeout_s=args.agent_timeout,
                log_path=agent_log,
            )
            if agent_rc in RECOVERABLE_AGENT_EXITS:
                action = _handle_agent_interrupted(
                    agent_rc=agent_rc,
                    agent_timeout=args.agent_timeout,
                    cwd=loop_cwd,
                    iteration=iteration,
                    max_iterations=args.max_iterations,
                )
                if action == "abort":
                    return agent_rc or 1
                if action == "retry":
                    summary_path = None
                    time.sleep(5)
                    continue

        git_commit = ""
        if not args.skip_agent and not args.no_commit and not args.dry_run:
            commit_rc, _ = commit_iteration_changes(
                loop_cwd,
                branch=fix_branch,
                iteration=iteration,
                summary_path=summary_path,
            )
            if commit_rc == 0:
                proc = subprocess.run(
                    ["git", "rev-parse", "--short", "HEAD"],
                    cwd=loop_cwd,
                    capture_output=True,
                    text=True,
                )
                git_commit = (proc.stdout or "").strip()

        verify_results: list[tuple[list[str], int]] = []
        verify_ok = True
        if not args.skip_verify and report.rerun_cmds:
            max_verify = effective_max_verify_scenarios(
                layer=layer, agent_rc=agent_rc, configured=args.max_verify_scenarios
            )
            if agent_rc in RECOVERABLE_AGENT_EXITS and max_verify > 0:
                print(
                    f">>> Agent exited {agent_exit_label(agent_rc)} — verify capped to {max_verify} scenario(s)",
                    flush=True,
                )
            verify_ok, verify_results = run_targeted_verification(
                report.rerun_cmds, hub_url=hub_url, cwd=loop_cwd, max_scenarios=max_verify
            )

        log_path = testing_dir / f"{LOOP_LOG_PREFIX}-{layer}-{iter_stamp}.md"
        write_iteration_log(
            log_path,
            layer=layer,
            iteration=iteration,
            stamp=iter_stamp,
            summary_path=summary_path,
            gate_rc=gate_rc,
            agent_rc=agent_rc,
            agent_out=agent_out,
            verify_results=verify_results,
            git_commit=git_commit,
            fix_branch=fix_branch,
        )
        print(f"Iteration log: {log_path}")

        if verify_ok and not args.skip_verify and report.rerun_cmds:
            print("\n>>> Targeted verification PASS — re-running layer gate")
            summary_path = None
            if not prepare_next_iteration_hub(
                hub_url, layer=layer, next_iter=iteration + 1, cwd=loop_cwd, no_restart_hub=args.no_restart_hub
            ):
                return 1
            continue

        if iteration >= args.max_iterations:
            print("Max iterations reached.", file=sys.stderr)
            return 1
        summary_path = None
        if not prepare_next_iteration_hub(
            hub_url, layer=layer, next_iter=iteration + 1, cwd=loop_cwd, no_restart_hub=args.no_restart_hub
        ):
            return 1
        time.sleep(5)

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
