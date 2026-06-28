#!/usr/bin/env python3
"""Run implement-scenarios multiple times and fail if any sweep drops below threshold."""
from __future__ import annotations

import argparse
import re
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
TESTING_DIR = ROOT / "docs" / "testing"

PASS_RE = re.compile(r"^=== PASS: (\S+) ===")
FAIL_RE = re.compile(r"^=== FAIL: (\S+) ===")

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.hub_regression import restart_regression_hub, wait_for_hub  # noqa: E402
from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.release_prep_env import release_prep_env  # noqa: E402


def parse_results(output: str) -> tuple[list[str], list[str]]:
    passed: list[str] = []
    failed: list[str] = []
    for line in output.splitlines():
        m = PASS_RE.match(line.strip())
        if m:
            passed.append(m.group(1))
            continue
        m = FAIL_RE.match(line.strip())
        if m:
            failed.append(m.group(1))
    return passed, failed


def run_sweep(hub_url: str, script: Path) -> tuple[int, str, list[str], list[str]]:
    env = release_prep_env(ROOT)
    proc = subprocess.run(
        [sys.executable, str(script), "--all", "--hub", hub_url],
        cwd=ROOT,
        capture_output=True,
        text=True,
        env=env,
    )
    output = proc.stdout + proc.stderr
    passed, failed = parse_results(output)
    return proc.returncode, output, passed, failed


def main() -> int:
    p = argparse.ArgumentParser(description="Stability gate for implement-scenarios")
    p.add_argument("--runs", type=int, default=3, help="Number of full sweeps (default 3)")
    p.add_argument("--min-pass", type=int, default=20, help="Minimum scenarios that must pass per run (20 total)")
    p.add_argument("--hub", default="http://127.0.0.1:18765")
    p.add_argument("--log-dir", default=str(TESTING_DIR))
    p.add_argument(
        "--restart-between",
        action="store_true",
        help="make stop && make server-regression between sweeps (avoids hub OOM)",
    )
    args = p.parse_args()

    if args.runs < 1:
        print("runs must be >= 1", file=sys.stderr)
        return 1

    script = SCRIPTS_DIR / "implement-scenarios.py"
    if not script.is_file():
        print(f"missing {script}", file=sys.stderr)
        return 1

    log_dir = Path(args.log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    suffix = "-restart" if args.restart_between else ""
    log_path = log_dir / f"parity-stable{suffix}-{stamp}.log"

    lines: list[str] = []
    lines.append(f"# implement-scenarios stability — {stamp} UTC")
    lines.append(
        f"hub={args.hub} runs={args.runs} min_pass={args.min_pass} "
        f"restart_between={args.restart_between}"
    )
    lines.append("")

    preflight_regression_run(ROOT, args.hub, label="parity-stable preflight")

    all_ok = True
    for run in range(1, args.runs + 1):
        lines.append(f"## run {run}/{args.runs}")
        if args.restart_between:
            lines.append("restarting regression hub...")
            env = release_prep_env(ROOT)
            if not restart_regression_hub(ROOT, args.hub, env=env):
                lines.append("hub restart failed")
                all_ok = False
                lines.append(f"RESULT run {run}: FAIL (hub restart)")
                lines.append("")
                continue
            if not wait_for_hub(args.hub, timeout_s=120.0):
                lines.append(f"hub unhealthy after restart: {args.hub}")
                all_ok = False
                lines.append(f"RESULT run {run}: FAIL (hub down after restart)")
                lines.append("")
                continue
            preflight_regression_run(ROOT, args.hub, label=f"parity-stable run-{run} preflight")
            time.sleep(45.0)
        elif not wait_for_hub(args.hub):
            lines.append(f"hub unhealthy before run {run}: {args.hub}")
            all_ok = False
            lines.append(f"RESULT run {run}: FAIL (hub down)")
            lines.append("")
            continue
        elif run > 1:
            time.sleep(45.0)
        code, output, passed, failed = run_sweep(args.hub, script)
        pass_count = len(passed)
        lines.append(f"exit_code={code} pass={pass_count} fail={len(failed)}")
        if failed:
            lines.append(f"failed: {', '.join(failed)}")
        lines.append("")
        lines.append(output.rstrip())
        lines.append("")

        if code != 0 or pass_count < args.min_pass:
            all_ok = False
            lines.append(f"RESULT run {run}: FAIL (need>={args.min_pass} pass)")
        else:
            lines.append(f"RESULT run {run}: OK")
        lines.append("")

    lines.append(f"OVERALL: {'PASS' if all_ok else 'FAIL'} ({args.runs} runs, min {args.min_pass}/sweep)")
    log_text = "\n".join(lines)
    log_path.write_text(log_text, encoding="utf-8")
    print(log_text)
    print(f"\nLog archived: {log_path}")
    return 0 if all_ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
