#!/usr/bin/env python3
"""Run real-world user-flow scenarios (product journeys).

Scenarios live under scenarios/user-flows/{implement,collab}/ so they do not
inflate the core implement 20/20 or collab-core gates. Core scenarios can be
referenced by name (source=core) and run from the default scenario dirs.

  ./scripts/user-flow-scenarios.py --list
  ./scripts/user-flow-scenarios.py --scenario trip-research-vacation
  ./scripts/user-flow-scenarios.py --all
"""

from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS_DIR = ROOT / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.hub_regression import wait_for_hub  # noqa: E402
from lib.user_flow_scenarios import USER_FLOW_SCENARIOS, UserFlowEntry  # noqa: E402

USER_FLOWS_ROOT = ROOT / "scenarios" / "user-flows"
EMPTY_FIXTURE = ROOT / "scenarios" / "fixtures" / "user-flow-empty"
KEEP_NAMES = frozenset({".scenario-baseline", "README.md", ".gitkeep"})


def wipe_empty_fixture() -> None:
    """Remove agent-created files from the greenfield fixture; keep baseline + README."""
    if not EMPTY_FIXTURE.is_dir():
        return
    for child in EMPTY_FIXTURE.iterdir():
        if child.name in KEEP_NAMES:
            continue
        if child.is_dir():
            shutil.rmtree(child, ignore_errors=True)
        else:
            try:
                child.unlink()
            except OSError:
                pass
    baseline = EMPTY_FIXTURE / ".scenario-baseline" / "README.md"
    readme = EMPTY_FIXTURE / "README.md"
    if baseline.is_file():
        readme.write_bytes(baseline.read_bytes())


def run_cmd(args: list[str], *, env: dict[str, str] | None = None) -> int:
    print(f"\n>>> {' '.join(args)}")
    merged = os.environ.copy()
    merged["NEURAL_JUNKIE_RATE_LIMIT"] = "0"
    if env:
        merged.update(env)
    return subprocess.call(args, cwd=str(ROOT), env=merged)


def scenarios_dir_for(entry: UserFlowEntry) -> Path | None:
    if entry.source == "user-flows":
        return USER_FLOWS_ROOT / entry.kind
    return None


def run_entry(entry: UserFlowEntry, *, verbose: bool) -> int:
    wipe_empty_fixture()
    verbose_args = ["--verbose"] if verbose else []
    env: dict[str, str] = {}
    pack_dir = scenarios_dir_for(entry)
    if pack_dir is not None:
        env["NJ_PACK_SCENARIOS_DIR"] = str(pack_dir.resolve())

    if entry.kind == "implement":
        cmd = [
            "python3",
            "scripts/implement-scenarios.py",
            "--scenario",
            entry.name,
            "--hub",
            os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"),
            *verbose_args,
        ]
    elif entry.kind == "collab":
        cmd = [
            "python3",
            "scripts/collab-scenarios.py",
            "--scenario",
            entry.name,
            "--hub",
            os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"),
            *verbose_args,
        ]
    else:
        print(f"FAIL: unknown kind {entry.kind!r} for {entry.name}", file=sys.stderr)
        return 1

    code = run_cmd(cmd, env=env)
    wipe_empty_fixture()
    return code


def list_entries() -> None:
    print(f"{'name':<36} {'kind':<10} {'source':<12} {'status':<10} description")
    print("-" * 110)
    for entry in USER_FLOW_SCENARIOS:
        status = "skip" if entry.skip_reason else "active"
        print(
            f"{entry.name:<36} {entry.kind:<10} {entry.source:<12} {status:<10} {entry.description}"
        )
        if entry.skip_reason:
            print(f"{'':36} skip_reason: {entry.skip_reason}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--list", action="store_true", help="List suite members")
    parser.add_argument("--all", action="store_true", help="Run entire user-flow suite")
    parser.add_argument("--scenario", help="Run one scenario by name")
    parser.add_argument("--kind", choices=("implement", "collab"), help="Filter by kind")
    parser.add_argument("--verbose", action="store_true")
    parser.add_argument(
        "--include-skipped",
        action="store_true",
        help="With --all, also run entries that have skip_reason (default: park them)",
    )
    parser.add_argument(
        "--hub",
        default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"),
        help="Hub URL (also sets NEURAL_JUNKIE_HUB_URL for child runners)",
    )
    parser.add_argument(
        "--skip-hub-wait",
        action="store_true",
        help="Do not wait for hub (caller already booted)",
    )
    args = parser.parse_args()

    if args.list:
        list_entries()
        return 0

    selected: list[UserFlowEntry] = []
    if args.scenario:
        for entry in USER_FLOW_SCENARIOS:
            if entry.name == args.scenario:
                if args.kind and entry.kind != args.kind:
                    continue
                selected.append(entry)
                break
        if not selected:
            print(f"FAIL: unknown user-flow scenario {args.scenario!r}", file=sys.stderr)
            print("Known:", ", ".join(e.name for e in USER_FLOW_SCENARIOS), file=sys.stderr)
            return 1
    elif args.all:
        selected = [
            e for e in USER_FLOW_SCENARIOS if not args.kind or e.kind == args.kind
        ]
    else:
        parser.error("specify --list, --all, or --scenario <name>")

    # --all parks WIP journeys (e.g. trip research until mapping lands).
    # Explicit --scenario still runs skipped entries for debugging.
    skipped: list[UserFlowEntry] = []
    if args.all and not args.include_skipped:
        active: list[UserFlowEntry] = []
        for entry in selected:
            if entry.skip_reason:
                skipped.append(entry)
            else:
                active.append(entry)
        selected = active

    hub_url = (args.hub or os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765")).rstrip("/")
    os.environ["NEURAL_JUNKIE_HUB_URL"] = hub_url
    if not args.skip_hub_wait:
        print(f"Waiting for hub at {hub_url}...")
        if not wait_for_hub(hub_url, timeout_s=120.0):
            print(
                "FAIL: hub not reachable (start with: make server-regression "
                "or run via make user-flow-scenarios / layer-gate LAYER=user-flows)",
                file=sys.stderr,
            )
            return 1
        print("OK: hub ready")
        preflight_regression_run(ROOT, hub_url, label="user-flow-scenarios preflight")

    wipe_empty_fixture()
    for entry in skipped:
        print(f"\n=== SKIP user-flow [{entry.kind}/{entry.source}]: {entry.name} ===")
        print(f"    {entry.description}")
        print(f"    reason: {entry.skip_reason}")

    failed: list[str] = []
    for entry in selected:
        print(f"\n=== user-flow [{entry.kind}/{entry.source}]: {entry.name} ===")
        print(f"    {entry.description}")
        code = run_entry(entry, verbose=args.verbose)
        if code != 0:
            failed.append(f"{entry.kind}:{entry.name}")

    print("\n=== User-flow summary ===")
    passed = len(selected) - len(failed)
    print(f"PASS {passed}/{len(selected)}")
    if skipped:
        print(f"SKIPPED {len(skipped)}: {', '.join(e.name for e in skipped)}")
    if failed:
        print("FAILED:", ", ".join(failed), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
