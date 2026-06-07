#!/usr/bin/env python3
"""Pre-release live regression bundle: implement + chat + conversation quality."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
TESTING_DIR = ROOT / "docs" / "testing"

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.hub_regression import wait_for_hub  # noqa: E402

STAGES: list[tuple[str, list[str]]] = [
    (
        "implement",
        [sys.executable, "scripts/implement-scenarios.py", "--all"],
    ),
    (
        "chat-regression",
        [sys.executable, "scripts/chat-scenarios.py", "--all", "--tag", "regression"],
    ),
    (
        "conversation-regression",
        [sys.executable, "scripts/conversation-scenarios-regression.py"],
    ),
]


def run_stage(name: str, cmd: list[str], hub_url: str, verbose: bool) -> tuple[int, str]:
    full = list(cmd)
    if any("implement-scenarios.py" in part for part in full) and "--hub" not in full:
        full.extend(["--hub", hub_url])
    # implement-scenarios.py has no --verbose flag
    if verbose and "--verbose" not in full and not any(
        "implement-scenarios.py" in part for part in full
    ):
        full.append("--verbose")
    env = os.environ.copy()
    env.setdefault("NEURAL_JUNKIE_RATE_LIMIT", "0")
    env.setdefault("NEURAL_JUNKIE_HUB_URL", hub_url)
    print(f"\n>>> {' '.join(full)}")
    proc = subprocess.run(full, cwd=ROOT, env=env, capture_output=True, text=True)
    out = (proc.stdout or "") + (proc.stderr or "")
    print(out.rstrip())
    return proc.returncode, out


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(TESTING_DIR))
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--skip-implement", action="store_true")
    p.add_argument("--skip-chat", action="store_true")
    p.add_argument("--skip-conversation", action="store_true")
    args = p.parse_args()

    skip = {
        "implement": args.skip_implement,
        "chat-regression": args.skip_chat,
        "conversation-regression": args.skip_conversation,
    }

    if not wait_for_hub(args.hub):
        print(f"hub unhealthy: {args.hub}", file=sys.stderr)
        print("Start with: make server-regression", file=sys.stderr)
        return 1

    log_dir = Path(args.log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    log_path = log_dir / f"regression-bundle-{stamp}.log"

    lines: list[str] = [
        f"# regression bundle — {stamp} UTC",
        f"hub={args.hub}",
        "",
    ]
    failed: list[str] = []
    ran = 0

    for name, cmd in STAGES:
        if skip.get(name):
            lines.append(f"## {name} — SKIPPED")
            lines.append("")
            continue
        ran += 1
        lines.append(f"## {name}")
        code, output = run_stage(name, cmd, args.hub, args.verbose)
        lines.append(f"exit_code={code}")
        lines.append("")
        lines.append(output.rstrip())
        lines.append("")
        if code != 0:
            failed.append(name)
            lines.append(f"RESULT {name}: FAIL")
        else:
            lines.append(f"RESULT {name}: OK")
        lines.append("")

    passed = ran - len(failed)
    lines.append(f"OVERALL: {'PASS' if not failed else 'FAIL'} ({passed}/{ran} stages)")
    if failed:
        lines.append(f"FAILED: {', '.join(failed)}")

    log_text = "\n".join(lines)
    log_path.write_text(log_text, encoding="utf-8")
    print("\n=== Summary ===")
    print(f"PASS {passed}/{ran}")
    if failed:
        print("FAILED:", ", ".join(failed), file=sys.stderr)
    print(f"Log archived: {log_path}")
    return 0 if not failed else 1


if __name__ == "__main__":
    raise SystemExit(main())
