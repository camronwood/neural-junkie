#!/usr/bin/env python3
"""Run release-prep layers in order (layer-climb).

Default: stop on first failure.
With --continue-on-fail / CONTINUE=1: run every layer and write a rollup report.

Live progress:
  - scoreboard banners between layers
  - stage output streamed from layer-gate (no longer buffered)
  - docs/testing/layer-climb-status.txt updated frequently (tail -f friendly)
  - heartbeats while a layer is still running
"""

from __future__ import annotations

import argparse
import os
import signal
import subprocess
import sys
import threading
import time
from datetime import datetime, timezone
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
DEFAULT_TESTING_DIR = ROOT / "docs" / "testing"
PY = sys.executable
STATUS_NAME = "layer-climb-status.txt"
HEARTBEAT_SECS = 60
# Hard wall clock per layer = est_minutes * multiplier (override via NJ_LAYER_TIMEOUT_MULT).
DEFAULT_LAYER_TIMEOUT_MULT = 1.5
LAYER_TIMEOUT_EXIT = 124

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.proc_timeout import wait_with_timeout  # noqa: E402
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402
from lib.release_prep_layers import LAYER_ORDER, get_layer, list_layers  # noqa: E402
from lib.regression_models import (  # noqa: E402
    DEFAULT_REGRESSION_AGENT_MODEL,
    resolve_regression_agent_model,
)


def _fmt_dur(seconds: float) -> str:
    s = int(max(0, seconds))
    h, rem = divmod(s, 3600)
    m, sec = divmod(rem, 60)
    if h:
        return f"{h}h{m:02d}m"
    if m:
        return f"{m}m{sec:02d}s"
    return f"{sec}s"


def write_status(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text.rstrip() + "\n", encoding="utf-8")


def scoreboard_lines(
    *,
    stamp: str,
    hub_url: str,
    continue_on_fail: bool,
    rows: list[tuple[str, str, float, int]],
    current: str | None,
    current_idx: int,
    total: int,
    layer_elapsed: float | None = None,
    note: str = "",
) -> str:
    ok = sum(1 for _, status, _, _ in rows if status == "PASS")
    fail = sum(1 for _, status, _, _ in rows if status == "FAIL")
    done = len(rows)
    remaining = [n for n in LAYER_ORDER[done:] if n != current]
    lines = [
        f"layer-climb status — {stamp}",
        f"updated={datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M:%S')}Z",
        f"hub={hub_url}",
        f"continue_on_fail={str(continue_on_fail).lower()}",
        f"progress={done}/{total} layers finished · PASS={ok} FAIL={fail}",
    ]
    if current:
        est = get_layer(current).est_minutes
        elapsed = f" elapsed={_fmt_dur(layer_elapsed)}" if layer_elapsed is not None else ""
        lines.append(
            f"CURRENT [{current_idx}/{total}] layer={current} (~{est}m est){elapsed}"
        )
    else:
        lines.append("CURRENT (idle / finished)")
    if remaining:
        lines.append(f"remaining={', '.join(remaining)}")
    else:
        lines.append("remaining=(none)")
    if note:
        lines.append(f"note={note}")
    lines.append("")
    lines.append("scoreboard:")
    for name, status, duration, rc in rows:
        lines.append(f"  {status:4}  {name:12}  {_fmt_dur(duration):>8}  exit={rc}")
    if current and current not in {r[0] for r in rows}:
        age = _fmt_dur(layer_elapsed or 0)
        lines.append(f"  RUN   {current:12}  {age:>8}  …")
    return "\n".join(lines)


def print_banner(text: str) -> None:
    bar = "=" * 72
    print(f"\n{bar}\n{text}\n{bar}", flush=True)


def layer_timeout_seconds(layer: str) -> float:
    """Hard wall-clock budget for one layer (est * mult, min 15m)."""
    raw = (os.environ.get("NJ_LAYER_TIMEOUT_MULT") or "").strip()
    try:
        mult = float(raw) if raw else DEFAULT_LAYER_TIMEOUT_MULT
    except ValueError:
        mult = DEFAULT_LAYER_TIMEOUT_MULT
    if mult <= 0:
        return 0.0  # disabled
    est = max(1, int(get_layer(layer).est_minutes))
    return max(15 * 60.0, est * 60.0 * mult)


