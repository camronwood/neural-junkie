#!/usr/bin/env python3
"""Pure-Python CAD compile evaluation (Ollama generate + OpenSCAD), no curl."""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_OLLAMA = "http://127.0.0.1:11434"
AGGREGATE_SCENARIO = "cad-compile"

CANDIDATE_PROMPT_PATHS = [
    lambda: Path(os.environ["NJ_CAD_PROMPTS"]) if os.environ.get("NJ_CAD_PROMPTS") else None,
    lambda: (ROOT.parent / "neural-junkie-pack-cad" / "scenarios" / "model-eval" / "prompts.json"),
    lambda: Path(
        "/Users/camronwood/development/projects/neural-junkie-pack-cad/scenarios/model-eval/prompts.json"
    ),
]


def resolve_prompts_path() -> Path:
    for resolver in CANDIDATE_PROMPT_PATHS:
        try:
            path = resolver()
        except Exception:
            path = None
        if path is None:
            continue
        p = Path(path).expanduser()
        if p.is_file():
            return p
    raise FileNotFoundError(
        "CAD prompts.json not found (set NJ_CAD_PROMPTS or install neural-junkie-pack-cad)"
    )


def load_prompts(path: Path) -> list[dict[str, Any]]:
    data = json.loads(path.read_text(encoding="utf-8"))
    prompts = data.get("prompts") if isinstance(data, dict) else data
    if not isinstance(prompts, list):
        raise ValueError(f"invalid prompts in {path}")
    return [p for p in prompts if isinstance(p, dict) and (p.get("id") or p.get("prompt"))]


def filter_prompts(prompts: list[dict[str, Any]], scenario: str | None) -> list[dict[str, Any]]:
    if not scenario or scenario.strip().lower() in {"", "all", "*", AGGREGATE_SCENARIO}:
        return list(prompts)
    wanted = scenario.strip()
    selected = [p for p in prompts if str(p.get("id") or "").strip() == wanted]
    if not selected:
        raise ValueError(f"unknown prompt id {wanted!r}")
    return selected


def extract_scad(text: str) -> str:
    raw = (text or "").strip()
    if "```" in raw:
        parts = raw.split("```")
        for part in parts:
            chunk = part.strip()
            if chunk.lower().startswith("openscad"):
                chunk = chunk.split("\n", 1)[-1].strip() if "\n" in chunk else ""
            if "module" in chunk or "cube" in chunk or "=" in chunk or "cylinder" in chunk:
                return chunk.strip()
    return raw


def ollama_generate(ollama: str, model: str, prompt: str, *, timeout: float = 300.0) -> dict[str, Any]:
    url = f"{ollama.rstrip('/')}/api/generate"
    body = json.dumps(
        {
            "model": model,
            "prompt": f"Write only valid OpenSCAD code for: {prompt}",
            "stream": False,
            "options": {"temperature": 0.2},
        }
    ).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    return data if isinstance(data, dict) else {"response": str(data)}


def tokens_from_ollama(response: dict[str, Any]) -> tuple[int | None, int | None]:
    prompt = response.get("prompt_eval_count")
    completion = response.get("eval_count")
    try:
        prompt_n = int(prompt) if prompt is not None else None
    except (TypeError, ValueError):
        prompt_n = None
    try:
        completion_n = int(completion) if completion is not None else None
    except (TypeError, ValueError):
        completion_n = None
    return prompt_n, completion_n


def compile_scad(openscad_bin: str, scad_path: Path, stl_path: Path, log_path: Path) -> bool:
    try:
        with log_path.open("w", encoding="utf-8") as logf:
            proc = subprocess.run(
                [openscad_bin, "-o", str(stl_path), str(scad_path)],
                stdout=logf,
                stderr=subprocess.STDOUT,
                timeout=120,
                check=False,
            )
    except FileNotFoundError:
        return False
    except subprocess.TimeoutExpired:
        log_path.write_text("openscad timed out\n", encoding="utf-8")
        return False
    return proc.returncode == 0 and stl_path.is_file() and stl_path.stat().st_size > 0


