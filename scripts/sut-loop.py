#!/usr/bin/env python3
"""SUT self-improve loop: Claude Human → local NJ SUT → Claude Judge → Cursor Fix → LoRA rows.

Release-engineering only (not a pack / product feature). Mirror of layer-fix-loop patterns.
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
DEFAULT_TESTING_DIR = ROOT / "docs" / "testing"
DEFAULT_LORA_DIR = DEFAULT_TESTING_DIR / "sut-lora-rows"
PY = sys.executable
LOOP_LOG_PREFIX = "sut-loop"
SUT_DM_USER = "SutScenario"

sys.path.insert(0, str(SCRIPTS_DIR))

from lib.cursor_fix_agent import (  # noqa: E402
    DEFAULT_AGENT_TIMEOUT_S,
    RECOVERABLE_AGENT_EXITS,
    agent_binary,
    agent_exit_label,
    build_agent_invoke_message,
    run_fix_agent,
)
from lib.eval_telemetry import finish, new_run  # noqa: E402
from lib.fix_loop_git import commit_iteration_changes, prepare_fix_loop_cwd  # noqa: E402
from lib.release_prep_env import apply_release_prep_env, provision_hub_automation_key  # noqa: E402
from lib.regression_boot import boot_regression_stack, restart_hub_for_live_run  # noqa: E402
from lib import collab_hub as hub  # noqa: E402
from lib.sut_episode import (  # noqa: E402
    fix_enabled,
    list_episodes,
    load_episode,
    validate_sut_episode,
)
from lib.sut_human import next_user_turn  # noqa: E402
from lib.sut_judge import JudgeVerdict, judge_episode  # noqa: E402
from lib.sut_lora_emit import append_jsonl, rows_from_failure  # noqa: E402
from lib.sut_prompt import build_sut_agent_prompt  # noqa: E402


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


def _last_agent_content(base: str, channel: str, agent: str) -> str:
    msgs = hub.chat_agent_messages(hub.list_messages(base, channel, 200))
    want = agent.strip().lstrip("@")
    for m in reversed(msgs):
        if (m.get("from") or {}).get("name") == want:
            return (m.get("content") or "").strip()
    return ""


def ensure_sut_channel(base: str, episode: dict[str, Any]) -> str | None:
    agent = str(episode.get("target_agent") or "").strip()
    user = str(episode.get("dm_user") or SUT_DM_USER).strip()
    ch_type = str(episode.get("channel_type") or "dm").strip().lower()
    if ch_type == "dm":
        return hub.ensure_dm_channel(base, user, agent)
    channel = str(episode.get("channel") or f"sut-{episode.get('name', 'episode')}").strip()
    required = episode.get("required_agents") or [agent]
    ok, failed = hub.ensure_channel_with_agents(
        base, channel, required, episode.get("description") or "SUT self-improve"
    )
    if not ok:
        print(f"FAIL: could not ensure channel {channel}: {failed}", file=sys.stderr)
        return None
    return channel


def run_episode_once(
    *,
    hub_url: str,
    episode_name: str,
    episode: dict[str, Any],
    dry_run: bool = False,
) -> tuple[bool, list[dict[str, str]], JudgeVerdict, dict[str, Any]]:
    """Simulate + judge one episode. Returns (passed, transcript, verdict, telemetry)."""
    telemetry = new_run("sut", episode_name)
    transcript: list[dict[str, str]] = []
    empty = JudgeVerdict(passed=False, reason="not run", failure_kind="infra")

    if dry_run:
        print(f"  [dry-run] would simulate episode {episode_name}")
        return True, transcript, JudgeVerdict(passed=True, reason="dry-run skip"), telemetry

    target = str(episode.get("target_agent") or "").strip()
    required = episode.get("required_agents") or [target]
    ok, missing = hub.verify_agents_online(hub_url, list(required))
    if not ok:
        return False, transcript, JudgeVerdict(
            passed=False,
            reason=f"agents offline: {', '.join(missing)}",
            failure_kind="infra",
        ), telemetry

    channel = ensure_sut_channel(hub_url, episode)
    if not channel:
        return False, transcript, JudgeVerdict(
            passed=False, reason="could not create SUT channel", failure_kind="infra"
        ), telemetry

    hub.clear_channel_history(hub_url, channel)
    time.sleep(1.0)

    # Optional seed steps: only support send / wait_reply for setup
    for step in episode.get("steps") or []:
        if not isinstance(step, dict):
            continue
        action = str(step.get("action") or "").strip()
        if action == "send":
            content = str(step.get("content") or "").strip()
            if not content:
                continue
            baseline = hub.count_chat_agent_messages(
                hub.list_messages(hub_url, channel, 200), target
            )
            code, _ = hub.send_message(
                hub_url,
                channel,
                content,
                from_name=str(episode.get("dm_user") or SUT_DM_USER),
                metadata=step.get("metadata")
                if isinstance(step.get("metadata"), dict)
                else {"conversation_mode": "chat", "editor_mode": "ask"},
            )
            if code != 200:
                return False, transcript, JudgeVerdict(
                    passed=False, reason=f"seed send failed HTTP {code}", failure_kind="infra"
                ), telemetry
            transcript.append({"role": "user", "content": content})
            ok_wait, detail = hub.wait_chat_reply(
                hub_url,
                channel,
                from_agent=target,
                baseline_count=baseline,
                timeout=hub.parse_timeout(step.get("timeout", 180)),
                detect_failures=True,
            )
            if not ok_wait:
                return False, transcript, JudgeVerdict(
                    passed=False, reason=f"seed wait_reply: {detail}", failure_kind="infra"
                ), telemetry
            reply = _last_agent_content(hub_url, channel, target)
            transcript.append({"role": "assistant", "content": reply})
            print(f"  seed: user→sut ok ({len(reply)} chars)")
        elif action == "wait_reply":
            continue  # paired with send above when using seed send

    human = episode.get("human") if isinstance(episode.get("human"), dict) else {}
    max_turns = int(human.get("max_turns") or 4)
    reply_timeout = float(os.environ.get("NJ_SUT_REPLY_TIMEOUT", "180"))

    for turn_i in range(1, max_turns + 1):
        print(f"  human-sim turn {turn_i}/{max_turns}…", flush=True)
        ht = next_user_turn(
            hub_base=hub_url,
            human=human,
            transcript=transcript,
            turn_index=turn_i,
        )
        if ht.kind == "error":
            return False, transcript, JudgeVerdict(
                passed=False, reason=ht.text, failure_kind="infra"
            ), telemetry
        if ht.kind == "stop":
            print(f"  human-sim STOP: {ht.text}")
            break

        user_msg = ht.text
        print(f"  user: {user_msg[:120]}{'…' if len(user_msg) > 120 else ''}")
        baseline = hub.count_chat_agent_messages(hub.list_messages(hub_url, channel, 200), target)
        code, _ = hub.send_message(
            hub_url,
            channel,
            user_msg,
            from_name=str(episode.get("dm_user") or SUT_DM_USER),
            metadata={"conversation_mode": "chat", "editor_mode": "ask", "sut_eval": True},
        )
        if code != 200:
            return False, transcript, JudgeVerdict(
                passed=False, reason=f"SUT send failed HTTP {code}", failure_kind="infra"
            ), telemetry
        transcript.append({"role": "user", "content": user_msg})

        ok_wait, detail = hub.wait_chat_reply(
            hub_url,
            channel,
            from_agent=target,
            baseline_count=baseline,
            timeout=reply_timeout,
            detect_failures=True,
        )
        if not ok_wait:
            transcript.append({"role": "assistant", "content": f"(no reply: {detail})"})
            print(f"  sut wait FAIL: {detail}")
            break
        reply = _last_agent_content(hub_url, channel, target)
        transcript.append({"role": "assistant", "content": reply})
        print(f"  sut: {reply[:120]}{'…' if len(reply) > 120 else ''}")

    if not transcript:
        return False, transcript, JudgeVerdict(
            passed=False, reason="empty transcript", failure_kind="infra"
        ), telemetry

    print("  judging…", flush=True)
    metrics_note = f"turns={len(transcript)} target=@{target}"
    verdict = judge_episode(
        hub_base=hub_url,
        episode=episode,
        transcript=transcript,
        metrics_note=metrics_note,
    )
    telemetry["attempts"] = 1
    telemetry["passed_at_1"] = verdict.passed
    finish(telemetry, eventual_pass=verdict.passed)
    print(f"  judge: {'PASS' if verdict.passed else 'FAIL'} — {verdict.reason[:160]}")
    return verdict.passed, transcript, verdict, telemetry


def write_report(
    path: Path,
    *,
    episode_name: str,
    stamp: str,
    iteration: int,
    passed: bool,
    transcript: list[dict[str, str]],
    verdict: JudgeVerdict,
    agent_rc: int | None,
    lora_path: str,
    git_commit: str = "",
) -> None:
    lines = [
        f"# sut-loop — {episode_name} — iteration {iteration} — {stamp} UTC",
        "",
        f"passed={passed}",
        f"failure_kind={verdict.failure_kind}",
        f"score={verdict.score}",
        f"reason={verdict.reason}",
        f"agent_rc={agent_rc}",
        f"git_commit={git_commit or '(none)'}",
        f"lora_rows={lora_path or '(none)'}",
        "",
        "## Transcript",
        "",
    ]
    for t in transcript:
        role = t.get("role", "?")
        content = (t.get("content") or "").strip()
        lines.append(f"### {role}")
        lines.append("")
        lines.append(content)
        lines.append("")
    if verdict.gold_output:
        lines.extend(["## Gold output", "", verdict.gold_output, ""])
    if verdict.raw:
        lines.extend(["## Judge raw", "", "```text", verdict.raw[:8000], "```", ""])
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--scenario", help="Episode stem under scenarios/sut/")
    p.add_argument("--all", action="store_true", help="Run all sut episodes")
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--lora-dir", default=str(DEFAULT_LORA_DIR))
    p.add_argument("--max-iterations", type=int, default=3)
    p.add_argument("--skip-agent", action="store_true")
    p.add_argument("--skip-verify", action="store_true")
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--no-restart-hub", action="store_true")
    p.add_argument("--skip-boot", action="store_true")
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--model", help="Cursor agent model")
    p.add_argument("--prefer-sdk", action="store_true")
    p.add_argument(
        "--agent-timeout",
        type=int,
        default=int(os.environ.get("NJ_AGENT_TIMEOUT", str(DEFAULT_AGENT_TIMEOUT_S))),
    )
    p.add_argument("--cwd", type=Path, default=ROOT)
    p.add_argument("--no-commit", action="store_true")
    p.add_argument("--fix-branch", help="Git branch for fixes")
    p.add_argument("--base-branch", help="Base ref when creating fix branch")
    p.add_argument("--use-worktree", action="store_true", default=False)
    p.add_argument("--no-worktree", action="store_false", dest="use_worktree")
    p.add_argument("--list", action="store_true")
    p.add_argument("--validate", action="store_true", help="Validate episode contracts only")
    args = p.parse_args()

    if args.list:
        for name in list_episodes():
            try:
                ep = load_episode(name)
                desc = (ep.get("description") or "").strip()
                print(f"{name}: {desc}" if desc else name)
            except (OSError, json.JSONDecodeError, ValueError) as exc:
                print(f"{name}: (invalid: {exc})")
        return 0

    if args.validate:
        errors: list[str] = []
        for name in list_episodes():
            try:
                ep = load_episode(name)
            except (OSError, json.JSONDecodeError, ValueError) as exc:
                errors.append(f"sut/{name}.json: {exc}")
                continue
            errors.extend(validate_sut_episode(f"sut/{name}.json", ep))
        if errors:
            for e in errors:
                print(e, file=sys.stderr)
            return 1
        print(f"OK: {len(list_episodes())} sut episode(s) valid")
        return 0

    names = list_episodes()
    if args.scenario:
        stem = args.scenario.strip().removesuffix(".json")
        if stem not in names:
            print(f"Unknown episode {stem!r}. Use --list.", file=sys.stderr)
            return 2
        names = [stem]
    elif not args.all:
        print("Specify --scenario NAME or --all (see --list)", file=sys.stderr)
        return 2

    if not names:
        print("No episodes under scenarios/sut/", file=sys.stderr)
        return 1

    if args.max_iterations < 1:
        print("max-iterations must be >= 1", file=sys.stderr)
        return 1

    apply_release_prep_env(ROOT)
    provision_hub_automation_key(ROOT)
    testing_dir = Path(args.log_dir)
    testing_dir.mkdir(parents=True, exist_ok=True)
    lora_dir = Path(args.lora_dir)
    lora_dir.mkdir(parents=True, exist_ok=True)
    hub_url = args.hub.rstrip("/")

    if args.dry_run:
        args.skip_agent = True
        args.skip_verify = True

    if not args.skip_boot and not args.dry_run:
        if not boot_regression_stack(
            ROOT,
            hub_url,
            label="sut-loop",
            no_restart_hub=args.no_restart_hub,
        ):
            print("Regression boot failed", file=sys.stderr)
            return 1

    loop_stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    repo_root = args.cwd.resolve()
    loop_cwd = repo_root
    fix_branch = args.fix_branch or f"sut-loop/{loop_stamp}"

    if not args.no_commit and not args.dry_run and not args.skip_agent:
        git_rc, loop_cwd, fix_branch = prepare_fix_loop_cwd(
            repo_root,
            branch=fix_branch,
            base_branch=args.base_branch,
            use_worktree=bool(args.use_worktree),
            no_commit=args.no_commit,
            dry_run=args.dry_run,
        )
        if git_rc != 0:
            print("Git branch/worktree setup failed; use --no-commit to skip.", file=sys.stderr)
            return git_rc

    overall_rc = 0
    for episode_name in names:
        episode = load_episode(episode_name)
        contract_errs = validate_sut_episode(f"sut/{episode_name}.json", episode)
        if contract_errs:
            for e in contract_errs:
                print(e, file=sys.stderr)
            overall_rc = 1
            continue

        print(f"\n{'=' * 72}\n=== SUT episode {episode_name} ===\n{'=' * 72}")
        episode_passed = False
        lora_file = lora_dir / f"{episode_name}-{loop_stamp}.jsonl"

        for iteration in range(1, args.max_iterations + 1):
            iter_stamp = f"{loop_stamp}-iter{iteration}"
            print(f"\n--- iteration {iteration}/{args.max_iterations} ---")

            passed, transcript, verdict, _tel = run_episode_once(
                hub_url=hub_url,
                episode_name=episode_name,
                episode=episode,
                dry_run=args.dry_run,
            )

            report_path = testing_dir / f"{LOOP_LOG_PREFIX}-{episode_name}-{iter_stamp}.md"
            agent_rc: int | None = None
            git_commit = ""

            if not passed and verdict.failure_kind != "infra" and not args.dry_run:
                rows = rows_from_failure(
                    episode_name=episode_name,
                    transcript=transcript,
                    gold_output=verdict.gold_output,
                    target_agent=str(episode.get("target_agent") or ""),
                )
                n = append_jsonl(lora_file, rows)
                if n:
                    print(f"  emitted {n} LoRA row(s) → {lora_file}")

            if passed:
                write_report(
                    report_path,
                    episode_name=episode_name,
                    stamp=iter_stamp,
                    iteration=iteration,
                    passed=True,
                    transcript=transcript,
                    verdict=verdict,
                    agent_rc=None,
                    lora_path=str(lora_file) if lora_file.is_file() else "",
                )
                print(f"=== PASS {episode_name} — {report_path}")
                episode_passed = True
                break

            # FAIL path
            write_report(
                report_path,
                episode_name=episode_name,
                stamp=iter_stamp,
                iteration=iteration,
                passed=False,
                transcript=transcript,
                verdict=verdict,
                agent_rc=None,
                lora_path=str(lora_file) if lora_file.is_file() else "",
            )

            if verdict.failure_kind == "infra":
                print(f"Infra failure — not invoking Cursor: {verdict.reason}", file=sys.stderr)
                overall_rc = 1
                break

            if args.skip_agent or not fix_enabled(episode):
                print("  skip Cursor fix (--skip-agent or fix.enabled=false)")
                if iteration >= args.max_iterations:
                    overall_rc = 1
                break

            prompt = build_sut_agent_prompt(
                episode=episode,
                episode_name=episode_name,
                verdict=verdict,
                transcript=transcript,
                report_path=report_path,
            )
            prompt_path = testing_dir / f"{LOOP_LOG_PREFIX}-prompt-{episode_name}-{iter_stamp}.md"
            prompt_path.write_text(prompt + "\n", encoding="utf-8")
            invoke_msg = build_agent_invoke_message(prompt_path, repo_root=loop_cwd)
            print(f"  Cursor prompt: {prompt_path}")

            if not agent_binary() and not args.prefer_sdk:
                print("Cursor CLI 'agent' not on PATH.", file=sys.stderr)
                overall_rc = 127
                break

            agent_log = testing_dir / f"{LOOP_LOG_PREFIX}-agent-{episode_name}-{iter_stamp}.log"
            agent_rc, agent_out = run_fix_agent(
                invoke_msg,
                cwd=loop_cwd,
                model=args.model,
                prefer_sdk=args.prefer_sdk,
                timeout_s=args.agent_timeout,
                log_path=agent_log,
            )
            print(f"  Cursor agent_rc={agent_rc} ({agent_exit_label(agent_rc) if agent_rc else 'ok'})")
            if agent_rc in RECOVERABLE_AGENT_EXITS:
                print(f"  recoverable agent exit: {agent_exit_label(agent_rc)}")

            if not args.no_commit and not args.dry_run:
                commit_rc, _ = commit_iteration_changes(
                    loop_cwd,
                    branch=fix_branch,
                    iteration=iteration,
                    summary_path=report_path,
                )
                if commit_rc == 0:
                    proc = subprocess.run(
                        ["git", "rev-parse", "--short", "HEAD"],
                        cwd=loop_cwd,
                        capture_output=True,
                        text=True,
                    )
                    git_commit = (proc.stdout or "").strip()

            write_report(
                report_path,
                episode_name=episode_name,
                stamp=iter_stamp,
                iteration=iteration,
                passed=False,
                transcript=transcript,
                verdict=verdict,
                agent_rc=agent_rc,
                lora_path=str(lora_file) if lora_file.is_file() else "",
                git_commit=git_commit,
            )
            if agent_out.strip():
                with report_path.open("a", encoding="utf-8") as f:
                    f.write("\n## Cursor agent output\n\n```text\n")
                    f.write(agent_out.strip()[-16000:])
                    f.write("\n```\n")

            if args.skip_verify:
                overall_rc = 1
                break

            if iteration >= args.max_iterations:
                print("Max iterations reached.", file=sys.stderr)
                overall_rc = 1
                break

            if not args.no_restart_hub:
                if not restart_hub_for_live_run(
                    loop_cwd, hub_url, label=f"sut-loop-{episode_name}-iter{iteration + 1}"
                ):
                    overall_rc = 1
                    break
            time.sleep(3)

        if not episode_passed:
            overall_rc = overall_rc or 1

    return overall_rc


if __name__ == "__main__":
    raise SystemExit(main())