def run_layer_gate(
    layer: str,
    *,
    hub_url: str,
    verbose: bool,
    no_restart_hub: bool,
    status_path: Path,
    status_fn,
) -> int:
    cmd = [
        PY,
        str(SCRIPTS_DIR / "layer-gate.py"),
        "--layer",
        layer,
        "--hub",
        hub_url,
    ]
    if verbose:
        cmd.append("--verbose")
    if no_restart_hub:
        cmd.append("--no-restart-hub")
    env = release_prep_env(ROOT)
    env["PYTHONUNBUFFERED"] = "1"
    env["NEURAL_JUNKIE_RATE_LIMIT"] = "0"
    env["NEURAL_JUNKIE_HUB_URL"] = hub_url
    # Lock the climb to one agent model for the whole run.
    env.setdefault("NJ_REGRESSION_AGENT_MODEL", resolve_regression_agent_model(ROOT, env))
    env["OLLAMA_CODE_MODEL"] = env["NJ_REGRESSION_AGENT_MODEL"]
    env["NJ_REGRESSION_LOCK_MODEL"] = "1"

    timeout_s = layer_timeout_seconds(layer)
    print(f">>> {' '.join(cmd)}", flush=True)
    if timeout_s > 0:
        print(
            f"[layer-climb] hard timeout for {layer}: {_fmt_dur(timeout_s)} "
            f"(est={get_layer(layer).est_minutes}m × "
            f"{os.environ.get('NJ_LAYER_TIMEOUT_MULT') or DEFAULT_LAYER_TIMEOUT_MULT})",
            flush=True,
        )
    proc = subprocess.Popen(cmd, cwd=ROOT, env=env, start_new_session=True)
    stop = threading.Event()

    def heartbeat() -> None:
        while not stop.wait(HEARTBEAT_SECS):
            write_status(status_path, status_fn(note="heartbeat — layer still running"))
            print(
                f"[layer-climb heartbeat] still on {layer} · "
                f"status file: {status_path}",
                flush=True,
            )

    hb = threading.Thread(target=heartbeat, name=f"climb-hb-{layer}", daemon=True)
    hb.start()
    try:
        rc, timed_out = wait_with_timeout(proc, timeout_s if timeout_s > 0 else None)
        if timed_out:
            print(
                f"[layer-climb] TIMEOUT killed layer={layer} after {_fmt_dur(timeout_s)}",
                flush=True,
            )
            write_status(
                status_path,
                status_fn(note=f"TIMEOUT — killed {layer} after {_fmt_dur(timeout_s)}"),
            )
            return LAYER_TIMEOUT_EXIT
        return int(rc)
    finally:
        stop.set()
        hb.join(timeout=1)


def latest_layer_report(testing_dir: Path, layer: str) -> Path | None:
    matches = sorted(
        testing_dir.glob(f"layer-gate-{layer}-*.md"),
        key=lambda p: p.stat().st_mtime,
        reverse=True,
    )
    return matches[0] if matches else None


