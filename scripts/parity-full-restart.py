#!/usr/bin/env python3
"""Run implement + parity scenarios 3× with hub restart between sweeps."""
from __future__ import annotations

import argparse
import re
import subprocess
import sys
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


def run_script(script: str, hub_url: str) -> tuple[int, str, list[str], list[str]]:
    env = release_prep_env(ROOT)
    proc = subprocess.run(
        [sys.executable, str(SCRIPTS_DIR / script), "--all", "--hub", hub_url],
        cwd=ROOT,
        capture_output=True,
        text=True,
        env=env,
    )
    output = proc.stdout + proc.stderr
    passed, failed = parse_results(output)
    return proc.returncode, output, passed, failed


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--runs", type=int, default=3)
    p.add_argument("--hub", default="http://127.0.0.1:18765")
    p.add_argument("--min-implement", type=int, default=16)
    p.add_argument("--log-dir", type=Path, default=TESTING_DIR)
    args = p.parse_args()

    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    log_path = args.log_dir / f"parity-full-restart-{stamp}.log"
    args.log_dir.mkdir(parents=True, exist_ok=True)

    if not wait_for_hub(args.hub, timeout_s=5):
        print(f"Hub not reachable at {args.hub}", file=sys.stderr)
        return 1
    preflight_regression_run(ROOT)

    sweeps_ok = 0
    lines: list[str] = [f"# parity-full-restart — {stamp} UTC", f"hub={args.hub}", ""]

    for run in range(1, args.runs + 1):
        print(f"\n=== sweep {run}/{args.runs} ===")
        impl_code, impl_out, impl_pass, impl_fail = run_script("implement-scenarios.py", args.hub)
        parity_code, parity_out, parity_pass, parity_fail = run_script("parity-scenarios.py", args.hub)
        impl_ok = impl_code == 0 and len(impl_pass) >= args.min_implement and not impl_fail
        parity_ok = parity_code == 0 and not parity_fail
        sweep_ok = impl_ok and parity_ok
        if sweep_ok:
            sweeps_ok += 1
        status = "PASS" if sweep_ok else "FAIL"
        print(f"sweep {run}: {status} (implement {len(impl_pass)}/{args.min_implement}, parity {len(parity_pass)} pass)")
        lines.extend(
            [
                f"## sweep {run} — {status}",
                f"implement: exit={impl_code} pass={len(impl_pass)} fail={len(impl_fail)}",
                f"parity: exit={parity_code} pass={len(parity_pass)} fail={len(parity_fail)}",
                "",
            ]
        )
        if not sweep_ok:
            lines.append("```text")
            lines.append((impl_out + parity_out)[-12000:])
            lines.append("```")
            lines.append("")
        if run < args.runs:
            env = release_prep_env(ROOT)
            if not restart_regression_hub(ROOT, args.hub, env=env):
                print("hub restart failed", file=sys.stderr)
                return 1
            if not wait_for_hub(args.hub, timeout_s=120):
                print("hub did not come back after restart", file=sys.stderr)
                return 1

    overall = sweeps_ok == args.runs
    lines.append(f"OVERALL: {'PASS' if overall else 'FAIL'} ({sweeps_ok}/{args.runs} sweeps)")
    log_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"\nLog: {log_path}")
    print(f"OVERALL: {'PASS' if overall else 'FAIL'} ({sweeps_ok}/{args.runs})")
    return 0 if overall else 1


if __name__ == "__main__":
    raise SystemExit(main())
