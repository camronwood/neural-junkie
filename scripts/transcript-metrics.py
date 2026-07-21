#!/usr/bin/env python3
"""Run deterministic conversation metrics over sanitized transcript fixtures."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT = SCRIPT_DIR.parent
sys.path.insert(0, str(SCRIPT_DIR))

from lib.transcript_contract import check_thresholds, evaluate_contract, extract_transcript  # noqa: E402

DEFAULT_FIXTURES = SCRIPT_DIR / "fixtures" / "transcript_metrics"


def load_fixture(path: Path) -> dict:
    with path.open(encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise ValueError(f"{path}: expected JSON object")
    return data


def run_fixture(path: Path) -> tuple[bool, dict]:
    fixture = load_fixture(path)
    result = evaluate_contract(fixture)
    failures = check_thresholds(result, fixture.get("thresholds") or {})
    expected = fixture.get("expected_metrics") or {}
    for name, value in expected.items():
        actual = result["metrics"].get(name)
        if actual != value:
            failures.append(f"{name}={actual!r}, expected {value!r}")
    return not failures, {"fixture": path.name, **result, "failures": failures}


def extract_session(path: Path, channel: str) -> dict:
    data = load_fixture(path)
    channels = data.get("channels") or {}
    if channel not in channels:
        choices = ", ".join(sorted(channels)[:10])
        raise ValueError(f"{path}: channel {channel!r} not found (available: {choices})")
    messages = (channels[channel] or {}).get("messages") or []
    return extract_transcript(messages, source=f"session:{channel}")


def advisory_judge(reports: list[dict]) -> None:
    """An LLM assessment is deliberately informational and never changes exit status."""
    try:
        from lib.deliverable_judge import judge_deliverable
    except ImportError as exc:
        print(f"ADVISORY judge unavailable: {exc}", file=sys.stderr)
        return
    body = json.dumps(reports, indent=2, sort_keys=True)
    ok, detail, score = judge_deliverable(
        question="Review these deterministic conversation metric results for obvious blind spots.",
        rel_path="sanitized-transcript-metrics.json",
        file_body=body,
        criteria="This is advisory only. Discuss coverage; do not override deterministic results.",
        work_dir=str(ROOT),
    )
    print(f"ADVISORY judge: {'PASS' if ok else 'NOTE'} score={score} {detail}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("fixtures", nargs="*", type=Path)
    parser.add_argument("--json", action="store_true", help="Emit machine-readable reports")
    parser.add_argument("--extract-session", type=Path, help="Sanitize one persisted hub session")
    parser.add_argument("--channel", help="Channel to extract with --extract-session")
    parser.add_argument("--output", type=Path, help="Write extracted contract here (default: stdout)")
    parser.add_argument(
        "--advisory-judge",
        action="store_true",
        help="Request a non-gating LLM review after deterministic evaluation",
    )
    args = parser.parse_args()
    if args.extract_session:
        if not args.channel:
            parser.error("--channel is required with --extract-session")
        contract = extract_session(args.extract_session.expanduser(), args.channel)
        rendered = json.dumps(contract, indent=2, sort_keys=True) + "\n"
        if args.output:
            args.output.write_text(rendered, encoding="utf-8")
            print(args.output)
        else:
            print(rendered, end="")
        return 0
    paths = args.fixtures or sorted(DEFAULT_FIXTURES.glob("*.json"))
    if not paths:
        parser.error("no transcript fixtures found")

    reports: list[dict] = []
    all_ok = True
    for path in paths:
        ok, report = run_fixture(path)
        reports.append(report)
        all_ok = all_ok and ok
        if not args.json:
            print(f"{'PASS' if ok else 'FAIL'} {path.name}")
            for name, value in sorted(report["metrics"].items()):
                print(f"  {name}: {value:.3f}")
            for failure in report["failures"]:
                print(f"  - {failure}")
    if args.json:
        print(json.dumps(reports, indent=2, sort_keys=True))
    if args.advisory_judge:
        advisory_judge(reports)
    return 0 if all_ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
