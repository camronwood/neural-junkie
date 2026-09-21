#!/usr/bin/env python3
"""Away overnight orchestrator: real-user gates → fix PRs → agent-ready → morning RC.

Order (see docs/AWAY_OPERATIONS.md):
  1. make test-all
  2. make layer-gate LAYER=user-flows  (real user journeys)
  3. thin climb canary (implement + chat) unless AWAY_GATE=full → layer-climb
  4. make desktop-e2e (UI click journeys) unless SKIP_DESKTOP_E2E=1
  5. on failure → layer-fix-loop / Cursor → open agent-pr when possible
  6. drain agent-ready via away-agent-loop until AWAY_DEADLINE_CT
  7. wait for open agent-pr merges
  8. away-morning-rc.sh
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path
from zoneinfo import ZoneInfo

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
PY = sys.executable
CT = ZoneInfo("America/Chicago")

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402


def run(cmd: list[str], *, cwd: Path = ROOT) -> int:
    env = release_prep_env(ROOT)
    env["PYTHONUNBUFFERED"] = "1"
    print(f"\n>>> {' '.join(cmd)}", flush=True)
    proc = subprocess.run(cmd, cwd=cwd, env=env)
    return proc.returncode


def past_deadline(deadline_ct: str) -> bool:
    """deadline_ct like '05:00' in America/Chicago."""
    now = datetime.now(CT)
    try:
        hh, mm = deadline_ct.strip().split(":")
        deadline = now.replace(hour=int(hh), minute=int(mm), second=0, microsecond=0)
    except ValueError:
        return False
    # If deadline is early morning and now is evening, not past yet.
    if now.hour >= 12 and deadline.hour < 12:
        return False
    return now >= deadline


def open_agent_pr_count() -> int:
    proc = subprocess.run(
        ["gh", "pr", "list", "--label", "agent-pr", "--state", "open", "--json", "number"],
        cwd=ROOT,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        return 0
    try:
        return len(json.loads(proc.stdout or "[]"))
    except json.JSONDecodeError:
        return 0


def wait_for_prs(timeout_s: int = 3600) -> None:
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        n = open_agent_pr_count()
        if n == 0:
            print("No open agent-pr PRs.")
            return
        print(f"Waiting for {n} agent-pr merge(s)...")
        time.sleep(60)
    print("WARN: timed out waiting for agent-pr merges; continuing to morning RC.")


def run_fix_loop(layer: str, max_iter: int) -> int:
    return run(
        [
            "make",
            "layer-fix-loop",
            f"LAYER={layer}",
            f"MAX_ITER={max_iter}",
        ]
    )


def append_status(lines: list[str]) -> Path:
    stamp = datetime.now(CT).strftime("%Y-%m-%d")
    path = ROOT / "docs" / "testing" / f"away-morning-{stamp}.md"
    path.parent.mkdir(parents=True, exist_ok=True)
    existing = path.read_text(encoding="utf-8") if path.is_file() else f"# Away morning status — {stamp}\n\n"
    block = "\n".join(f"- {line}" for line in lines) + "\n"
    if "## Overnight gates" not in existing:
        existing += "\n## Overnight gates\n\n"
    existing += block
    path.write_text(existing, encoding="utf-8")
    return path


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--max-iter", type=int, default=int(os.environ.get("MAX_ITER") or "3"))
    ap.add_argument("--max-issues", type=int, default=int(os.environ.get("MAX_ISSUES") or "3"))
    ap.add_argument(
        "--deadline-ct",
        default=os.environ.get("AWAY_DEADLINE_CT") or "05:00",
        help="Stop claiming new work after this America/Chicago clock time",
    )
    ap.add_argument("--skip-desktop-e2e", action="store_true", default=os.environ.get("SKIP_DESKTOP_E2E") == "1")
    ap.add_argument("--skip-morning-rc", action="store_true")
    args = ap.parse_args()

    apply_release_prep_env(ROOT)
    status: list[str] = []
    away_gate = (os.environ.get("AWAY_GATE") or "user-flows").strip().lower()

    # 1. Unit gate
    rc = run(["make", "test-all"])
    status.append(f"`make test-all`: {'PASS' if rc == 0 else 'FAIL'}")
    if rc != 0:
        run_fix_loop("ci", args.max_iter)
        status.append("Opened fix-loop for ci/unit failures")
        append_status(status)
        # Still try morning RC if merges land
        wait_for_prs()
        if not args.skip_morning_rc:
            run([str(SCRIPTS_DIR / "away-morning-rc.sh")])
        return rc

    # 2. User-flow real journeys (primary bug signal)
    if away_gate in ("user-flows", "full", ""):
        uf = run(["make", "layer-gate", "LAYER=user-flows"])
        status.append(f"`layer-gate LAYER=user-flows`: {'PASS' if uf == 0 else 'FAIL'}")
        if uf != 0:
            run_fix_loop("user-flows", args.max_iter)
            status.append("Ran layer-fix-loop for user-flows")

    # 3. Climb canary / full
    if away_gate == "full":
        climb = run(["make", "layer-climb", "CONTINUE=1"])
        status.append(f"`layer-climb CONTINUE=1`: {'PASS' if climb == 0 else 'FAIL'}")
        if climb != 0:
            run_fix_loop("implement", args.max_iter)
    else:
        for layer in ("implement", "chat"):
            if past_deadline(args.deadline_ct):
                status.append(f"Skipped remaining climb layers after deadline {args.deadline_ct} CT")
                break
            lr = run(["make", "layer-gate", f"LAYER={layer}"])
            status.append(f"`layer-gate LAYER={layer}`: {'PASS' if lr == 0 else 'FAIL'}")
            if lr != 0:
                run_fix_loop(layer, args.max_iter)

    # 4. Desktop UI E2E
    if not args.skip_desktop_e2e and not past_deadline(args.deadline_ct):
        e2e = run(["make", "desktop-e2e"])
        status.append(f"`make desktop-e2e`: {'PASS' if e2e == 0 else 'FAIL'}")
        if e2e != 0:
            # Treat as UI layer — agent fix via issue-style brief using layer-fix-loop chat as proxy
            run_fix_loop("chat", args.max_iter)
            status.append("desktop-e2e failed — ran chat fix-loop as proxy for UI wiring")

    # 5. Drain agent-ready issues
    issues_done = 0
    while issues_done < args.max_issues and not past_deadline(args.deadline_ct):
        if open_agent_pr_count() > 0:
            print("agent-pr in flight — waiting before next issue")
            wait_for_prs(timeout_s=1800)
            continue
        irc = run([PY, str(SCRIPTS_DIR / "away-agent-loop.py")])
        if irc != 0:
            status.append(f"away-agent-loop exit {irc}")
            break
        # away-agent-loop returns 0 also when no issues — detect by re-listing
        issues_done += 1
        # If no ready issues, loop exits quickly; break after one empty claim
        proc = subprocess.run(
            ["gh", "issue", "list", "--label", "agent-ready", "--state", "open", "--limit", "1", "--json", "number"],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        try:
            remaining = json.loads(proc.stdout or "[]")
        except json.JSONDecodeError:
            remaining = []
        if not remaining:
            status.append(f"Drained agent-ready (processed up to {issues_done} loop(s))")
            break

    wait_for_prs()
    append_status(status)

    if not args.skip_morning_rc:
        chmod = subprocess.run(["chmod", "+x", str(SCRIPTS_DIR / "away-morning-rc.sh")])
        _ = chmod
        mrc = run([str(SCRIPTS_DIR / "away-morning-rc.sh")])
        status.append(f"morning RC script exit {mrc}")
        append_status([status[-1]])

    print("Away overnight complete.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
