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

from lib import collab_hub as hub  # noqa: E402
from lib.hub_regression import wait_for_hub  # noqa: E402
from lib.model_benchmark import (  # noqa: E402
    MODELS_CONFIG,
    ModelBenchmarkResult,
    ScenarioResult,
    SUITES_CONFIG,
    build_scenario_catalog,
    fetch_hardware_snapshot,
    filter_models_by_max_params,
    format_duration,
    load_models,
    load_scenario_meta,
    load_suite,
    model_is_installed,
    model_params_b,
    model_pull_tag,
    model_runtime_tag,
    ollama_installed_tags,
    pull_ollama_model,
    resolve_judge_provider_note,
    resolve_suite_max_params_b,
    resolve_suite_model_tags,
    resolve_suite_scenarios,
    run_script_scenario,
    switch_all_ollama,
    write_reports,
)

DEFAULT_OUT = ROOT / "docs" / "testing"


def parse_models_arg(
    raw: str,
    config_path: Path | None,
    *,
    max_params_b: float | None = None,
    allow_large: bool = False,
) -> list[dict]:
    catalog = {m["tag"]: m for m in load_models(config_path)}
    tags = [t.strip() for t in raw.split(",") if t.strip()]
    out: list[dict] = []
    for tag in tags:
        if tag in catalog:
            out.append(catalog[tag])
        else:
            out.append({"id": tag.replace(":", "-"), "tag": tag, "title": tag, "size_hint_gb": None, "notes": "cli override"})
    if max_params_b is not None and not allow_large:
        out = filter_models_by_max_params(out, max_params_b, allow_unknown=True)
    return out


def resolve_benchmark_models(
    suite: dict[str, Any],
    *,
    models_arg: str | None,
    models_config: Path,
    suites_config: Path,
    max_params_b: float | None,
    allow_large: bool,
) -> list[dict]:
    cap = max_params_b if max_params_b is not None else resolve_suite_max_params_b(suite, models_config)

    if models_arg:
        models = parse_models_arg(
            models_arg,
            models_config,
            max_params_b=cap,
            allow_large=allow_large,
        )
    elif resolve_suite_model_tags(suite):
        models = parse_models_arg(
            ",".join(resolve_suite_model_tags(suite)),
            models_config,
            max_params_b=cap,
            allow_large=allow_large,
        )
    else:
        models = load_models(models_config)
        if not allow_large:
            models = filter_models_by_max_params(models, cap)

    if not models:
        raise ValueError(f"no models at or below {cap}B params (use --allow-large-models to bypass cap)")
    return models


