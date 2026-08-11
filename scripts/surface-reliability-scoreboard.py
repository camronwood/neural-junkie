#!/usr/bin/env python3
"""Run stamp + memory + work-surface session gates and write a combined scoreboard."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TESTING = ROOT / "docs" / "testing"

STAMP_MIN_ACC = 0.90
STAMP_MAX_MISSTAMP = 0.05
MEM_MIN_HIT = 0.90
MEM_MAX_FORBIDDEN = 0.05


def run(cmd: list[str], env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    merged = os.environ.copy()
    if env:
        merged.update(env)
    print(">>>", " ".join(cmd), flush=True)
    return subprocess.run(
        cmd,
        cwd=ROOT,
        env=merged,
        text=True,
        capture_output=False,
    )


def load_json(path: Path) -> dict:
    if not path.is_file():
        return {}
    return json.loads(path.read_text())


def _num(payload: dict, key: str, default: float) -> float:
    if key not in payload or payload[key] is None:
        return default
    return float(payload[key])


def stamp_status(payload: dict) -> dict:
    acc = _num(payload, "action_accuracy", 0.0)
    miss = _num(payload, "misstamp_rate", 1.0)
    ok = acc >= STAMP_MIN_ACC and miss <= STAMP_MAX_MISSTAMP
    return {
        "name": "stamp",
        "ok": ok,
        "action_accuracy": acc,
        "misstamp_rate": miss,
        "bar": f"action>={STAMP_MIN_ACC} misstamp<={STAMP_MAX_MISSTAMP}",
    }


def memory_status(payload: dict) -> dict:
    hit = _num(payload, "hit_rate", 0.0)
    forbidden = _num(payload, "forbidden_hit_rate", 1.0)
    ok = hit >= MEM_MIN_HIT and forbidden <= MEM_MAX_FORBIDDEN
    return {
        "name": "memory",
        "ok": ok,
        "hit_rate": hit,
        "forbidden_hit_rate": forbidden,
        "bar": f"hit>={MEM_MIN_HIT} forbidden<={MEM_MAX_FORBIDDEN}",
    }


def session_status(log_text: str, proc: subprocess.CompletedProcess[str]) -> dict:
    evals: list[dict] = []
    for line in log_text.splitlines():
        if line.startswith("EVAL_JSON:"):
            try:
                evals.append(json.loads(line[len("EVAL_JSON:") :]))
            except json.JSONDecodeError:
                continue
    # Last EVAL_JSON per scenario wins (retries rewrite the banner).
    by_name: dict[str, dict] = {}
    for ev in evals:
        name = str(ev.get("scenario") or "").strip()
        if name:
            by_name[name] = ev
    passed = [n for n, ev in by_name.items() if ev.get("passed_at_1")]
    failed = [n for n, ev in by_name.items() if not ev.get("passed_at_1")]
    ok = bool(by_name) and not failed
    return {
        "name": "session",
        "ok": ok,
        "passed": sorted(passed),
        "failed": sorted(failed),
        "exit_code": proc.returncode,
        "bar": "all work-surface scenarios PASS @1",
    }


def write_report(stamp: str, gates: list[dict], artifacts: dict) -> tuple[Path, Path]:
    overall = all(g["ok"] for g in gates)
    payload = {
        "schema_version": 1,
        "kind": "surface_reliability",
        "stamp": stamp,
        "overall": "PASS" if overall else "FAIL",
        "gates": gates,
        "artifacts": artifacts,
    }
    json_path = TESTING / f"surface-reliability-{stamp}.json"
    md_path = TESTING / f"surface-reliability-{stamp}.md"
    json_path.write_text(json.dumps(payload, indent=2) + "\n")
    lines = [
        f"# Surface reliability — {stamp}",
        "",
        f"Overall: **{payload['overall']}**",
        "",
        "| Gate | Status | Detail |",
        "|------|--------|--------|",
    ]
    for g in gates:
        status = "PASS" if g["ok"] else "FAIL"
        if g["name"] == "stamp":
            detail = f"action={g['action_accuracy']:.3f} misstamp={g['misstamp_rate']:.3f}"
        elif g["name"] == "memory":
            detail = f"hit={g['hit_rate']:.3f} forbidden={g['forbidden_hit_rate']:.3f}"
        else:
            detail = f"pass={len(g.get('passed') or [])} fail={len(g.get('failed') or [])}"
        lines.append(f"| `{g['name']}` | {status} | {detail} ({g['bar']}) |")
    lines.extend(["", "Artifacts:", ""])
    for k, v in artifacts.items():
        lines.append(f"- {k}: `{v}`")
    lines.append("")
    md_path.write_text("\n".join(lines))
    return json_path, md_path


def main() -> int:
    p = argparse.ArgumentParser(description="Combined stamp/memory/work-surface scoreboard")
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--skip-stamp", action="store_true", help="Reuse latest semantic-eval JSON")
    p.add_argument("--skip-memory", action="store_true", help="Reuse latest memory-eval JSON")
    args = p.parse_args()

    TESTING.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    sem_out = TESTING / f"semantic-eval-{stamp}.json"
    mem_out = TESTING / f"memory-eval-{stamp}.json"
    session_log = TESTING / f"work-surface-{stamp}.log"

    if args.skip_stamp:
        prior = sorted(TESTING.glob("semantic-eval-*.json"))
        sem_out = prior[-1] if prior else sem_out
        sem = subprocess.CompletedProcess([], 0 if prior else 1)
        print(f">>> skip stamp, using {sem_out}", flush=True)
    else:
        sem = run(
            ["go", "test", "./cmd/server/", "-count=1", "-run", "TestLocalSemanticIntentEvaluation$", "-timeout", "30m", "-v"],
            env={
                "NJ_RUN_LOCAL_SEMANTIC_EVAL": "1",
                "NJ_SEMANTIC_EVAL_OUT": str(sem_out),
            },
        )
    if args.skip_memory:
        prior = sorted(TESTING.glob("memory-eval-*.json"))
        mem_out = prior[-1] if prior else mem_out
        mem = subprocess.CompletedProcess([], 0 if prior else 1)
        print(f">>> skip memory, using {mem_out}", flush=True)
    else:
        mem = run(
            ["go", "test", "./internal/memory/", "-count=1", "-run", "TestLiveMemoryRetrievalEvaluation$", "-timeout", "15m", "-v"],
            env={
                "NJ_RUN_LOCAL_MEMORY_EVAL": "1",
                "NJ_MEMORY_EVAL_OUT": str(mem_out),
            },
        )
    session_env = {
        "NEURAL_JUNKIE_RATE_LIMIT": "0",
        "PYTHONUNBUFFERED": "1",
    }
    session_cmd = [
        sys.executable,
        "-u",
        str(ROOT / "scripts" / "chat-scenarios.py"),
        "--hub",
        args.hub,
        "--all",
        "--tag",
        "work-surface",
        "-v",
    ]
    print(">>>", " ".join(session_cmd), flush=True)
    with session_log.open("w") as fh:
        sess = subprocess.run(session_cmd, cwd=ROOT, env={**os.environ, **session_env}, text=True, stdout=fh, stderr=subprocess.STDOUT)
    session_text = session_log.read_text() if session_log.is_file() else ""
    print(session_text[-4000:] if len(session_text) > 4000 else session_text)

    gates = [
        stamp_status(load_json(sem_out) if sem.returncode == 0 else {"action_accuracy": 0, "misstamp_rate": 1}),
        memory_status(load_json(mem_out) if mem.returncode == 0 else {"hit_rate": 0, "forbidden_hit_rate": 1}),
        session_status(session_text, sess),
    ]
    if sem.returncode != 0:
        gates[0]["ok"] = False
    if mem.returncode != 0:
        gates[1]["ok"] = False

    json_path, md_path = write_report(
        stamp,
        gates,
        {
            "semantic_eval": str(sem_out.relative_to(ROOT)),
            "memory_eval": str(mem_out.relative_to(ROOT)),
            "session_log": str(session_log.relative_to(ROOT)),
        },
    )
    print(f"Scoreboard JSON: {json_path}")
    print(f"Scoreboard MD:   {md_path}")
    overall = all(g["ok"] for g in gates)
    print("OVERALL:", "PASS" if overall else "FAIL")
    return 0 if overall else 1


if __name__ == "__main__":
    raise SystemExit(main())
