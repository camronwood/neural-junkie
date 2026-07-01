#!/usr/bin/env python3
"""Run CI smoke + live scenario harness; archive one reviewable log under docs/testing/."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
DEFAULT_TESTING_DIR = ROOT / "docs" / "testing"

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.hub_regression import (  # noqa: E402
    HubRecoveryJournal,
    ensure_hub_with_recovery,
    hub_is_healthy,
    read_recovery_log_text,
    recover_regression_hub,
    restart_regression_hub,
    wait_for_hub,
)
from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402
from lib.regression_boot import maybe_boot_regression  # noqa: E402


@dataclass
class StageResult:
    name: str
    status: str  # OK | FAIL | SKIPPED
    exit_code: int = 0
    duration_s: float = 0.0
    output: str = ""
    note: str = ""


@dataclass
class RunReport:
    stamp: str
    hub_url: str
    full: bool
    skip_live: bool
    results: list[StageResult] = field(default_factory=list)
    log_path: Path = Path()
    summary_path: Path = Path()

    @property
    def ran(self) -> int:
        return sum(1 for r in self.results if r.status != "SKIPPED")

    @property
    def passed(self) -> int:
        return sum(1 for r in self.results if r.status == "OK")

    @property
    def failed(self) -> list[StageResult]:
        return [r for r in self.results if r.status == "FAIL"]


def run_cmd(
    name: str,
    cmd: list[str],
    *,
    cwd: Path = ROOT,
    env: dict | None = None,
) -> StageResult:
    merged = release_prep_env(ROOT)
    if env:
        merged.update(env)
    merged["PYTHONUNBUFFERED"] = "1"
    t0 = time.monotonic()
    print(f"\n>>> [{name}] {' '.join(cmd)}", flush=True)
    proc = subprocess.Popen(
        cmd,
        cwd=cwd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        env=merged,
        bufsize=1,
    )
    chunks: list[str] = []
    assert proc.stdout is not None
    for line in proc.stdout:
        sys.stdout.write(line)
        sys.stdout.flush()
        chunks.append(line)
    rc = proc.wait()
    out = "".join(chunks)
    dur = time.monotonic() - t0
    return StageResult(
        name=name,
        status="OK" if rc == 0 else "FAIL",
        exit_code=rc,
        duration_s=dur,
        output=out,
    )


def run_make(target: str, *, env: dict | None = None) -> StageResult:
    return run_cmd(target, ["make", target], env=env)


def ci_stage_names() -> list[str]:
    return [
        "test-all",
        "test-conversation-contract",
        "test-collab-plan",
        "test-scenario-assert",
        "collab-smoke",
        "learning-lora-smoke",
    ]


def live_stage_cmds(hub_url: str, full: bool, verbose: bool) -> list[tuple[str, list[str]]]:
    py = sys.executable
    stages: list[tuple[str, list[str]]] = [
        (
            "collab-preflight",
            [py, "scripts/collab-preflight.py", *(["--require-gemini"] if full else [])],
        ),
        ("implement-scenarios", [py, "scripts/implement-scenarios.py", "--all", "--hub", hub_url]),
        (
            "chat-scenarios-regression",
            [py, "scripts/chat-scenarios.py", "--all", "--tag", "regression", *(["--verbose"] if verbose else [])],
        ),
        (
            "conversation-scenarios-regression",
            [py, "scripts/conversation-scenarios-regression.py", *(["--verbose"] if verbose else [])],
        ),
    ]
    # collab-scenario-regression is a Makefile aggregate (many scenarios)
    stages.append(("collab-scenario-regression", ["make", "collab-scenario-regression"]))
    if full:
        stages.append(
            (
                "collab-scenarios-all",
                [py, "scripts/collab-scenarios.py", "--all", *(["--verbose"] if verbose else [])],
            )
        )
    return stages


def write_reports(report: RunReport, testing_dir: Path) -> None:
    testing_dir.mkdir(parents=True, exist_ok=True)
    report.log_path = testing_dir / f"test-everything-{report.stamp}.log"
    report.summary_path = testing_dir / f"test-everything-{report.stamp}.md"

    lines: list[str] = [
        f"# test-everything — {report.stamp} UTC",
        "",
        f"- Hub: `{report.hub_url}`",
        f"- Full collab sweep (`FULL=1`): `{report.full}`",
        f"- Skip live: `{report.skip_live}`",
        f"- Overall: **{'PASS' if not report.failed else 'FAIL'}** ({report.passed}/{report.ran} stages)",
        "",
        "## Stage summary",
        "",
        "| Stage | Status | Duration |",
        "|-------|--------|----------|",
    ]
    for r in report.results:
        dur = f"{r.duration_s:.0f}s" if r.duration_s else "—"
        note = f" ({r.note})" if r.note else ""
        lines.append(f"| `{r.name}` | {r.status}{note} | {dur} |")

    lines.extend(
        [
            "",
            "## Artifacts",
            "",
            f"- Full log: `{report.log_path}`",
            f"- Hub recovery log: `{testing_dir / f'hub-recovery-{report.stamp}.log'}`",
            "",
        ]
    )

    if report.failed:
        lines.extend(["## Failures (tail)", ""])
        for r in report.failed:
            lines.append(f"### {r.name} (exit {r.exit_code})")
            lines.append("")
            lines.append("```text")
            lines.append((r.output or "(no output)").strip()[-12000:])
            lines.append("```")
            lines.append("")

    recovery_text = read_recovery_log_text()
    if recovery_text.strip():
        lines.extend(["## Hub recovery log", "", "```text", recovery_text.strip()[-12000:], "```", ""])

    report.summary_path.write_text("\n".join(lines) + "\n", encoding="utf-8")

    log_lines: list[str] = [
        f"# test-everything full log — {report.stamp} UTC",
        f"hub={report.hub_url} full={report.full} skip_live={report.skip_live}",
        f"summary={report.summary_path}",
        "",
    ]
    for r in report.results:
        log_lines.append("=" * 72)
        log_lines.append(f"## {r.name} — {r.status} (exit={r.exit_code}, {r.duration_s:.1f}s)")
        if r.note:
            log_lines.append(f"note: {r.note}")
        log_lines.append("")
        log_lines.append((r.output or "").rstrip())
        log_lines.append("")
    log_lines.append("=" * 72)
    log_lines.append(f"OVERALL: {'PASS' if not report.failed else 'FAIL'} ({report.passed}/{report.ran})")
    report.log_path.write_text("\n".join(log_lines) + "\n", encoding="utf-8")


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--full", action="store_true", help="Also run collab-scenarios-all (~1-3h)")
    p.add_argument("--skip-live", action="store_true", help="CI/smoke only (no hub required)")
    p.add_argument("--verbose", action="store_true", help="Verbose live scenario output")
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--stamp", help="Report stamp (default: UTC YYYY-MM-DD-HHMM)")
    p.add_argument(
        "--continue-on-fail",
        action="store_true",
        help="Keep running later stages after a failure (default: stop on first fail)",
    )
    args = p.parse_args()

    apply_release_prep_env(ROOT)

    testing_dir = Path(args.log_dir)
    stamp = args.stamp or datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    report = RunReport(
        stamp=stamp,
        hub_url=args.hub.rstrip("/"),
        full=args.full,
        skip_live=args.skip_live,
    )
    live_env = release_prep_env(ROOT)
    live_env["NEURAL_JUNKIE_HUB_URL"] = report.hub_url
    recovery_log = testing_dir / f"hub-recovery-{stamp}.log"
    live_env["NEURAL_JUNKIE_HUB_RECOVERY_LOG"] = str(recovery_log)
    os.environ["NEURAL_JUNKIE_HUB_RECOVERY_LOG"] = str(recovery_log)
    hub_journal = HubRecoveryJournal()

    print(f"test-everything → {testing_dir}/test-everything-{stamp}.{{md,log}}")

    for target in ci_stage_names():
        result = run_make(target)
        report.results.append(result)
        if result.status == "FAIL" and not args.continue_on_fail:
            write_reports(report, testing_dir)
            _print_final(report)
            return 1

    if args.skip_live:
        report.results.append(
            StageResult(name="live-harness", status="SKIPPED", note="--skip-live")
        )
        write_reports(report, testing_dir)
        _print_final(report)
        return 0 if not report.failed else 1

    print(f"\n>>> Live regression boot")
    if not maybe_boot_regression(report.hub_url, root=ROOT, label="test-everything"):
        report.results.append(
            StageResult(
                name="regression-boot",
                status="FAIL",
                exit_code=1,
                note="Ollama/hub boot failed — see log above",
            )
        )
        write_reports(report, testing_dir)
        _print_final(report)
        return 1

    preflight_regression_run(ROOT, report.hub_url, label="test-everything preflight")

    for name, cmd in live_stage_cmds(report.hub_url, args.full, args.verbose):
        env = live_env if name != "collab-scenario-regression" else {**live_env}
        if not ensure_hub_with_recovery(
            ROOT,
            report.hub_url,
            context=f"pre-stage:{name}",
            journal=hub_journal,
            env=live_env,
        ):
            report.results.append(
                StageResult(
                    name=f"{name}-hub-recovery",
                    status="FAIL",
                    exit_code=1,
                    note="hub recovery exhausted before stage",
                )
            )
            if not args.continue_on_fail:
                break
            continue

        result = run_cmd(name, cmd, env=env)
        report.results.append(result)

        if not hub_is_healthy(report.hub_url):
            if recover_regression_hub(
                ROOT,
                report.hub_url,
                context=f"post-stage:{name}",
                journal=hub_journal,
                env=live_env,
            ):
                if result.status == "FAIL":
                    print(f"\n>>> [{name}] retrying stage after hub recovery")
                    retry = run_cmd(f"{name}-retry-after-hub-recovery", cmd, env=env)
                    retry.note = "retried after hub crash recovery"
                    report.results.append(retry)
                    if retry.status == "OK":
                        result.status = "OK"
                        result.note = "initial run failed (hub crash); retry passed"
            elif not args.continue_on_fail:
                break

        if result.status == "FAIL" and not args.continue_on_fail:
            break

    write_reports(report, testing_dir)
    _print_final(report)
    return 0 if not report.failed else 1


def _print_final(report: RunReport) -> None:
    print("\n=== Summary ===")
    print(f"PASS {report.passed}/{report.ran}")
    if report.failed:
        print("FAILED:", ", ".join(r.name for r in report.failed), file=sys.stderr)
    print(f"Review:   {report.summary_path}")
    print(f"Full log: {report.log_path}")


if __name__ == "__main__":
    raise SystemExit(main())