def write_rollup(
    path: Path,
    *,
    stamp: str,
    hub_url: str,
    continue_on_fail: bool,
    rows: list[tuple[str, str, float, int, Path | None]],
    aborted: str = "",
    pinned_model: str = "",
) -> None:
    ok = sum(1 for _, status, _, _, _ in rows if status == "PASS")
    failed = [name for name, status, _, _, _ in rows if status not in ("PASS",)]
    if aborted:
        overall = "ABORTED"
    elif not failed and len(rows) == len(LAYER_ORDER):
        overall = "PASS"
    elif not failed:
        overall = "PARTIAL"
    else:
        overall = "FAIL"
    lines = [
        f"# Layer climb — {stamp} UTC",
        "",
        f"hub={hub_url}",
        f"continue_on_fail={str(continue_on_fail).lower()}",
        f"Overall: **{overall}** ({ok}/{len(rows)} layers finished)",
    ]
    if pinned_model:
        lines.append(f"pinned_model={pinned_model}")
    if aborted:
        lines.append(f"aborted={aborted}")
    lines.extend(
        [
            "",
            "## Layer summary",
            "",
            "| Layer | Status | Duration | Exit | Report |",
            "|-------|--------|----------|------|--------|",
        ]
    )
    for name, status, duration, rc, report in rows:
        link = f"`{report.name}`" if report else "—"
        lines.append(f"| `{name}` | {status} | {_fmt_dur(duration)} | {rc} | {link} |")
    lines.extend(["", "## Child reports", ""])
    for name, status, _, _, report in rows:
        if report:
            lines.append(f"- `{name}` ({status}): `{report}`")
        else:
            lines.append(f"- `{name}` ({status}): (no report found)")
    if failed:
        lines.extend(["", "## Failed layers", ""])
        for name in failed:
            lines.append(f"- `{name}`")
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--stamp", help="UTC stamp for rollup filename (default: now)")
    p.add_argument(
        "--continue-on-fail",
        action="store_true",
        help="Run all layers even when one fails; write rollup report",
    )
    p.add_argument("--no-restart-hub", action="store_true")
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--list", action="store_true", help="List layers and exit")
    args = p.parse_args()

    if args.list:
        for spec in list_layers():
            nxt = f" → {spec.next_layer}" if spec.next_layer else ""
            print(f"{spec.name:12} ~{spec.est_minutes:3}m  hub={spec.requires_hub}  {spec.description}{nxt}")
        return 0

    apply_release_prep_env(ROOT)
    # Detach from the launcher process group (IDE/shell teardown often SIGKILLs the
    # group). layer-gate already uses start_new_session; without this, climb dies
    # mid-ci while the orphaned gate finishes alone.
    try:
        os.setsid()
    except OSError:
        pass
    try:
        signal.signal(signal.SIGHUP, signal.SIG_IGN)
    except (AttributeError, ValueError, OSError):
        pass
    # Pin one agent model for the whole climb unless already set.
    pinned = (os.environ.get("NJ_REGRESSION_AGENT_MODEL") or "").strip()
    if not pinned:
        pinned = resolve_regression_agent_model(ROOT) or DEFAULT_REGRESSION_AGENT_MODEL
        os.environ["NJ_REGRESSION_AGENT_MODEL"] = pinned
    os.environ["OLLAMA_CODE_MODEL"] = pinned
    os.environ["NJ_REGRESSION_LOCK_MODEL"] = "1"

    hub_url = args.hub.rstrip("/")
    testing_dir = Path(args.log_dir)
    testing_dir.mkdir(parents=True, exist_ok=True)
    stamp = args.stamp or datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    continue_on_fail = args.continue_on_fail
    status_path = testing_dir / STATUS_NAME
    climb_t0 = time.time()
    total = len(LAYER_ORDER)

    finished: list[tuple[str, str, float, int]] = []
    report_rows: list[tuple[str, str, float, int, Path | None]] = []
    overall_rc = 0
    current_layer = {"name": None, "idx": 0, "t0": climb_t0}
    abort_reason = {"value": ""}

    def status_snapshot(note: str = "") -> str:
        elapsed = time.time() - current_layer["t0"] if current_layer["name"] else None
        return scoreboard_lines(
            stamp=stamp,
            hub_url=hub_url,
            continue_on_fail=continue_on_fail,
            rows=finished,
            current=current_layer["name"],
            current_idx=int(current_layer["idx"]),
            total=total,
            layer_elapsed=elapsed,
            note=note,
        )

    def finalize(aborted: str = "") -> Path:
        rollup = testing_dir / f"layer-climb-{stamp}.md"
        write_rollup(
            rollup,
            stamp=stamp,
            hub_url=hub_url,
            continue_on_fail=continue_on_fail,
            rows=report_rows,
            aborted=aborted,
            pinned_model=pinned,
        )
        note = (
            f"ABORTED ({aborted}); rollup={rollup}"
            if aborted
            else f"done overall={'PASS' if overall_rc == 0 else 'FAIL'}; rollup={rollup}"
        )
        write_status(status_path, status_snapshot(note=note))
        return rollup

    def on_signal(signum: int, _frame) -> None:
        name = signal.Signals(signum).name if hasattr(signal, "Signals") else str(signum)
        abort_reason["value"] = f"signal {name}"
        write_status(status_path, status_snapshot(note=f"ABORTED — {abort_reason['value']}"))
        raise SystemExit(130 if signum == signal.SIGINT else 143)

    # Survive launcher shell exit (nohup/make/IDE) so multi-hour climbs are not orphaned
    # after the first layer while layer-gate (start_new_session) keeps running alone.
    try:
        signal.signal(signal.SIGHUP, signal.SIG_IGN)
    except (AttributeError, ValueError, OSError):
        pass
    signal.signal(signal.SIGINT, on_signal)
    signal.signal(signal.SIGTERM, on_signal)

    print_banner(
        f"layer-climb START · {total} layers · continue_on_fail={continue_on_fail}\n"
        f"pinned_model={pinned}\n"
        f"live status: {status_path}  (tail -f {status_path})"
    )
    write_status(status_path, status_snapshot(note=f"starting · pinned_model={pinned}"))

    try:
        for idx, layer in enumerate(LAYER_ORDER, 1):
            remaining = LAYER_ORDER[idx:]
            rem = ", ".join(remaining) if remaining else "(none)"
            est = get_layer(layer).est_minutes
            print_banner(
                f"[{idx}/{total}] START layer={layer} (~{est}m est)\n"
                f"PASS so far={sum(1 for _, s, _, _ in finished if s == 'PASS')}  "
                f"FAIL so far={sum(1 for _, s, _, _ in finished if s == 'FAIL')}\n"
                f"remaining after this: {rem}\n"
                f"climb elapsed={_fmt_dur(time.time() - climb_t0)}"
            )
            current_layer.update(name=layer, idx=idx, t0=time.time())
            write_status(status_path, status_snapshot(note=f"starting {layer}"))

            t0 = time.time()
            rc = run_layer_gate(
                layer,
                hub_url=hub_url,
                verbose=args.verbose,
                no_restart_hub=args.no_restart_hub,
                status_path=status_path,
                status_fn=status_snapshot,
            )
            duration = time.time() - t0
            if rc == LAYER_TIMEOUT_EXIT:
                status = "TIMEOUT"
            else:
                status = "PASS" if rc == 0 else "FAIL"
            finished.append((layer, status, duration, rc))
            report = latest_layer_report(testing_dir, layer)
            report_rows.append((layer, status, duration, rc, report))
            current_layer.update(name=None)

            print_banner(
                f"[{idx}/{total}] {status} layer={layer} ({_fmt_dur(duration)}, exit {rc})\n"
                f"scoreboard: "
                + " · ".join(f"{n}={s}" for n, s, _, _ in finished)
                + f"\nclimb elapsed={_fmt_dur(time.time() - climb_t0)}"
            )
            write_status(status_path, status_snapshot(note=f"finished {layer}={status}"))

            if rc != 0:
                overall_rc = rc if overall_rc == 0 else overall_rc
                if not continue_on_fail:
                    print(
                        "Stopping at first failure (re-run with CONTINUE=1 for full scoreboard).",
                        flush=True,
                    )
                    break
    except SystemExit as exc:
        rollup = finalize(aborted=abort_reason["value"] or "SystemExit")
        print_banner(
            f"layer-climb ABORTED · {abort_reason['value'] or 'SystemExit'}\n"
            f"Partial rollup: {rollup}\n"
            f"Status: {status_path}"
        )
        raise exc

    rollup = finalize()
    ok = sum(1 for _, s, _, _, _ in report_rows if s == "PASS")
    print_banner(
        f"layer-climb DONE · {'PASS' if overall_rc == 0 else 'FAIL'} "
        f"({ok}/{len(report_rows)} layers · total {_fmt_dur(time.time() - climb_t0)})\n"
        f"Rollup: {rollup}\n"
        f"Status: {status_path}"
    )
    if overall_rc == 0 and len(report_rows) == len(LAYER_ORDER):
        print("=== layer-climb: all layers PASS ===", flush=True)
    return 0 if overall_rc == 0 else (overall_rc if overall_rc else 1)


if __name__ == "__main__":
    raise SystemExit(main())