def benchmark_model(
    hub_url: str,
    model: dict,
    *,
    implement: list[str],
    chat: list[str],
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
    if not model_is_installed(installed, tag, pull_tag=pull_target):
        if pull:
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
    for name in implement:
        print(f"\n  [implement] {name}")
        passed, elapsed, step_detail, judge_passed, judge_reason, uses_judge = run_script_scenario(
            "implement-scenarios.py", name, hub_url
        )
        meta = load_scenario_meta("implement", name)
        result.scenarios.append(
            ScenarioResult(
                name=name,
                kind="implement",
                passed=passed,
                duration_s=elapsed,
                detail=step_detail,
                judge_passed=judge_passed,
                judge_reason=judge_reason,
                uses_llm_judge=uses_judge or bool(meta.get("llm_judge")),
            )
        )
        mark = "PASS" if passed else "FAIL"
        print(f"    {mark} in {format_duration(elapsed)} — {step_detail}")
        if verbose and not passed:
            return result
        _recover_implement_channel(hub_url)

    for name in chat:
        print(f"\n  [chat] {name}")
        passed, elapsed, step_detail, judge_passed, judge_reason, uses_judge = run_script_scenario(
            "chat-scenarios.py", name, hub_url
        )
        meta = load_scenario_meta("chat", name)
        result.scenarios.append(
            ScenarioResult(
                name=name,
                kind="chat",
                passed=passed,
                duration_s=elapsed,
                detail=step_detail,
                judge_passed=judge_passed,
                judge_reason=judge_reason,
                uses_llm_judge=uses_judge or bool(meta.get("llm_judge")),
            )
        )
        mark = "PASS" if passed else "FAIL"
        print(f"    {mark} in {format_duration(elapsed)} — {step_detail}")

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
        "--max-params-b",
        type=float,
        default=None,
        help="Cap benchmark roster to this size in billions of params (default: suite max_params_b or 24)",
    )
    p.add_argument(
        "--allow-large-models",
        action="store_true",
        help="Allow models above max-params-b cap (default cap is 24B)",
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
    args = p.parse_args()

    if args.list_suites:
        import json

        data = json.loads(args.suites_config.read_text(encoding="utf-8"))
        for name, suite in data.items():
            if not isinstance(suite, dict):
                continue
            impl, chat = resolve_suite_scenarios(suite)
            print(f"{name}\t{suite.get('description', '')}\timplement={len(impl)}\tchat={len(chat)}")
        return 0

    if args.list_models:
        cap = args.max_params_b if args.max_params_b is not None else resolve_suite_max_params_b({}, args.models_config)
        roster = load_models(args.models_config)
        if not args.allow_large_models:
            roster = filter_models_by_max_params(roster, cap)
        for m in roster:
            gb = m.get("size_hint_gb")
            size = f"~{gb} GB" if gb else "?"
            params = model_params_b(m)
            pb = f"{params}B" if params is not None else "?"
            print(f"{m.get('tag')}\t{m.get('title')}\t{pb}\t{size}\t{m.get('notes', '')}")
        if not args.allow_large_models:
            print(f"\n(roster capped at {cap}B — use --allow-large-models for full catalog)")
        return 0

    hub_url = args.hub.rstrip("/")
    if not wait_for_hub(hub_url):
        print(f"hub unhealthy: {hub_url}\nStart with: make server-regression", file=sys.stderr)
        return 1

    suite = load_suite(args.suite, args.suites_config)
    implement, chat = resolve_suite_scenarios(suite)
    if not implement and not chat:
        print(f"suite {args.suite!r} has no scenarios", file=sys.stderr)
        return 1

    try:
        models = resolve_benchmark_models(
            suite,
            models_arg=args.models,
            models_config=args.models_config,
            suites_config=args.suites_config,
            max_params_b=args.max_params_b,
            allow_large=args.allow_large_models,
        )
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    suite_desc = str(suite.get("description") or args.suite)
    cap = args.max_params_b if args.max_params_b is not None else resolve_suite_max_params_b(suite, args.models_config)

    print(f"Model benchmark suite={args.suite}")
    print(f"  {suite_desc}")
    print(f"  max_params_b: {cap}" + (" (large models allowed)" if args.allow_large_models else ""))
    print(f"  models: {', '.join(m['tag'] for m in models)}")
    print(f"  implement ({len(implement)}): {', '.join(implement) or '(none)'}")
    print(f"  chat ({len(chat)}): {', '.join(chat) or '(none)'}")

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
                implement=implement,
                chat=chat,
                pull=args.pull,
                skip_missing=args.skip_missing,
                cooldown_s=args.cooldown,
                verbose=args.verbose,
            )
        )
        if i+1 < len(models):
            _recover_hub_between_models(hub_url)

    md_path, json_path, tsv_path = write_reports(
        args.out_dir,
        suite_name=args.suite,
        suite_desc=suite_desc,
        hub_url=hub_url,
        results=results,
        implement_names=implement,
        chat_names=chat,
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

    winner = max(active, key=lambda r: (r.pass_rate, r.passed, -r.total_duration_s))
    print("\n=== Benchmark complete ===")
    print(f"Winner: {winner.model_tag} ({winner.passed}/{winner.total}, {winner.pass_rate * 100:.0f}%)")
    print(f"Markdown: {md_path}")
    print(f"JSON:     {json_path}")
    print(f"TSV:      {tsv_path}")

    try_publish_website()

    any_ran = any(r.scenarios for r in results)
    all_pass = all(r.passed == r.total for r in active if r.scenarios)
    winner_ok = winner.pass_rate >= args.min_winner_pass_rate
    if any_ran and winner_ok and not all_pass:
        print(
            f"Benchmark exit: winner pass ({winner.model_tag} "
            f"{winner.pass_rate * 100:.0f}% >= {args.min_winner_pass_rate * 100:.0f}% gate; weak models ignored)"
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
