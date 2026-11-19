#!/usr/bin/env python3
"""Away overnight orchestrator: run real-user gates → morning bug report.

Report-only by default. Unattended pre-stable path:

  AWAY_GATE=full MORNING_RC=1 AWAY_FIX_ON_FAIL=1 AUTO_STABLE_CUT=1

Order (see docs/AWAY_OPERATIONS.md / docs/PRE_STABLE_RELIABILITY.md):
  1. make test-all
  2. make layer-gate LAYER=user-flows  (real user journeys)
  3. thin climb canary (implement + chat) unless AWAY_GATE=full → layer-climb
  4. make desktop-e2e (UI click journeys) unless SKIP_DESKTOP_E2E=1
  5. write away-morning bug report
  6. optional morning RC if MORNING_RC=1 / --morning-rc
  7. optional AWAY_FIX_ON_FAIL=1 → layer-fix-loop / away-agent on FAIL
  8. optional AUTO_STABLE_CUT=1 → scripts/auto-cut-stable-if-ready.sh when 2× green
"""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from zoneinfo import ZoneInfo

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
TESTING = ROOT / "docs" / "testing"
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
    if now.hour >= 12 and deadline.hour < 12:
        return False
    return now >= deadline


def _latest_layer_summaries(prefix: str, limit: int = 3) -> list[Path]:
    if not TESTING.is_dir():
        return []
    paths = sorted(TESTING.glob(f"{prefix}*.md"), key=lambda p: p.stat().st_mtime, reverse=True)
    return paths[:limit]


def _read_overall(path: Path) -> str:
    try:
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines()[:20]:
            if line.startswith("Overall:"):
                return line.strip()
    except OSError:
        pass
    return "(summary unreadable)"


