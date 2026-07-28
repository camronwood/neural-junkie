#!/usr/bin/env python3
"""Run a single release-prep layer gate and write a reviewable report."""

from __future__ import annotations

import argparse
import os
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

sys.path.insert(0, str(SCRIPTS_DIR))
from lib.release_prep_env import apply_release_prep_env, release_prep_env  # noqa: E402
from lib.regression_boot import ensure_ollama_stack, maybe_boot_regression  # noqa: E402
from lib.release_prep_layers import (  # noqa: E402
    LAYER_ORDER,
    get_layer,
    layer_report_paths,
    list_layers,
    resolve_stage_cmd,
)


def run_cmd(
    cmd: list[str],
    *,
    env: dict | None = None,
    cwd: Path = ROOT,
    timeout_s: float | None = None,
) -> tuple[int, str]:
    """Run a stage command, streaming stdout/stderr live and collecting a log buffer."""
    from lib.proc_timeout import kill_process_tree, wait_with_timeout

    merged = release_prep_env(ROOT)
    if env:
        merged.update(env)
    merged["PYTHONUNBUFFERED"] = "1"
    print(f"\n>>> {' '.join(cmd)}", flush=True)
    if timeout_s and timeout_s > 0:
        print(f"[layer-gate] stage timeout={int(timeout_s)}s", flush=True)
    proc = subprocess.Popen(
        cmd,
        cwd=cwd,
        env=merged,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
        start_new_session=True,
    )
    chunks: list[str] = []
    assert proc.stdout is not None

    def _drain() -> None:
        for line in proc.stdout:
            chunks.append(line)
            sys.stdout.write(line)
            sys.stdout.flush()

    reader = threading.Thread(target=_drain, name="layer-gate-drain", daemon=True)
    reader.start()
    rc, timed_out = wait_with_timeout(proc, timeout_s if timeout_s and timeout_s > 0 else None)
    reader.join(timeout=2)
    if timed_out:
        kill_process_tree(proc)
        chunks.append(f"\n[layer-gate] STAGE TIMEOUT after {int(timeout_s)}s — killed process tree\n")
        sys.stdout.write(chunks[-1])
        sys.stdout.flush()
        rc = 124
    out = "".join(chunks)
    if out and not out.endswith("\n"):
        sys.stdout.write("\n")
        sys.stdout.flush()
    return rc, out


def apply_verbose_to_stage_cmd(cmd: list[str], *, verbose: bool) -> tuple[list[str], dict[str, str]]:
    """Pass verbose to make targets via VERBOSE=1; python scripts get --verbose."""
    extra_env: dict[str, str] = {}
    if not verbose or "--verbose" in cmd:
        return cmd, extra_env
    if cmd and cmd[0] == "make":
        extra_env["VERBOSE"] = "1"
        return cmd, extra_env
    joined = " ".join(cmd)
    # These scripts either lack --verbose or already stream enough detail.
    if "implement-scenarios.py" in joined or "implement-scenarios-stable.py" in joined:
        return cmd, extra_env
    return [*cmd, "--verbose"], extra_env


