#!/usr/bin/env python3
"""Run a single release-prep layer gate and write a reviewable report."""

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
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402
from lib.regression_boot import maybe_boot_regression  # noqa: E402
from lib.release_prep_layers import (  # noqa: E402
    LAYER_ORDER,
    get_layer,
    layer_report_paths,
    list_layers,
    resolve_stage_cmd,
)


def run_cmd(cmd: list[str], *, env: dict | None = None, cwd: Path = ROOT) -> tuple[int, str]:
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


def ensure_hub_for_layer(hub_url: str, *, no_restart: bool, layer: str = "", cwd: Path = ROOT) -> bool:
    restart_layers = {"implement", "collab", "collab-core", "collab-full"}
    if layer in restart_layers and not no_restart:
        from lib.regression_boot import restart_hub_for_live_run

        label = f"layer-gate-{layer}"
        return restart_hub_for_live_run(cwd.resolve(), hub_url, label=label)
    return maybe_boot_regression(
        hub_url,
        root=cwd.resolve(),
        label="layer-gate",
        no_restart_hub=no_restart,
    )


def write_report(
    path: Path,
    *,
    layer: str,
    hub_url: str,
    stamp: str,
    stage_rows: list[tuple[str, str, float, int]],
    log_path: Path,
    overall_rc: int,
) -> None:
    ok_count = sum(1 for _, status, _, _ in stage_rows if status == "OK")
    total = len(stage_rows)
    overall = "PASS" if overall_rc == 0 else "FAIL"
    lines = [
        f"# Layer gate — {layer} — {stamp} UTC",
        "",
        f"layer={layer}",
        f"hub={hub_url}",
        f"Overall: **{overall}** ({ok_count}/{total} stages)",
        "",
        "## Stage summary",
        "",
        "| Stage | Status | Duration | Exit |",
        "|-------|--------|----------|------|",
    ]
    for name, status, duration, rc in stage_rows:
        lines.append(f"| `{name}` | {status} | {duration:.0f}s | {rc} |")
    lines.extend(
        [
            "",
            "## Child artifacts",
            "",
            f"- `{log_path}`",
            "",
            "## Failures (tail)",
            "",
        ]
    )
    for name, status, _, rc in stage_rows:
        if status != "FAIL":
            continue
        tail = ""
        if log_path.is_file():
            full = log_path.read_text(encoding="utf-8", errors="replace")
            marker = f">>> [{'{'}]{name}[{'}'}]" if False else f"## {name}"
            # Extract stage section from combined log
            parts = full.split(f"\n## {name}\n")
            if len(parts) > 1:
                tail = parts[1].split("\n## ", 1)[0]
            else:
                idx = full.find(f">>> ")
                tail = full[idx:] if idx >= 0 else full
        if len(tail) > 12000:
            tail = tail[-12000:]
        lines.extend([f"### {name} (exit {rc})", "", "```text", tail.rstrip(), "```", ""])
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--layer", required=True, help=f"Layer name: {', '.join(LAYER_ORDER)}")
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--log-dir", default=str(DEFAULT_TESTING_DIR))
    p.add_argument("--stamp", help="UTC stamp for report filenames (default: now)")
    p.add_argument("--no-restart-hub", action="store_true", help="Use healthy hub; skip restart")
    p.add_argument("--cwd", type=Path, default=ROOT, help="Repo root for hub boot and stage commands")
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--list", action="store_true", help="List layers and exit")
    args = p.parse_args()

    if args.list:
        for spec in list_layers():
            nxt = f" → {spec.next_layer}" if spec.next_layer else ""
            print(f"{spec.name:12} ~{spec.est_minutes:3}m  hub={spec.requires_hub}  {spec.description}{nxt}")
        return 0

    try:
        spec = get_layer(args.layer)
    except ValueError as err:
        print(err, file=sys.stderr)
        return 2

    apply_release_prep_env(ROOT)
    hub_url = args.hub.rstrip("/")
    repo_cwd = args.cwd.resolve()
    collab_layers = {"collab", "collab-core", "collab-full"}
    if spec.name in collab_layers:
        os.environ["NJ_REQUIRE_FULL_BOOT"] = "1"
        os.environ.pop("SKIP_BOOT", None)
        os.environ.pop("NJ_BOOT_DONE", None)
    if spec.name in ("collab", "collab-core"):
        os.environ["NJ_REGRESSION_SLIM_ROSTER"] = "1"
        os.environ["NJ_REGRESSION_CLAUDE_CLOUD"] = "1"
        os.environ["NJ_OLLAMA_MAX_CONCURRENCY"] = "1"
    elif spec.name == "collab-full":
        os.environ["NJ_REGRESSION_CLAUDE_CLOUD"] = "1"
    testing_dir = Path(args.log_dir)
    testing_dir.mkdir(parents=True, exist_ok=True)
    stamp = args.stamp or datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    summary_path, log_path = layer_report_paths(testing_dir, spec.name, stamp)

    if spec.requires_hub and not ensure_hub_for_layer(
        hub_url, no_restart=args.no_restart_hub, layer=spec.name, cwd=repo_cwd
    ):
        return 1

    log_lines: list[str] = [
        f"# layer gate log — {spec.name} — {stamp} UTC",
        f"hub={hub_url}",
        "",
    ]
    stage_rows: list[tuple[str, str, float, int]] = []
    overall_rc = 0

    for stage in spec.stages:
        cmd = resolve_stage_cmd(stage.cmd, hub_url=hub_url)
        if args.verbose and "--verbose" not in cmd and "implement-scenarios.py" not in " ".join(cmd):
            cmd = [*cmd, "--verbose"]
        log_lines.append(f"## {stage.name}")
        log_lines.append(f">>> {' '.join(cmd)}")
        log_lines.append("")
        t0 = time.time()
        rc, out = run_cmd(cmd, env={"NEURAL_JUNKIE_HUB_URL": hub_url}, cwd=repo_cwd)
        duration = time.time() - t0
        log_lines.append(out.rstrip())
        log_lines.append("")
        status = "OK" if rc == 0 else "FAIL"
        stage_rows.append((stage.name, status, duration, rc))
        log_lines.append(f"RESULT {stage.name}: {status} (exit {rc}, {duration:.0f}s)")
        log_lines.append("")
        if rc != 0:
            overall_rc = rc

    log_path.write_text("\n".join(log_lines) + "\n", encoding="utf-8")
    write_report(
        summary_path,
        layer=spec.name,
        hub_url=hub_url,
        stamp=stamp,
        stage_rows=stage_rows,
        log_path=log_path,
        overall_rc=overall_rc,
    )

    ok_count = sum(1 for _, s, _, _ in stage_rows if s == "OK")
    print(f"\n=== Layer {spec.name}: {'PASS' if overall_rc == 0 else 'FAIL'} ({ok_count}/{len(stage_rows)} stages) ===")
    print(f"Summary: {summary_path}")
    print(f"Log:     {log_path}")
    if spec.next_layer and overall_rc == 0:
        print(f"Next layer: {spec.next_layer}  (make layer-gate LAYER={spec.next_layer})")
    return overall_rc


if __name__ == "__main__":
    raise SystemExit(main())