def emit_metrics(metrics: dict[str, Any]) -> None:
    print("METRICS_JSON:" + json.dumps(metrics, separators=(",", ":")))


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--model", required=True, help="Ollama model tag")
    p.add_argument("--ollama", default=DEFAULT_OLLAMA, help="Ollama base URL")
    p.add_argument("--json-out", default="", help="Optional path to write summary JSON")
    p.add_argument(
        "--scenario",
        default=AGGREGATE_SCENARIO,
        help=f"Aggregate ({AGGREGATE_SCENARIO}) or a specific prompt id",
    )
    p.add_argument("--hub", default="", help="Unused (suite compatibility)")
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)

    openscad = shutil.which("openscad")
    if not openscad:
        print("FAIL: openscad not found")
        metrics = {
            "scenario": args.scenario or AGGREGATE_SCENARIO,
            "kind": "cad",
            "model": args.model,
            "prompt_tokens": None,
            "completion_tokens": None,
            "passed": 0,
            "failed": 0,
            "total": 0,
            "pass_rate": 0.0,
            "skipped": True,
            "skip_reason": "openscad not found",
            "capability_passed": False,
        }
        emit_metrics(metrics)
        if args.json_out:
            Path(args.json_out).write_text(json.dumps(metrics, indent=2) + "\n", encoding="utf-8")
        return 2

    try:
        prompts_path = resolve_prompts_path()
        prompts = filter_prompts(load_prompts(prompts_path), args.scenario)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: prompts ({exc})")
        emit_metrics(
            {
                "scenario": args.scenario or AGGREGATE_SCENARIO,
                "kind": "cad",
                "model": args.model,
                "prompt_tokens": None,
                "completion_tokens": None,
                "passed": 0,
                "failed": 0,
                "total": 0,
                "pass_rate": 0.0,
                "skipped": False,
                "capability_passed": False,
                "error": str(exc),
            }
        )
        return 1

    prompt_tokens_sum = 0
    completion_tokens_sum = 0
    have_prompt_tokens = False
    have_completion_tokens = False
    passed = 0
    failed = 0
    results: list[dict[str, Any]] = []

    with tempfile.TemporaryDirectory(prefix="nj-cad-benchmark-") as tmp:
        tmp_path = Path(tmp)
        for item in prompts:
            pid = str(item.get("id") or "prompt").strip()
            prompt_text = str(item.get("prompt") or "").strip()
            scad_path = tmp_path / f"{pid}.scad"
            stl_path = tmp_path / f"{pid}.stl"
            log_path = tmp_path / f"{pid}.log"
            ok = False
            detail = ""
            try:
                resp = ollama_generate(args.ollama, args.model, prompt_text)
                pt, ct = tokens_from_ollama(resp)
                if pt is not None:
                    prompt_tokens_sum += pt
                    have_prompt_tokens = True
                if ct is not None:
                    completion_tokens_sum += ct
                    have_completion_tokens = True
                scad = extract_scad(str(resp.get("response") or ""))
                if not scad.strip():
                    detail = "empty scad"
                else:
                    scad_path.write_text(scad, encoding="utf-8")
                    if compile_scad(openscad, scad_path, stl_path, log_path):
                        ok = True
                        detail = "compile ok"
                    else:
                        detail = f"openscad failed ({log_path.name})"
            except urllib.error.HTTPError as exc:
                detail = f"ollama HTTP {exc.code}"
            except urllib.error.URLError as exc:
                detail = f"ollama unavailable ({exc})"
            except Exception as exc:  # noqa: BLE001 — per-prompt isolation
                detail = str(exc)[:200]

            if ok:
                passed += 1
                print(f"PASS: {pid} {detail}")
            else:
                failed += 1
                print(f"FAIL: {pid} {detail}")
            results.append({"id": pid, "passed": ok, "detail": detail})

    total = passed + failed
    pass_rate = (passed / total) if total else 0.0
    # Aggregate scenario passes at >= 50%; single-prompt requires that prompt to pass.
    is_aggregate = (args.scenario or "").strip().lower() in {"", "all", "*", AGGREGATE_SCENARIO}
    if is_aggregate:
        capability_ok = pass_rate >= 0.5
    else:
        capability_ok = passed == total and total > 0

    metrics: dict[str, Any] = {
        "scenario": AGGREGATE_SCENARIO if is_aggregate else (args.scenario or AGGREGATE_SCENARIO),
        "kind": "cad",
        "model": args.model,
        "prompt_tokens": prompt_tokens_sum if have_prompt_tokens else None,
        "completion_tokens": completion_tokens_sum if have_completion_tokens else None,
        "passed": passed,
        "failed": failed,
        "total": total,
        "pass_rate": round(pass_rate, 4),
        "skipped": False,
        "capability_passed": capability_ok,
        "prompts_path": str(prompts_path),
        "results": results,
    }
    emit_metrics(metrics)
    if args.json_out:
        out = Path(args.json_out)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(json.dumps(metrics, indent=2) + "\n", encoding="utf-8")
        print(f"Wrote {out}")

    return 0 if capability_ok else 1


if __name__ == "__main__":
    sys.exit(main())
