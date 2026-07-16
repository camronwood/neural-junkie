#!/usr/bin/env python3
"""External HumanEval-25 calibration runner (Ollama generate + subprocess tests)."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
from pathlib import Path
from typing import Any

SCRIPTS_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.humaneval_runner import (  # noqa: E402
    coding_prompt,
    emit_metrics_line,
    extract_python_code,
    filter_problems,
    load_problems,
    ollama_generate,
    run_harness,
    tokens_from_ollama,
)

DEFAULT_OLLAMA = "http://127.0.0.1:11434"
FULL_SET_FLOOR = 0.2


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--model", required=True, help="Ollama model tag")
    p.add_argument("--ollama", default=DEFAULT_OLLAMA, help="Ollama base URL")
    p.add_argument(
        "--scenario",
        default="all",
        help="all (default) or a problem id / comma-separated ids",
    )
    p.add_argument("--hub", default="", help="Unused (suite compatibility)")
    p.add_argument(
        "--problems",
        default="",
        help="Optional path to humaneval JSON (default: scripts/config/humaneval-25.json)",
    )
    p.add_argument("--timeout", type=float, default=5.0, help="Per-problem harness timeout seconds")
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    problems_path = Path(args.problems) if args.problems else None
    try:
        license_note, problems = load_problems(problems_path)
        selected = filter_problems(problems, args.scenario)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: load problems ({exc})")
        emit_metrics_line(
            {
                "scenario": args.scenario,
                "kind": "external",
                "model": args.model,
                "prompt_tokens": None,
                "completion_tokens": None,
                "passed": 0,
                "failed": 0,
                "total": 0,
                "pass_rate": 0.0,
                "pass_at_1": 0.0,
                "capability_passed": False,
                "error": str(exc),
            }
        )
        return 1

    if not selected:
        print("FAIL: no problems selected")
        emit_metrics_line(
            {
                "scenario": args.scenario,
                "kind": "external",
                "model": args.model,
                "prompt_tokens": None,
                "completion_tokens": None,
                "passed": 0,
                "failed": 0,
                "total": 0,
                "pass_rate": 0.0,
                "pass_at_1": 0.0,
                "capability_passed": False,
            }
        )
        return 1

    passed = 0
    failed = 0
    prompt_tokens_sum = 0
    completion_tokens_sum = 0
    have_prompt_tokens = False
    have_completion_tokens = False
    results: list[dict[str, Any]] = []

    for problem in selected:
        pid = str(problem.get("id") or "").strip() or "problem"
        entry = str(problem.get("entry_point") or "").strip()
        test_src = str(problem.get("test") or "")
        detail = ""
        ok = False
        try:
            resp = ollama_generate(args.ollama, args.model, coding_prompt(problem))
            pt, ct = tokens_from_ollama(resp)
            if pt is not None:
                prompt_tokens_sum += pt
                have_prompt_tokens = True
            if ct is not None:
                completion_tokens_sum += ct
                have_completion_tokens = True
            code = extract_python_code(str(resp.get("response") or ""), entry_point=entry)
            ok, detail = run_harness(code, test_src, entry, timeout_s=float(args.timeout))
        except urllib.error.HTTPError as exc:
            detail = f"ollama HTTP {exc.code}"
        except urllib.error.URLError as exc:
            detail = f"ollama unavailable ({exc})"
        except Exception as exc:  # noqa: BLE001 — per-problem isolation
            detail = str(exc)[:200]

        if ok:
            passed += 1
            print(f"PASS: {pid}")
        else:
            failed += 1
            print(f"FAIL: {pid} {detail}")
        results.append({"id": pid, "passed": ok, "detail": detail})

    total = passed + failed
    pass_rate = (passed / total) if total else 0.0
    is_full_set = (args.scenario or "all").strip().lower() in {"", "all", "*"}
    if is_full_set:
        capability_ok = pass_rate >= FULL_SET_FLOOR
    else:
        capability_ok = passed == total and total > 0

    metrics: dict[str, Any] = {
        "scenario": "humaneval-25" if is_full_set else args.scenario,
        "kind": "external",
        "model": args.model,
        "prompt_tokens": prompt_tokens_sum if have_prompt_tokens else None,
        "completion_tokens": completion_tokens_sum if have_completion_tokens else None,
        "passed": passed,
        "failed": failed,
        "total": total,
        "pass_rate": round(pass_rate, 4),
        "pass_at_1": round(pass_rate, 4),
        "capability_passed": capability_ok,
        "license_note": license_note or None,
        "results": results,
    }
    emit_metrics_line(metrics)
    return 0 if capability_ok else 1


if __name__ == "__main__":
    sys.exit(main())
