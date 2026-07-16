#!/usr/bin/env python3
"""Pull Ollama models used by model-benchmark-suite (≤~9 GB footprint by default)."""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.model_benchmark import (  # noqa: E402
    MODELS_CONFIG,
    SUITES_CONFIG,
    load_suite,
    model_params_b,
    model_pull_tag,
    model_requires_hf_import,
    model_size_gb,
    resolve_benchmark_models,
    resolve_suite_max_size_gb,
)


def pull_tag(tag: str) -> bool:
    print(f"\n>>> ollama pull {tag}")
    proc = subprocess.run(["ollama", "pull", tag], cwd=ROOT, check=False)
    return proc.returncode == 0


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--suite", default="quick", help="Benchmark suite name (default: quick)")
    p.add_argument("--models-config", type=Path, default=MODELS_CONFIG)
    p.add_argument("--suites-config", type=Path, default=SUITES_CONFIG)
    p.add_argument("--models", help="Comma-separated tags (overrides suite roster)")
    p.add_argument(
        "--allow-large-models",
        action="store_true",
        help="Include catalog models above the max-size-gb footprint cap",
    )
    p.add_argument(
        "--max-size-gb",
        type=float,
        default=None,
        help="Cap pulls by size_hint_gb (default: suite/config, usually 9)",
    )
    p.add_argument("--max-params-b", type=float, default=None, help=argparse.SUPPRESS)
    args = p.parse_args()

    suite = load_suite(args.suite, args.suites_config)
    try:
        models = resolve_benchmark_models(
            suite,
            models_arg=args.models,
            models_config=args.models_config,
            suites_config=args.suites_config,
            max_size_gb=args.max_size_gb if args.max_size_gb is not None else args.max_params_b,
            allow_large=args.allow_large_models,
        )
    except ValueError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        return 1

    cap = (
        args.max_size_gb
        if args.max_size_gb is not None
        else args.max_params_b
        if args.max_params_b is not None
        else resolve_suite_max_size_gb(suite, args.models_config)
    )
    print(f"Pulling {len(models)} model(s) for suite={args.suite} (max {cap} GB footprint)")
    failed: list[str] = []
    skipped_import: list[str] = []
    for model in models:
        tag = str(model.get("tag") or "").strip()
        if not tag:
            continue
        if model_requires_hf_import(model):
            skipped_import.append(tag)
            print(
                f"\n>>> SKIP {tag}: requires HF import "
                f"(Settings → Models; not ollama-pullable)",
                file=sys.stderr,
            )
            continue
        pull = model_pull_tag(model)
        size = model_size_gb(model)
        params = model_params_b(model)
        parts = []
        if params is not None:
            parts.append(f"{params}B params")
        if size is not None:
            parts.append(f"~{size} GB")
        label = f"{tag} ({', '.join(parts)})" if parts else tag
        if not pull_tag(pull):
            failed.append(label)

    if skipped_import:
        print(
            f"\nSkipped HF-import models (import manually if needed): {', '.join(skipped_import)}",
            file=sys.stderr,
        )
    if failed:
        print(f"\nFAIL: pull failed for: {', '.join(failed)}", file=sys.stderr)
        return 1
    print("\nAll pullable benchmark models pulled.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