def write_bug_report(
    *,
    stamp: str,
    gate_rows: list[tuple[str, int, str]],
    notes: list[str],
    deadline_ct: str,
) -> Path:
    """Write/replace the morning bug report for human triage."""
    path = TESTING / f"away-morning-{stamp}.md"
    TESTING.mkdir(parents=True, exist_ok=True)
    now = datetime.now(CT).strftime("%Y-%m-%d %H:%M %Z")

    fails = [(name, note) for name, rc, note in gate_rows if rc != 0]
    passes = [name for name, rc, _ in gate_rows if rc == 0]

    lines: list[str] = [
        f"# Away morning bug report — {stamp}",
        "",
        f"Generated: {now}  ",
        "Mode: **unattended ship bar** (agents/scripts own FAIL → PR → RC → stable)",
        "",
        "## Summary",
        "",
        f"- Gates run: {len(gate_rows)}",
        f"- PASS: {len(passes)}",
        f"- FAIL: {len(fails)}",
        f"- Deadline CT: {deadline_ct}",
        "",
        "## Gate results",
        "",
        "| Gate | Result | Notes |",
        "|------|--------|-------|",
    ]
    for name, rc, note in gate_rows:
        result = "PASS" if rc == 0 else "FAIL"
        lines.append(f"| `{name}` | **{result}** | {note or '—'} |")

    lines.extend(["", "## Act on these (FAIL)", ""])
    if not fails:
        lines.append("_No failing gates — optional: skim PASS notes and desktop traces._")
    else:
        for name, note in fails:
            lines.append(f"### `{name}`")
            lines.append("")
            lines.append(f"- {note or 'See latest layer-gate / desktop-e2e report under `docs/testing/`.'}")
            lines.append(
                "- Unattended: `AWAY_FIX_ON_FAIL=1` runs layer-fix-loop / away-agent; "
                "do not weaken scenarios."
            )
            lines.append("")

    lines.extend(
        [
            "## Latest artifacts",
            "",
        ]
    )
    for prefix, label in (
        ("layer-gate-user-flows-", "user-flows"),
        ("layer-gate-implement-", "implement"),
        ("layer-gate-chat-", "chat"),
        ("desktop-e2e-", "desktop-e2e"),
        ("layer-gate-ci-", "ci"),
    ):
        latest = _latest_layer_summaries(prefix, 1)
        if latest:
            p = latest[0]
            lines.append(f"- **{label}**: `{p.relative_to(ROOT)}` — {_read_overall(p)}")
        else:
            lines.append(f"- **{label}**: _(none this run)_")

    if notes:
        lines.extend(["", "## Run notes", ""])
        for n in notes:
            lines.append(f"- {n}")

    lines.extend(
        [
            "",
            "## Unattended follow-up",
            "",
            "1. FAIL → `AWAY_FIX_ON_FAIL` / daytime `make away-agent` (no human triage required).",
            "2. Open linked `docs/testing/layer-gate-*.md` / desktop-e2e reports only if debugging agents.",
            "3. `MORNING_RC=1` cuts RC when green + HEAD delta.",
            "4. `AUTO_STABLE_CUT=1` + 2 consecutive green mornings → `scripts/auto-cut-stable-if-ready.sh`.",
            "",
        ]
    )
    path.write_text("\n".join(lines), encoding="utf-8")
    print(f"\nMorning bug report: {path}", flush=True)
    return path


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--deadline-ct",
        default=os.environ.get("AWAY_DEADLINE_CT") or "05:00",
        help="Stop starting new gates after this America/Chicago clock time",
    )
    ap.add_argument(
        "--skip-desktop-e2e",
        action="store_true",
        default=os.environ.get("SKIP_DESKTOP_E2E") == "1",
    )
    # Report-only default: no morning RC unless explicitly requested.
    morning_rc_env = (os.environ.get("MORNING_RC") or "").strip() == "1"
    skip_rc_env = (os.environ.get("SKIP_MORNING_RC") or "").strip() == "1"
    ap.add_argument(
        "--morning-rc",
        action="store_true",
        default=morning_rc_env and not skip_rc_env,
        help="Also cut morning RC (off by default in report-only mode)",
    )
    ap.add_argument(
        "--skip-morning-rc",
        action="store_true",
        default=False,
        help="Force-skip morning RC even if --morning-rc / MORNING_RC=1",
    )
    args = ap.parse_args()
    do_morning_rc = args.morning_rc and not args.skip_morning_rc and not skip_rc_env

    apply_release_prep_env(ROOT)
    stamp = datetime.now(CT).strftime("%Y-%m-%d")
    gate_rows: list[tuple[str, int, str]] = []
    fix_on_fail = (os.environ.get("AWAY_FIX_ON_FAIL") or "").strip() == "1"
    auto_stable = (os.environ.get("AUTO_STABLE_CUT") or "").strip() == "1"
    notes: list[str] = [
        "Away overnight ship bar: gates ran"
        + ("; AWAY_FIX_ON_FAIL=1" if fix_on_fail else "; report-only fix path")
        + ("; AUTO_STABLE_CUT=1" if auto_stable else "")
        + ".",
    ]
    away_gate = (os.environ.get("AWAY_GATE") or "user-flows").strip().lower()
    worst = 0

    def record(name: str, rc: int, note: str = "") -> None:
        nonlocal worst
        gate_rows.append((name, rc, note))
        if rc != 0:
            worst = rc if worst == 0 else worst

    # 1. Unit gate (record FAIL but continue — do not abort the hunt)
    rc = run(["make", "test-all"])
    record("make test-all", rc)
    if rc != 0:
        notes.append(
            "test-all failed — recorded for triage; continuing live gates (user-flows / climb / desktop-e2e)."
        )

    # 2. User-flow real journeys (primary bug signal)
    if away_gate in ("user-flows", "full", ""):
        if past_deadline(args.deadline_ct):
            notes.append(f"Skipped user-flows after deadline {args.deadline_ct} CT")
        else:
            uf = run(["make", "layer-gate", "LAYER=user-flows"])
            latest = _latest_layer_summaries("layer-gate-user-flows-", 1)
            note = _read_overall(latest[0]) if latest else ""
            record("layer-gate LAYER=user-flows", uf, note)

    # 3. Climb canary / full
    if away_gate == "full":
        if past_deadline(args.deadline_ct):
            notes.append(f"Skipped layer-climb after deadline {args.deadline_ct} CT")
        else:
            climb = run(["make", "layer-climb", "CONTINUE=1"])
            record("layer-climb CONTINUE=1", climb)
    else:
        for layer in ("implement", "chat"):
            if past_deadline(args.deadline_ct):
                notes.append(f"Skipped remaining climb layers after deadline {args.deadline_ct} CT")
                break
            lr = run(["make", "layer-gate", f"LAYER={layer}"])
            latest = _latest_layer_summaries(f"layer-gate-{layer}-", 1)
            note = _read_overall(latest[0]) if latest else ""
            record(f"layer-gate LAYER={layer}", lr, note)

    # 4. Desktop UI E2E
    if not args.skip_desktop_e2e and not past_deadline(args.deadline_ct):
        e2e = run(["make", "desktop-e2e"])
        latest = _latest_layer_summaries("desktop-e2e-", 1)
        note = str(latest[0].relative_to(ROOT)) if latest else ""
        record("make desktop-e2e", e2e, note)
    elif args.skip_desktop_e2e:
        notes.append("SKIP_DESKTOP_E2E=1 — desktop UI clicks not run")

    write_bug_report(
        stamp=stamp,
        gate_rows=gate_rows,
        notes=notes,
        deadline_ct=args.deadline_ct,
    )

    # Unattended fix path: first failing layer → layer-fix-loop; also drain agent-ready.
    if fix_on_fail and worst != 0:
        fail_names = [name for name, rc, _ in gate_rows if rc != 0]
        notes.append(f"AWAY_FIX_ON_FAIL: failing gates={fail_names}")
        layer = "implement"
        for name in fail_names:
            if "user-flows" in name:
                layer = "user-flows"
                break
            if "chat" in name:
                layer = "chat"
                break
            if "collab" in name:
                layer = "collab"
                break
            if "implement" in name or "climb" in name:
                layer = "implement"
                break
            if "desktop-e2e" in name or "test-all" in name:
                layer = "ci"
                break
        notes.append(f"Starting layer-fix-loop LAYER={layer} MAX_ITER=2")
        write_bug_report(
            stamp=stamp,
            gate_rows=gate_rows,
            notes=notes,
            deadline_ct=args.deadline_ct,
        )
        fl = run(
            [
                "make",
                "layer-fix-loop",
                f"LAYER={layer}",
                "MAX_ITER=2",
            ]
        )
        notes.append(f"layer-fix-loop exit {fl}")
        ag = run(["make", "away-agent"])
        notes.append(f"away-agent exit {ag}")
        write_bug_report(
            stamp=stamp,
            gate_rows=gate_rows,
            notes=notes,
            deadline_ct=args.deadline_ct,
        )

    if do_morning_rc and worst == 0:
        subprocess.run(["chmod", "+x", str(SCRIPTS_DIR / "away-morning-rc.sh")], check=False)
        mrc = run([str(SCRIPTS_DIR / "away-morning-rc.sh")])
        notes.append(f"morning RC script exit {mrc}")
        write_bug_report(
            stamp=stamp,
            gate_rows=gate_rows,
            notes=notes,
            deadline_ct=args.deadline_ct,
        )
    elif do_morning_rc and worst != 0:
        notes.append("Skipped morning RC — ship bar not all green.")
        print("Skipping morning RC (gates failed).", flush=True)
        write_bug_report(
            stamp=stamp,
            gate_rows=gate_rows,
            notes=notes,
            deadline_ct=args.deadline_ct,
        )
    else:
        print("Skipping morning RC (set MORNING_RC=1 to enable).", flush=True)

    if auto_stable and worst == 0:
        subprocess.run(
            ["chmod", "+x", str(SCRIPTS_DIR / "auto-cut-stable-if-ready.sh")],
            check=False,
        )
        cut = run([str(SCRIPTS_DIR / "auto-cut-stable-if-ready.sh"), "--execute"])
        notes.append(f"auto-cut-stable-if-ready exit {cut}")
        write_bug_report(
            stamp=stamp,
            gate_rows=gate_rows,
            notes=notes,
            deadline_ct=args.deadline_ct,
        )

    print("Away overnight complete.")
    return 0 if worst == 0 else worst


if __name__ == "__main__":
    raise SystemExit(main())
