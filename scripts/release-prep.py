#!/usr/bin/env python3
"""Release prep orchestrator: test-everything-full + parity-restart + model benchmark."""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
DEFAULT_TESTING_DIR = ROOT / "docs" / "testing"
PY = sys.executable

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402
from lib.release_prep_hub import ensure_hub_for_release_prep  # noqa: E402
from lib.hub_regression import recover_regression_hub, read_recovery_log_text, restart_regression_hub, wait_for_hub  # noqa: E402

REVIEW_RE = re.compile(r"^Review:\s+(.+)$", re.MULTILINE)
LOG_RE = re.compile(r"^Full log:\s+(.+)$", re.MULTILINE)
PARITY_LOG_RE = re.compile(r"^Log archived:\s+(.+)$", re.MULTILINE)
BENCH_MD_RE = re.compile(r"^Markdown:\s+(.+)$", re.MULTILINE)
BENCH_JSON_RE = re.compile(r"^JSON:\s+(.+)$", re.MULTILINE)
BENCH_TSV_RE = re.compile(r"^TSV:\s+(.+)$", re.MULTILINE)


@dataclass
class PhaseResult:
    name: str
    status: str  # OK | FAIL | SKIPPED
    exit_code: int = 0
    duration_s: float = 0.0
    output: str = ""
    note: str = ""
    artifacts: list[str] = field(default_factory=list)


@dataclass
class ReleasePrepReport:
    stamp: str
    hub_url: str
    skip_live: bool
    full: bool
    benchmark_suite: str
    phases: list[PhaseResult] = field(default_factory=list)
    summary_path: Path = Path()
    log_path: Path = Path()

    @property
    def ran(self) -> int:
        return sum(1 for p in self.phases if p.status != "SKIPPED")

    @property
    def passed(self) -> int:
        return sum(1 for p in self.phases if p.status == "OK")

    @property
    def failed(self) -> list[PhaseResult]:
        return [p for p in self.phases if p.status == "FAIL"]


def run_phase(name: str, cmd: list[str], *, env: dict | None = None) -> PhaseResult:
    merged = release_prep_env(ROOT)
    if env:
        merged.update(env)
    t0 = time.monotonic()
    print(f"\n>>> [{name}] {' '.join(cmd)}")
    proc = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, env=merged)
    out = (proc.stdout or "") + (proc.stderr or "")
    if out.strip():
        tail = out.rstrip()
        if len(tail) > 12000:
            print("...(truncated in terminal)...")
            print(tail[-12000:])
        else:
            print(tail)
    return PhaseResult(
        name=name,
        status="OK" if proc.returncode == 0 else "FAIL",
        exit_code=proc.returncode,
        duration_s=time.monotonic() - t0,
        output=out,
    )


def first_match(pattern: re.Pattern[str], text: str) -> str:
    m = pattern.search(text)
    return m.group(1).strip() if m else ""


def parity_scorecard_section(report: ReleasePrepReport) -> list[str]:
    """Append Cursor parity gate summary for release prep."""
    impl = next((p for p in report.phases if p.name == "test-parity-stable-restart"), None)
    parity_full = [p for p in report.phases if "parity-full" in (p.note or "")]
    lines = [
        "",
        "## Parity scorecard (native Cursor)",
        "",
        "| Dimension | Release gate | Status |",
        "|-----------|--------------|--------|",
        f"| Implement regression (16 scenarios) | `make implement-scenarios` | {impl.status if impl else '—'} |",
        f"| Stability (3× restart) | `make test-parity-stable-restart` | {impl.status if impl else '—'} |",
        "| Parity contract | `make parity-scenarios` | run via `make test-parity-full-restart` |",
        "| Agent Runtime v2 | `features.agent_runtime_v2` default on | config |",
        "| Model-aware context | `ollama.num_ctx` → agent budget | config |",
        "| Disk-backed code index | SQLite under `~/.neural-junkie/codeindex/` | build-time |",
        "",
        "## Child reports",
        "",
    ]
    return lines


