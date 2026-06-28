#!/usr/bin/env python3
"""Pull, warm-load, and smoke-test Ollama models before long regression runs."""

from __future__ import annotations

import argparse
import importlib.util
import json
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.model_benchmark import (  # noqa: E402
    MODELS_CONFIG,
    SUITES_CONFIG,
    filter_models_by_max_params,
    load_models,
    load_suite,
    resolve_suite_max_params_b,
    resolve_suite_model_tags,
)

OLLAMA_BASE = "http://127.0.0.1:11434"


def _load_expected_agent_models() -> list[str]:
    spec = importlib.util.spec_from_file_location(
        "collab_preflight", SCRIPTS_DIR / "collab-preflight.py"
    )
    if spec is None or spec.loader is None:
        return ["qwen3.5:9b"]
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod.load_expected_ollama_models()


def _ollama_get(path: str, *, timeout: float = 10.0) -> dict | list | None:
    url = f"{OLLAMA_BASE.rstrip('/')}{path}"
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            return json.loads(resp.read().decode())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError):
        return None


def installed_tags() -> set[str]:
    data = _ollama_get("/api/tags")
    if not isinstance(data, dict):
        return set()
    names: set[str] = set()
    for item in data.get("models") or []:
        if isinstance(item, dict):
            name = (item.get("name") or "").strip()
            if name:
                names.add(name)
    return names


def loaded_tags() -> set[str]:
    data = _ollama_get("/api/ps")
    if not isinstance(data, dict):
        return set()
    names: set[str] = set()
    for item in data.get("models") or []:
        if isinstance(item, dict):
            name = (item.get("name") or item.get("model") or "").strip()
            if name:
                names.add(name)
    return names


def pull_tag(tag: str) -> bool:
    print(f"  pull {tag} …")
    proc = subprocess.run(["ollama", "pull", tag], cwd=ROOT, check=False)
    return proc.returncode == 0