def ensure_hub_for_layer(hub_url: str, *, no_restart: bool, layer: str = "", cwd: Path = ROOT) -> bool:
    repo = cwd.resolve()
    restart_layers = {"implement", "collab", "collab-core", "collab-full", "user-flows"}
    if layer in restart_layers and not no_restart:
        print("\n=== Layer gate Ollama prep ===")
        if not ensure_ollama_stack(repo):
            return False
        from lib.regression_boot import restart_hub_for_live_run

        label = f"layer-gate-{layer}"
        return restart_hub_for_live_run(repo, hub_url, label=label)
    return maybe_boot_regression(
        hub_url,
        root=repo,
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
    collab_layers = {"collab", "collab-core", "collab-full", "user-flows"}
    if spec.name in collab_layers:
        os.environ["NJ_REQUIRE_FULL_BOOT"] = "1"
        os.environ.pop("SKIP_BOOT", None)
        os.environ.pop("NJ_BOOT_DONE", None)
    if spec.name == "collab-core":
        os.environ["NJ_REGRESSION_SLIM_ROSTER"] = "1"
        # Core scenarios are mostly 2-agent planning; keep generation serial for VRAM.
        os.environ["NJ_OLLAMA_MAX_CONCURRENCY"] = "1"
        os.environ.pop("NJ_REGRESSION_COLLAB_EDGE", None)
    elif spec.name == "collab":
        os.environ["NJ_REGRESSION_SLIM_ROSTER"] = "1"
        # Edge suite runs website/FE+Security(+Claude) in parallel; concurrency 1
        # caused cascading generation timeouts and pending-task stalls.
        os.environ["NJ_OLLAMA_MAX_CONCURRENCY"] = "2"
        os.environ["NJ_REGRESSION_COLLAB_EDGE"] = "1"
    elif spec.name == "user-flows":
        # Mixed implement + collab product journeys; keep VRAM pressure low.
        os.environ["NJ_OLLAMA_MAX_CONCURRENCY"] = "1"
        # Greenfield implement needs headroom past the 512-token default.
        os.environ["NJ_OLLAMA_NUM_PREDICT"] = "4096"
        # Prefer coder for implement-heavy journeys (env.local may pin qwen3.5:9b).
        os.environ["NJ_REGRESSION_AGENT_MODEL"] = "qwen2.5-coder:14b"
        # User-flows need FrontendEngineer/BackendEngineer online — do not use collab-core slim roster.
        os.environ["NJ_REGRESSION_SLIM_ROSTER"] = "0"
        os.environ["NJ_REGRESSION_USER_FLOWS"] = "1"
        os.environ.pop("NJ_REGRESSION_COLLAB_EDGE", None)
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
    total_stages = len(spec.stages)

    for idx, stage in enumerate(spec.stages, 1):
        cmd = resolve_stage_cmd(stage.cmd, hub_url=hub_url)
        cmd, verbose_env = apply_verbose_to_stage_cmd(cmd, verbose=args.verbose)
        stage_env = {"NEURAL_JUNKIE_HUB_URL": hub_url, **verbose_env}
        print(
            f"\n=== layer-gate [{spec.name}] stage {idx}/{total_stages}: {stage.name} ===",
            flush=True,
        )
        log_lines.append(f"## {stage.name}")
        log_lines.append(f">>> {' '.join(cmd)}")
        log_lines.append("")
        t0 = time.time()
        # Stage budget: layer est * 1.5 / stages, min 10m (env NJ_LAYER_TIMEOUT_MULT).
        raw_mult = (os.environ.get("NJ_LAYER_TIMEOUT_MULT") or "1.5").strip()
        try:
            mult = float(raw_mult)
        except ValueError:
            mult = 1.5
        stage_timeout = None
        if mult > 0 and spec.est_minutes > 0:
            stage_timeout = max(600.0, (spec.est_minutes * 60.0 * mult) / max(1, total_stages))
        # user-flows journeys can exceed 3h wall (10 scenarios + flake retries).
        if spec.name == "user-flows" and stage_timeout is not None:
            stage_timeout = max(stage_timeout, 21600.0)
        if spec.name == "chat" and stage_timeout is not None:
            # Multi-turn DMs + flake retries routinely exceed the naive split budget.
            stage_timeout = max(stage_timeout, 7200.0)
        rc, out = run_cmd(cmd, env=stage_env, cwd=repo_cwd, timeout_s=stage_timeout)
        duration = time.time() - t0
        log_lines.append(out.rstrip())
        log_lines.append("")
        status = "OK" if rc == 0 else "FAIL"
        stage_rows.append((stage.name, status, duration, rc))
        log_lines.append(f"RESULT {stage.name}: {status} (exit {rc}, {duration:.0f}s)")
        log_lines.append("")
        print(
            f"=== layer-gate [{spec.name}] stage {idx}/{total_stages} {status}: "
            f"{stage.name} ({duration:.0f}s, exit {rc}) ===",
            flush=True,
        )
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