def write_reports(report: ReleasePrepReport, testing_dir: Path) -> None:
    testing_dir.mkdir(parents=True, exist_ok=True)
    report.summary_path = testing_dir / f"release-prep-{report.stamp}.md"
    report.log_path = testing_dir / f"release-prep-{report.stamp}.log"

    overall = "PASS" if not report.failed else "FAIL"
    lines = [
        f"# Release prep — {report.stamp} UTC",
        "",
        f"- Hub: `{report.hub_url}`",
        f"- Skip live: `{report.skip_live}`",
        f"- test-everything full: `{report.full}`",
        f"- Benchmark suite: `{report.benchmark_suite}`",
        f"- Overall: **{overall}** ({report.passed}/{report.ran} phases)",
        "",
        "## Phase summary",
        "",
        "| Phase | Status | Duration | Artifacts |",
        "|-------|--------|----------|-----------|",
    ]
    for p in report.phases:
        dur = f"{p.duration_s:.0f}s" if p.duration_s else "—"
        note = f" ({p.note})" if p.note else ""
        arts = ", ".join(f"`{a}`" for a in p.artifacts) if p.artifacts else "—"
        lines.append(f"| `{p.name}` | {p.status}{note} | {dur} | {arts} |")

    lines.extend(parity_scorecard_section(report))
    for p in report.phases:
        if p.artifacts:
            lines.append(f"### {p.name}")
            for a in p.artifacts:
                lines.append(f"- `{a}`")
            lines.append("")

    if report.failed:
        lines.extend(["## Failures (tail)", ""])
        for p in report.failed:
            lines.append(f"### {p.name} (exit {p.exit_code})")
            lines.append("")
            lines.append("```text")
            lines.append((p.output or "(no output)").strip()[-16000:])
            lines.append("```")
            lines.append("")

    recovery_text = read_recovery_log_text()
    if recovery_text.strip():
        recovery_path = os.environ.get("NEURAL_JUNKIE_HUB_RECOVERY_LOG", "")
        lines.extend(
            [
                "## Hub recovery log",
                "",
                *( [f"Artifact: `{recovery_path}`", ""] if recovery_path else [] ),
                "```text",
                recovery_text.strip()[-16000:],
                "```",
                "",
            ]
        )

    report.summary_path.write_text("\n".join(lines) + "\n", encoding="utf-8")

    log_lines = [
        f"# release-prep full log — {report.stamp} UTC",
        f"hub={report.hub_url} skip_live={report.skip_live} full={report.full} suite={report.benchmark_suite}",
        f"summary={report.summary_path}",
        "",
    ]
    for p in report.phases:
        log_lines.append("=" * 72)
        log_lines.append(f"## {p.name} — {p.status} (exit={p.exit_code}, {p.duration_s:.1f}s)")
        if p.note:
            log_lines.append(f"note: {p.note}")
        if p.artifacts:
            log_lines.append("artifacts:")
            for a in p.artifacts:
                log_lines.append(f"  - {a}")
        log_lines.append("")
        log_lines.append((p.output or "").rstrip())
        log_lines.append("")
    log_lines.append("=" * 72)
    log_lines.append(f"OVERALL: {overall} ({report.passed}/{report.ran})")
    report.log_path.write_text("\n".join(log_lines) + "\n", encoding="utf-8")


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--stamp", help="UTC stamp for reports (default: now YYYY-MM-DD-HHMM)")
    p.add_argument("--skip-live", action="store_true", help="CI/smoke only (skip live phases)")
    p.add_argument("--skip-everything", action="store_true", help="Skip test-everything phase")
    p.add_argument("--skip-parity", action="store_true", help="Skip test-parity-stable-restart")
    p.add_argument("--skip-benchmark", action="store_true", help="Skip model-benchmark phase")
    p.add_argument("--no-full", action="store_true", help="test-everything without collab-scenarios-all")
    p.add_argument("--benchmark-suite", default="quick", help="model-benchmark suite (default: quick — full ≤24B roster)")
    p.add_argument("--benchmark-models", help="Comma-separated Ollama tags for benchmark")
    p.add_argument(
        "--no-pull-models",
        action="store_true",
        help="Do not pull missing benchmark models via hub (default: pull before each model)",
    )
    p.add_argument(
        "--benchmark-allow-large",
        action="store_true",
        help="Allow benchmark models above the max-params-b cap (default 24B)",
    )
    p.add_argument(
        "--no-restart-hub",
        action="store_true",
        help="Do not start/restart regression hub when Gemini judge is unhealthy",
    )
    p.add_argument("--verbose", action="store_true")
    p.add_argument(
        "--stop-on-fail",
        action="store_true",
        help="Stop after first failed phase (default: run all phases)",
    )
    args = p.parse_args()

    apply_release_prep_env(ROOT)
    stamp = args.stamp or datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    hub_url = args.hub.rstrip("/")
    testing_dir = Path(args.log_dir)
    recovery_log = testing_dir / f"hub-recovery-{stamp}.log"
    hub_recovery_env = {
        "NEURAL_JUNKIE_HUB_URL": hub_url,
        "NEURAL_JUNKIE_HUB_RECOVERY_LOG": str(recovery_log),
    }
    os.environ["NEURAL_JUNKIE_HUB_RECOVERY_LOG"] = str(recovery_log)
    report = ReleasePrepReport(
        stamp=stamp,
        hub_url=hub_url,
        skip_live=args.skip_live,
        full=not args.no_full,
        benchmark_suite=args.benchmark_suite,
    )

    print(f"release-prep → {testing_dir}/release-prep-{stamp}.{{md,log}}")

    if not args.skip_live:
        print("\n>>> [release-prep setup] env + hub + Gemini judge")
        if not ensure_hub_for_release_prep(
            hub_url,
            root=ROOT,
            allow_restart=not args.no_restart_hub,
            verbose=args.verbose,
        ):
            write_reports(report, testing_dir)
            print("Release prep aborted: hub/Gemini judge not ready.", file=sys.stderr)
            return 1
        preflight_regression_run(ROOT, hub_url, label="release-prep preflight")
        preflight_cmd = [
            PY,
            str(SCRIPTS_DIR / "collab-preflight.py"),
            "--hub",
            hub_url,
            "--require-gemini",
            "--skip-judge-smoke",
        ]
        preflight_phase = run_phase("release-prep-preflight", preflight_cmd, env=hub_recovery_env)
        if preflight_phase.status == "FAIL":
            report.phases.append(preflight_phase)
            write_reports(report, testing_dir)
            _print_final(report)
            return 1

    if not args.skip_everything:
        te_cmd = [
            PY,
            str(SCRIPTS_DIR / "test-everything.py"),
            "--hub",
            hub_url,
            "--stamp",
            stamp,
            "--log-dir",
            str(testing_dir),
            "--continue-on-fail",
        ]
        if report.full:
            te_cmd.append("--full")
        if args.skip_live:
            te_cmd.append("--skip-live")
        if args.verbose:
            te_cmd.append("--verbose")
        phase = run_phase("test-everything", te_cmd, env=hub_recovery_env)
        review = first_match(REVIEW_RE, phase.output)
        log_file = first_match(LOG_RE, phase.output)
        if review:
            phase.artifacts.append(review)
        if log_file:
            phase.artifacts.append(log_file)
        if not review:
            phase.artifacts.append(str(testing_dir / f"test-everything-{stamp}.md"))
        report.phases.append(phase)
        if phase.status == "FAIL" and args.stop_on_fail:
            write_reports(report, testing_dir)
            _print_final(report)
            return 1

        print("\n>>> [release-prep] restarting hub after test-everything (fresh hub for parity/benchmark)")
        env = release_prep_env(ROOT)
        if not restart_regression_hub(ROOT, hub_url, env=env):
            phase = PhaseResult(
                name="hub-restart-after-everything",
                status="FAIL",
                exit_code=1,
                note="hub restart failed before parity/benchmark",
            )
            report.phases.append(phase)
            write_reports(report, testing_dir)
            _print_final(report)
            return 1
        if not wait_for_hub(hub_url, timeout_s=120.0):
            phase = PhaseResult(
                name="hub-restart-after-everything",
                status="FAIL",
                exit_code=1,
                note="hub unhealthy after restart before parity/benchmark",
            )
            report.phases.append(phase)
            write_reports(report, testing_dir)
            _print_final(report)
            return 1
        time.sleep(45.0)
        preflight_regression_run(ROOT, hub_url, label="release-prep post-everything preflight")

    if not args.skip_live and not args.skip_parity:
        parity_cmd = [
            PY,
            str(SCRIPTS_DIR / "implement-scenarios-stable.py"),
            "--runs",
            "3",
            "--min-pass",
            "17",
            "--restart-between",
            "--hub",
            hub_url,
            "--log-dir",
            str(testing_dir),
        ]
        phase = run_phase("test-parity-stable-restart", parity_cmd, env=hub_recovery_env)
        parity_log = first_match(PARITY_LOG_RE, phase.output)
        if parity_log:
            phase.artifacts.append(parity_log)
        report.phases.append(phase)
        if phase.status == "FAIL" and args.stop_on_fail:
            write_reports(report, testing_dir)
            _print_final(report)
            return 1
    elif args.skip_parity:
        report.phases.append(PhaseResult(name="test-parity-stable-restart", status="SKIPPED", note="--skip-parity"))
    else:
        report.phases.append(
            PhaseResult(name="test-parity-stable-restart", status="SKIPPED", note="--skip-live")
        )

    if not args.skip_live and not args.skip_benchmark:
        bench_cmd = [
            PY,
            str(SCRIPTS_DIR / "model-benchmark-suite.py"),
            "--hub",
            hub_url,
            "--suite",
            args.benchmark_suite,
            "--out-dir",
            str(testing_dir),
            "--pull",
            "--min-winner-pass-rate",
            "1.0",
        ]
        if args.no_pull_models:
            bench_cmd.remove("--pull")
        if args.benchmark_allow_large:
            bench_cmd.append("--allow-large-models")
        if args.benchmark_models:
            bench_cmd.extend(["--models", args.benchmark_models])
        if args.verbose:
            bench_cmd.append("--verbose")
        phase = run_phase("model-benchmark", bench_cmd, env=hub_recovery_env)
        for pattern in (BENCH_MD_RE, BENCH_JSON_RE, BENCH_TSV_RE):
            path = first_match(pattern, phase.output)
            if path:
                phase.artifacts.append(path)
        report.phases.append(phase)
        if phase.status == "FAIL" and args.stop_on_fail:
            write_reports(report, testing_dir)
            _print_final(report)
            return 1
    elif args.skip_benchmark:
        report.phases.append(PhaseResult(name="model-benchmark", status="SKIPPED", note="--skip-benchmark"))
    else:
        report.phases.append(PhaseResult(name="model-benchmark", status="SKIPPED", note="--skip-live"))

    write_reports(report, testing_dir)
    _print_final(report)
    return 0 if not report.failed else 1


def _print_final(report: ReleasePrepReport) -> None:
    print("\n=== Release prep summary ===")
    print(f"PASS {report.passed}/{report.ran}")
    if report.failed:
        print("FAILED:", ", ".join(p.name for p in report.failed), file=sys.stderr)
    print(f"Review:   {report.summary_path}")
    print(f"Full log: {report.log_path}")


if __name__ == "__main__":
    raise SystemExit(main())