def warm_tag(tag: str, *, keep_alive: str, timeout_s: float) -> tuple[bool, str]:
    body = json.dumps(
        {
            "model": tag,
            "prompt": "ping",
            "stream": False,
            "keep_alive": keep_alive,
        }
    ).encode()
    req = urllib.request.Request(
        f"{OLLAMA_BASE}/api/generate",
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    started = time.monotonic()
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            raw = resp.read().decode(errors="replace")
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode(errors="replace") if exc.fp else str(exc)
        return False, f"HTTP {exc.code}: {detail[:240]}"
    except (urllib.error.URLError, TimeoutError) as exc:
        return False, str(exc)
    elapsed = time.monotonic() - started
    if tag not in loaded_tags():
        return False, f"generate ok in {elapsed:.0f}s but model not listed in /api/ps"
    return True, f"loaded in {elapsed:.0f}s (keep_alive={keep_alive})"


def smoke_tag(tag: str, *, timeout_s: float = 120.0) -> tuple[bool, str]:
    body = json.dumps(
        {
            "model": tag,
            "prompt": "Reply with exactly: ok",
            "stream": False,
        }
    ).encode()
    req = urllib.request.Request(
        f"{OLLAMA_BASE}/api/generate",
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            data = json.loads(resp.read().decode())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, urllib.error.HTTPError) as exc:
        return False, str(exc)
    text = (data.get("response") or "").strip()
    if not text:
        return False, "empty response"
    return True, text.splitlines()[0][:80]


def benchmark_pull_tags(
    *,
    suite: str,
    models_arg: str | None,
    allow_large: bool,
) -> list[str]:
    suite_cfg = load_suite(suite, SUITES_CONFIG)
    cap = resolve_suite_max_params_b(suite_cfg, MODELS_CONFIG)
    if models_arg:
        want = [t.strip() for t in models_arg.split(",") if t.strip()]
        catalog = {str(m.get("tag") or "").strip(): m for m in load_models(MODELS_CONFIG)}
        tags = [t for t in want if t in catalog or allow_large]
        return tags
    explicit = resolve_suite_model_tags(suite_cfg)
    if explicit:
        catalog = {str(m.get("tag") or "").strip(): m for m in load_models(MODELS_CONFIG)}
        return [t for t in explicit if t in catalog]
    models = load_models(MODELS_CONFIG)
    if not allow_large:
        models = filter_models_by_max_params(models, cap)
    tags: list[str] = []
    seen: set[str] = set()
    for model in models:
        tag = str(model.get("tag") or "").strip()
        if tag and tag not in seen:
            seen.add(tag)
            tags.append(tag)
    return tags


def resolve_tags(args: argparse.Namespace) -> tuple[list[str], list[str]]:
    if args.models:
        warm = [t.strip() for t in args.models.split(",") if t.strip()]
        pull = list(warm)
    else:
        warm = _load_expected_agent_models()
        pull = list(warm)
    if not args.skip_benchmark:
        for tag in benchmark_pull_tags(
            suite=args.suite,
            models_arg=args.benchmark_models,
            allow_large=args.allow_large_models,
        ):
            if tag not in pull:
                pull.append(tag)
    return warm, pull


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--suite", default="quick", help="Benchmark suite for extra pull tags")
    p.add_argument("--benchmark-models", help="Comma-separated benchmark tags (override suite roster)")
    p.add_argument(
        "--models",
        help="Comma-separated tags to warm (default: agent models from env/config)",
    )
    p.add_argument("--skip-benchmark", action="store_true", help="Do not pull benchmark suite tags")
    p.add_argument("--pull-missing", action="store_true", help="ollama pull any tag not installed")
    p.add_argument("--warm", action="store_true", help="Load warm models into RAM via /api/generate")
    p.add_argument("--smoke", action="store_true", help="Run a tiny generate on each warmed model")
    p.add_argument(
        "--keep-alive",
        default="24h",
        help="keep_alive passed to Ollama when warming (default: 24h)",
    )
    p.add_argument("--warm-timeout", type=float, default=600.0, help="Seconds per warm load")
    p.add_argument("--allow-large-models", action="store_true")
    args = p.parse_args()

    warm_tags, pull_tags = resolve_tags(args)
    if not warm_tags:
        print("FAIL: no models to warm", file=sys.stderr)
        return 1

    print(">>> Ollama model readiness")
    print(f"  warm: {', '.join(warm_tags)}")
    if pull_tags != warm_tags:
        print(f"  pull roster: {', '.join(pull_tags)}")

    installed = installed_tags()
    if not installed and _ollama_get("/api/tags") is None:
        print("FAIL: Ollama not reachable at 127.0.0.1:11434", file=sys.stderr)
        return 1

    if args.pull_missing:
        failed_pull: list[str] = []
        for tag in pull_tags:
            if tag in installed:
                print(f"  installed: {tag}")
                continue
            if not pull_tag(tag):
                failed_pull.append(tag)
                continue
            installed.add(tag)
        if failed_pull:
            print(f"FAIL: pull failed for: {', '.join(failed_pull)}", file=sys.stderr)
            return 1

    missing = [t for t in warm_tags if t not in installed]
    if missing:
        print(
            f"FAIL: models not installed: {', '.join(missing)} "
            f"(run with --pull-missing or: ollama pull <tag>)",
            file=sys.stderr,
        )
        return 1

    if args.warm:
        for tag in warm_tags:
            print(f"  warming {tag} …")
            ok, detail = warm_tag(tag, keep_alive=args.keep_alive, timeout_s=args.warm_timeout)
            if not ok:
                print(f"FAIL: warm {tag}: {detail}", file=sys.stderr)
                return 1
            print(f"  OK warm {tag}: {detail}")

    if args.smoke:
        for tag in warm_tags:
            print(f"  smoke {tag} …")
            ok, detail = smoke_tag(tag)
            if not ok:
                print(f"FAIL: smoke {tag}: {detail}", file=sys.stderr)
                return 1
            print(f"  OK smoke {tag}: {detail}")

    loaded = loaded_tags()
    still_cold = [t for t in warm_tags if t not in loaded]
    if args.warm and still_cold:
        print(f"FAIL: warmed models not in /api/ps: {', '.join(still_cold)}", file=sys.stderr)
        return 1

    if loaded:
        print(f"  loaded now: {', '.join(sorted(loaded))}")
    print("OK: Ollama models ready")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
