#!/usr/bin/env python3
"""Run NJ live scenarios against multiple Ollama coder models and emit benchmarks."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
import time
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.model_benchmark import (  # noqa: E402
    MODELS_CONFIG,
    ModelBenchmarkResult,
    SCRIPT_BY_KIND,
    SUITES_CONFIG,
    SuiteTracks,
    fetch_hardware_snapshot,
    filter_models_by_max_size_gb,
    format_duration,
    load_models,
    load_scenario_meta,
    load_suite,
    model_is_installed,
    model_params_b,
    model_pull_tag,
    model_requires_hf_import,
    model_runtime_tag,
    ollama_installed_tags,
    pull_ollama_model,
    resolve_benchmark_models,
    resolve_judge_provider_note,
    resolve_suite_max_size_gb,
    resolve_suite_scenarios,
    run_script_scenario,
    switch_all_ollama,
    write_reports,
)
from lib.regression_boot import maybe_boot_regression  # noqa: E402

DEFAULT_OUT = ROOT / "docs" / "testing"


def configure_release_judge_isolation(model_tags: list[str]) -> None:
    os.environ.setdefault("NJ_DELIVERABLE_JUDGE_PROVIDER", "claude")
    os.environ["NJ_DELIVERABLE_JUDGE_STRICT"] = "1"
    provider = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "").lower()
    judge_model = os.environ.get("NJ_DELIVERABLE_JUDGE_MODEL", "qwen2.5-coder:14b")
    if provider == "ollama" and judge_model in model_tags:
        # refuse same-model judging
        os.environ["NJ_DELIVERABLE_JUDGE_MODEL"] = (
            "qwen3.5:9b" if "qwen3.5:9b" not in model_tags else "deepseek-coder:6.7b"
        )
        print("WARN: release judge model matched SUT; swapped judge model", file=sys.stderr)


def benchmark_model(
    hub_url: str,
    model: dict,
    *,
    tracks: SuiteTracks,
    pull: bool,
    skip_missing: bool,
    cooldown_s: float,
    verbose: bool,
) -> ModelBenchmarkResult:
    tag = (model.get("tag") or "").strip()
    pull_target = model_pull_tag(model)
    result = ModelBenchmarkResult(
        model_id=str(model.get("id") or tag),
        model_tag=tag,
        title=str(model.get("title") or tag),
        size_hint_gb=model.get("size_hint_gb"),
    )

    installed = ollama_installed_tags(hub_url)
    if model_requires_hf_import(model):
        if not model_is_installed(installed, tag):
            result.skipped = True
            result.skip_reason = (
                "requires HF import (Settings → Models); custom GGUF is not ollama-pullable"
            )
            return result
    elif not model_is_installed(installed, tag, pull_tag=pull_target):
        if pull and pull_target:
            print(f"  pulling {pull_target}…")
            ok, detail = pull_ollama_model(hub_url, pull_target)
            print(f"  pull: {detail}")
            if not ok:
                result.skipped = True
                result.skip_reason = f"pull failed: {detail}"
                return result
            installed = ollama_installed_tags(hub_url)
        if not model_is_installed(installed, tag, pull_tag=pull_target):
            if skip_missing:
                result.skipped = True
                result.skip_reason = "model not installed (use --pull or ollama pull)"
                return result
            print(f"  WARN: {tag} not in ollama list; continuing anyway", file=sys.stderr)

    runtime_tag = model_runtime_tag(model, installed)
    print(f"\n{'=' * 60}\nModel: {result.title} ({runtime_tag})\n{'=' * 60}")
    t_switch = time.monotonic()
    ok, detail = switch_all_ollama(hub_url, runtime_tag)
    result.switch_duration_s = time.monotonic() - t_switch
    if not ok:
        result.skipped = True
        result.skip_reason = detail
        print(f"  SKIP: {detail}", file=sys.stderr)
        return result
    print(f"  switched agents → {runtime_tag} ({detail})")
    if cooldown_s > 0:
        print(f"  cooldown {cooldown_s:.0f}s for agents to reload…")
        time.sleep(cooldown_s)

    model_t0 = time.monotonic()

    def _run_track(kind: str, names: list[str], *, with_model: bool = False) -> None:
        script = SCRIPT_BY_KIND[kind]
        for name in names:
            print(f"\n  [{kind}] {name}")
            extra: list[str] = []
            if with_model:
                extra.extend(["--model", runtime_tag])
            scenario = run_script_scenario(script, name, hub_url, kind=kind, extra_args=extra or None)
            meta = load_scenario_meta(kind, name)
            if meta.get("llm_judge"):
                scenario.uses_llm_judge = True
            result.scenarios.append(scenario)
            mark = "PASS" if scenario.passed else "FAIL"
            print(f"    {mark} in {format_duration(scenario.duration_s)} — {scenario.detail}")
            if verbose and not scenario.passed and kind == "implement":
                return
            if kind == "implement":
                _recover_implement_channel(hub_url)

    _run_track("implement", tracks.implement)
    if verbose and any(not s.passed for s in result.scenarios if s.kind == "implement"):
        result.total_duration_s = time.monotonic() - model_t0
        return result
    _run_track("chat", tracks.chat)
    _run_track("collab", tracks.collab)
    _run_track("arena", tracks.arena, with_model=True)
    _run_track("cad", tracks.cad, with_model=True)
    _run_track("external", tracks.external, with_model=True)

    result.total_duration_s = time.monotonic() - model_t0
    print(
        f"\n  Model total: {result.passed}/{result.total} pass in {format_duration(result.total_duration_s)}"
    )
    return result


def _recover_hub_between_models(hub_url: str) -> None:
    """Light reset between model switches so implement/chat scenarios do not inherit stale state."""
    try:
        from lib.fixture_cleanup import preflight_regression_run  # noqa: WPS433

        preflight_regression_run(ROOT, hub_url, label="model-benchmark between models")
    except Exception as exc:  # pragma: no cover - best effort
        print(f"  WARN: hub preflight between models: {exc}", file=sys.stderr)


def _recover_implement_channel(hub_url: str) -> None:
    """Reset implement-scenarios between heavy benchmark scenarios (e.g. go-handler → theme-toggle)."""
    try:
        from lib import collab_hub as hub  # noqa: WPS433
        from lib.fixture_baseline import reset_fixture_baseline  # noqa: WPS433

        hub.ensure_channel(hub_url, "implement-scenarios", description="Scenario regression channel")
        hub.clear_channel_history(hub_url, "implement-scenarios")
        reset_fixture_baseline({"workspace": {"fixture": "minimal-repo"}}, root=ROOT)
        time.sleep(2.0)
    except Exception as exc:  # pragma: no cover - best effort
        print(f"  WARN: implement channel reset: {exc}", file=sys.stderr)


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--suite", default="quick", help="Suite name from scripts/config/model-benchmark-suites.json")
    p.add_argument("--models-config", type=Path, default=MODELS_CONFIG)
    p.add_argument("--suites-config", type=Path, default=SUITES_CONFIG)
    p.add_argument(
        "--models",
        help="Comma-separated Ollama tags (overrides default roster)",
    )
    p.add_argument("--out-dir", type=Path, default=DEFAULT_OUT)
    p.add_argument("--pull", action="store_true", help="Pull missing models via hub before each run")
    p.add_argument("--skip-missing", action="store_true", help="Skip models that are not installed")
    p.add_argument(
        "--max-size-gb",
        type=float,
        default=None,
        help="Cap roster by size_hint_gb (default: suite max_size_gb or ~9 GB = Qwen 2.5 Coder 14B)",
    )
    p.add_argument(
        "--max-params-b",
        type=float,
        default=None,
        help=argparse.SUPPRESS,  # deprecated alias for --max-size-gb
    )
    p.add_argument(
        "--allow-large-models",
        action="store_true",
        help="Allow models above max-size-gb footprint cap (default ~9 GB)",
    )
    p.add_argument("--cooldown", type=float, default=5.0, help="Seconds after switch-all before scenarios")
    p.add_argument("--verbose", "-v", action="store_true")
    p.add_argument("--list-suites", action="store_true")
    p.add_argument("--list-models", action="store_true")
    p.add_argument(
        "--min-winner-pass-rate",
        type=float,
        default=1.0,
        help="Exit 0 when the top model meets this pass rate even if weaker models fail (default: 1.0)",
    )
    p.add_argument(
        "--min-model-pass-rate",
        type=float,
        default=0.5,
        help="Release gate: every active model must meet this pass rate (default: 0.5)",
    )
    args = p.parse_args()

    if args.list_suites:
        import json

        data = json.loads(args.suites_config.read_text(encoding="utf-8"))
        for name, suite in data.items():
            if not isinstance(suite, dict):
                continue
            tracks = resolve_suite_scenarios(suite)
            print(
                f"{name}\t{suite.get('description', '')}\t"
                f"implement={len(tracks.implement)}\tchat={len(tracks.chat)}\t"
                f"collab={len(tracks.collab)}\tarena={len(tracks.arena)}\t"
                f"cad={len(tracks.cad)}\texternal={len(tracks.external)}"
            )
        return 0

    if args.list_models:
        cap = (
            args.max_size_gb
            if args.max_size_gb is not None
            else args.max_params_b
            if args.max_params_b is not None
            else resolve_suite_max_size_gb({}, args.models_config)
        )
        roster = load_models(args.models_config)
        if not args.allow_large_models:
            roster = filter_models_by_max_size_gb(roster, cap)
        for m in roster:
            gb = m.get("size_hint_gb")
            size = f"~{gb} GB" if gb else "?"
            params = model_params_b(m)
            pb = f"{params}B" if params is not None else "?"
            print(f"{m.get('tag')}\t{m.get('title')}\t{pb}\t{size}\t{m.get('notes', '')}")
        if not args.allow_large_models:
            print(f"\n(roster capped at {cap} GB — use --allow-large-models for full catalog)")
        return 0

    hub_url = args.hub.rstrip("/")
    # Match implement/collab/test-everything: boot Ollama + models + regression hub.
    os.environ["BENCHMARK_SUITE"] = args.suite
    if args.models:
        os.environ.setdefault("BENCHMARK_MODELS", args.models)
    if args.allow_large_models:
        os.environ["BENCHMARK_ALLOW_LARGE"] = "1"
    if not maybe_boot_regression(hub_url, root=ROOT, label="model-benchmark"):
        return 1

    suite = load_suite(args.suite, args.suites_config)
    tracks = resolve_suite_scenarios(suite)
    if not tracks.has_any():
        print(f"suite {args.suite!r} has no scenarios", file=sys.stderr)
        return 1

    if tracks.arena:
        from lib.arena_pack import ensure_model_arena_pack

        ok_arena, detail_arena = ensure_model_arena_pack(hub_url)
        print(f"  arena pack: {detail_arena}")
        if not ok_arena:
            print(f"FAIL: {detail_arena}", file=sys.stderr)
            return 1

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
        print(str(exc), file=sys.stderr)
        return 1
    suite_desc = str(suite.get("description") or args.suite)
    cap = (
        args.max_size_gb
        if args.max_size_gb is not None
        else args.max_params_b
        if args.max_params_b is not None
        else resolve_suite_max_size_gb(suite, args.models_config)
    )

    if args.suite == "release":
        configure_release_judge_isolation([str(m.get("tag") or "").strip() for m in models])

    print(f"Model benchmark suite={args.suite}")
    print(f"  {suite_desc}")
    print(f"  max_size_gb: {cap}" + (" (large models allowed)" if args.allow_large_models else ""))
    print(f"  models: {', '.join(m['tag'] for m in models)}")
    print(f"  implement ({len(tracks.implement)}): {', '.join(tracks.implement) or '(none)'}")
    print(f"  chat ({len(tracks.chat)}): {', '.join(tracks.chat) or '(none)'}")
    print(f"  collab ({len(tracks.collab)}): {', '.join(tracks.collab) or '(none)'}")
    print(f"  arena ({len(tracks.arena)}): {', '.join(tracks.arena) or '(none)'}")
    print(f"  cad ({len(tracks.cad)}): {', '.join(tracks.cad) or '(none)'}")
    print(f"  external ({len(tracks.external)}): {', '.join(tracks.external) or '(none)'}")

    hardware = fetch_hardware_snapshot(hub_url)
    judge_provider = resolve_judge_provider_note()
    if hardware:
        print(f"  hardware: {hardware.get('total_memory_gb')} GB RAM ({hardware.get('tier')} tier)")
    print(f"  deliverable judge: {judge_provider}")

    _recover_hub_between_models(hub_url)

    results: list[ModelBenchmarkResult] = []
    for i, model in enumerate(models):
        results.append(
            benchmark_model(
                hub_url,
                model,
                tracks=tracks,
                pull=args.pull,
                skip_missing=args.skip_missing,
                cooldown_s=args.cooldown,
                verbose=args.verbose,
            )
        )
        if i + 1 < len(models):
            _recover_hub_between_models(hub_url)

    md_path, json_path, tsv_path = write_reports(
        args.out_dir,
        suite_name=args.suite,
        suite_desc=suite_desc,
        hub_url=hub_url,
        results=results,
        implement_names=tracks.implement,
        chat_names=tracks.chat,
        collab_names=tracks.collab,
        arena_names=tracks.arena,
        cad_names=tracks.cad,
        external_names=tracks.external,
        hardware=hardware,
        judge_provider=judge_provider,
    )

    active = [r for r in results if not r.skipped and r.scenarios]
    if not active:
        msg = "\nNo models benchmarked."
        if args.pull:
            msg += " Pull attempts did not produce any runnable models."
            print(msg, file=sys.stderr)
            if md_path:
                print(f"Markdown: {md_path}")
            return 1
        if args.skip_missing:
            msg += " (all skipped — use --pull or make pull-benchmark-models)"
            print(msg, file=sys.stderr)
            if md_path:
                print(f"Markdown: {md_path}")
            return 0
        print(msg, file=sys.stderr)
        return 1

    use_composite = args.suite == "release"
    rate_of = (lambda r: r.composite_pass_rate) if use_composite else (lambda r: r.pass_rate)
    winner = max(active, key=lambda r: (rate_of(r), r.passed, -r.total_duration_s))
    print("\n=== Benchmark complete ===")
    print(f"Winner: {winner.model_tag} ({winner.passed}/{winner.total}, {rate_of(winner) * 100:.0f}%)")
    print(f"Markdown: {md_path}")
    print(f"JSON:     {json_path}")
    print(f"TSV:      {tsv_path}")

    try_publish_website()

    any_ran = any(r.scenarios for r in results)
    all_pass = all(r.passed == r.total for r in active if r.scenarios)
    winner_ok = rate_of(winner) >= args.min_winner_pass_rate

    if args.suite == "release":
        models_ok = all(rate_of(r) >= args.min_model_pass_rate for r in active)
        if any_ran and winner_ok and not models_ok:
            print(
                f"Benchmark exit: release gate failed — some models below "
                f"{args.min_model_pass_rate * 100:.0f}% min model pass rate",
                file=sys.stderr,
            )
        return 0 if any_ran and winner_ok and models_ok else 1

    if any_ran and winner_ok and not all_pass:
        print(
            f"Benchmark exit: winner pass ({winner.model_tag} "
            f"{rate_of(winner) * 100:.0f}% >= {args.min_winner_pass_rate * 100:.0f}% gate; weak models ignored)"
        )
    return 0 if any_ran and (all_pass or winner_ok) else 1


def try_publish_website() -> None:
    publish_script = ROOT / "scripts" / "publish-model-benchmarks.py"
    if not publish_script.is_file():
        return
    print("\nPublishing to docs/data/model-benchmarks.json …")
    proc = subprocess.run(
        [sys.executable, str(publish_script)],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
    )
    if proc.stdout:
        print(proc.stdout.rstrip())
    if proc.returncode != 0 and proc.stderr:
        print(proc.stderr.rstrip(), file=sys.stderr)


if __name__ == "__main__":
    raise SystemExit(main())
