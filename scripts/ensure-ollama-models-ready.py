#!/usr/bin/env python3
"""Pull, warm-load, and smoke-test Ollama models before long regression runs."""

from __future__ import annotations

import argparse
import json
import os
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
    filter_models_by_max_size_gb,
    load_models,
    load_suite,
    model_is_installed,
    model_pull_tag,
    model_requires_hf_import,
    model_runtime_tag,
    resolve_suite_max_size_gb,
    resolve_suite_model_tags,
)

OLLAMA_BASE = "http://127.0.0.1:11434"


def _load_expected_agent_models() -> list[str]:
    from lib.regression_models import regression_ollama_warm_tags

    return regression_ollama_warm_tags(ROOT)


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


def catalog_by_tag() -> dict[str, dict]:
    return {
        str(m.get("tag") or "").strip(): m
        for m in load_models(MODELS_CONFIG)
        if isinstance(m, dict) and str(m.get("tag") or "").strip()
    }


def ollama_pull(tag: str) -> bool:
    print(f"  pull {tag} …")
    proc = subprocess.run(["ollama", "pull", tag], cwd=ROOT, check=False)
    return proc.returncode == 0


def resolve_pull_target(tag: str, catalog: dict[str, dict] | None = None) -> str:
    """Map brand/catalog tags (nj-ornith:9b) to their Ollama pull targets (hf.co/…)."""
    cat = catalog if catalog is not None else catalog_by_tag()
    model = cat.get(tag.strip())
    if not model:
        return tag.strip()
    if model_requires_hf_import(model):
        return ""
    return model_pull_tag(model) or tag.strip()


def is_optional_benchmark_tag(tag: str, warm_tags: list[str]) -> bool:
    return tag not in warm_tags


def warm_tag(tag: str, *, keep_alive: str, timeout_s: float) -> tuple[bool, str]:
    body = json.dumps(
        {
            "model": tag,
            "prompt": "ping",
            "stream": False,
            "keep_alive": keep_alive,
            # Cap tokens so a wedged runner fails fast instead of hanging for the full timeout.
            "options": {"num_predict": 4},
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


def smoke_tag(
    tag: str,
    *,
    keep_alive: str = "24h",
    timeout_s: float = 120.0,
) -> tuple[bool, str]:
    body = json.dumps(
        {
            "model": tag,
            "prompt": "Reply with exactly: ok",
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
    cap = resolve_suite_max_size_gb(suite_cfg, MODELS_CONFIG)
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
        models = filter_models_by_max_size_gb(models, cap)
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

    catalog = catalog_by_tag()
    if args.pull_missing:
        failed_pull: list[str] = []
        skipped_import: list[str] = []
        for tag in pull_tags:
            model = catalog.get(tag)
            if model and model_requires_hf_import(model):
                if model_is_installed(installed, tag):
                    print(f"  installed: {tag} (HF import)")
                else:
                    print(f"  >>> importing HF GGUF for {tag} …")
                    proc = subprocess.run(
                        [
                            sys.executable,
                            str(ROOT / "scripts" / "import-hf-gguf-models.py"),
                            "--models",
                            tag,
                        ],
                        cwd=ROOT,
                        check=False,
                    )
                    installed = installed_tags()
                    if proc.returncode != 0 or not model_is_installed(installed, tag):
                        if is_optional_benchmark_tag(tag, warm_tags):
                            skipped_import.append(tag)
                            print(
                                f"  WARN: HF import failed for {tag} (continuing; benchmark will skip)",
                                file=sys.stderr,
                            )
                        else:
                            failed_pull.append(f"{tag} (HF import)")
                continue
            target = resolve_pull_target(tag, catalog)
            if not target:
                continue
            if model_is_installed(installed, tag, pull_tag=target):
                label = tag if target == tag else f"{tag} (via {target})"
                print(f"  installed: {label}")
                continue
            if not ollama_pull(target):
                label = tag if target == tag else f"{tag} ← {target}"
                if is_optional_benchmark_tag(tag, warm_tags):
                    print(
                        f"  WARN: optional pull failed for {label} (continuing)",
                        file=sys.stderr,
                    )
                else:
                    failed_pull.append(label)
                continue
            installed = installed_tags()
        if failed_pull:
            print(f"FAIL: pull failed for: {', '.join(failed_pull)}", file=sys.stderr)
            return 1
        if skipped_import:
            print(
                f"  note: {len(skipped_import)} HF-import model(s) not installed — "
                f"benchmark will skip them unless imported",
                file=sys.stderr,
            )

    missing = [
        t
        for t in warm_tags
        if not model_is_installed(installed, t, pull_tag=resolve_pull_target(t, catalog) or "")
    ]
    if missing:
        print(
            f"FAIL: models not installed: {', '.join(missing)} "
            f"(run with --pull-missing or: ollama pull <tag>)",
            file=sys.stderr,
        )
        return 1

    # Suite roster (e.g. gemma3:12b for release) must also be present even when
    # NO_PULL=1 — otherwise overnight boots green and fails hours later in benchmark.
    if not args.skip_benchmark:
        missing_suite = [
            t
            for t in pull_tags
            if t not in warm_tags
            and not model_is_installed(installed, t, pull_tag=resolve_pull_target(t, catalog) or "")
        ]
        if missing_suite:
            print(
                f"FAIL: benchmark suite models not installed: {', '.join(missing_suite)} "
                f"(overnight NO_PULL=1 will not pull them — run: make ensure-ollama-models-ready SUITE={args.suite})",
                file=sys.stderr,
            )
            return 1

    def runtime(tag: str) -> str:
        model = catalog.get(tag)
        if model:
            return model_runtime_tag(model, installed)
        return tag

    if args.warm:
        for tag in warm_tags:
            use = runtime(tag)
            label = tag if use == tag else f"{tag} → {use}"
            print(f"  warming {label} …")
            ok, detail = warm_tag(use, keep_alive=args.keep_alive, timeout_s=args.warm_timeout)
            if not ok:
                print(f"FAIL: warm {tag}: {detail}", file=sys.stderr)
                return 1
            print(f"  OK warm {tag}: {detail}")

    if args.smoke:
        for tag in warm_tags:
            use = runtime(tag)
            label = tag if use == tag else f"{tag} → {use}"
            print(f"  smoke {label} …")
            ok, detail = smoke_tag(use, keep_alive=args.keep_alive)
            if not ok:
                print(f"FAIL: smoke {tag}: {detail}", file=sys.stderr)
                return 1
            print(f"  OK smoke {tag}: {detail}")

    # End with the primary agent model resident; smaller/tool models may be evicted on tight RAM.
    if args.warm:
        primary = warm_tags[0]
        primary_rt = runtime(primary)
        loaded = loaded_tags()
        if primary_rt not in loaded and primary not in loaded:
            print(f"  re-warm {primary} (primary agent model) …")
            ok, detail = warm_tag(primary_rt, keep_alive=args.keep_alive, timeout_s=args.warm_timeout)
            if not ok:
                print(f"FAIL: re-warm {primary}: {detail}", file=sys.stderr)
                return 1
            print(f"  OK re-warm {primary}: {detail}")

    loaded = loaded_tags()
    primary = warm_tags[0]
    primary_rt = runtime(primary)
    if args.warm and primary_rt not in loaded and primary not in loaded:
        print(f"FAIL: primary model not in /api/ps: {primary}", file=sys.stderr)
        return 1

    optional_cold = [
        t
        for t in warm_tags[1:]
        if runtime(t) not in loaded and t not in loaded
    ]
    if optional_cold:
        print(
            f"  WARN: not all warm models resident simultaneously "
            f"(smoke passed; cold: {', '.join(optional_cold)})",
            file=sys.stderr,
        )

    if loaded:
        print(f"  loaded now: {', '.join(sorted(loaded))}")
    print("OK: Ollama models ready")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
