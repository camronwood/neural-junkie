#!/usr/bin/env python3
"""CLI runner for model-benchmark Suite Arena track (logic + Connect4)."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

# Default hub port used by desktop / local arena harnesses.
DEFAULT_HUB = "http://127.0.0.1:18765"
LOGIC_MAX_STEPS = 5
CONNECT4_MAX_STEPS = 42
SCRIPTS_DIR = Path(__file__).resolve().parent


class ArenaUnavailable(Exception):
    """Raised when hub arena endpoints are not reachable or return HTTP errors."""


def _request(
    hub: str,
    method: str,
    path: str,
    body: dict[str, Any] | None = None,
    *,
    timeout: float = 300.0,
) -> dict[str, Any]:
    url = f"{hub.rstrip('/')}{path}"
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            if not raw.strip():
                return {}
            parsed = json.loads(raw)
            return parsed if isinstance(parsed, dict) else {"data": parsed}
    except urllib.error.HTTPError as exc:
        detail = ""
        try:
            detail = exc.read().decode("utf-8", errors="replace")[:240]
        except Exception:
            detail = exc.reason or str(exc)
        raise ArenaUnavailable(f"HTTP {exc.code} {path}: {detail}") from exc
    except urllib.error.URLError as exc:
        raise ArenaUnavailable(f"{path}: {exc}") from exc
    except json.JSONDecodeError as exc:
        raise ArenaUnavailable(f"invalid JSON from {path}") from exc


def _session_status(session: dict[str, Any]) -> str:
    state = session.get("state") if isinstance(session.get("state"), dict) else {}
    return str(session.get("status") or state.get("status") or "").strip().lower()


def _session_result(session: dict[str, Any]) -> str:
    state = session.get("state") if isinstance(session.get("state"), dict) else {}
    return str(session.get("result") or state.get("result") or "").strip().lower()


def _list_logic_puzzles(hub: str) -> list[dict[str, Any]]:
    data = _request(hub, "GET", "/api/arena/challenges")
    puzzles = data.get("puzzles") or []
    if isinstance(puzzles, list) and puzzles:
        return [p for p in puzzles if isinstance(p, dict)]
    # Fallback: unknown puzzle count — run a small fixed N.
    return [{"id": ""}, {"id": ""}, {"id": ""}]


def _create_session(hub: str, challenge: str, **extra: Any) -> dict[str, Any]:
    body: dict[str, Any] = {"challenge": challenge, **extra}
    return _request(hub, "POST", "/api/arena/sessions", body)


def _match_run(
    hub: str,
    session_id: str,
    *,
    provider_id: str,
    model: str,
    max_steps: int,
) -> dict[str, Any]:
    body: dict[str, Any] = {
        "session_id": session_id,
        "model": model,
        "max_steps": max_steps,
    }
    if provider_id:
        body["provider_id"] = provider_id
    return _request(hub, "POST", "/api/arena/match/run", body, timeout=600.0)


def _get_session(hub: str, session_id: str) -> dict[str, Any]:
    return _request(hub, "GET", f"/api/arena/sessions/{session_id}")


def run_logic_set(
    hub: str,
    *,
    model: str,
    provider_id: str,
) -> tuple[bool, dict[str, Any], list[str]]:
    lines: list[str] = []
    puzzles = _list_logic_puzzles(hub)
    correct = 0
    total = 0
    for idx, puzzle in enumerate(puzzles):
        puzzle_id = str(puzzle.get("id") or "").strip()
        label = puzzle_id or f"logic-{idx + 1}"
        create_body: dict[str, Any] = {
            # Sidecar defaults black="model" (placeholder); pin the real Ollama tag.
            "white": "human",
            "black": model,
        }
        if puzzle_id:
            create_body["puzzle_id"] = puzzle_id
        session = _create_session(hub, "logic", **create_body)
        session_id = str(session.get("id") or "").strip()
        if not session_id:
            lines.append(f"FAIL: {label} missing session id")
            total += 1
            continue
        try:
            final = _match_run(
                hub,
                session_id,
                provider_id=provider_id,
                model=model,
                max_steps=LOGIC_MAX_STEPS,
            )
        except ArenaUnavailable:
            # Re-fetch session in case match failed after answer was recorded.
            final = _get_session(hub, session_id)
            status = _session_status(final)
            if status not in {"correct", "incorrect"}:
                raise
        status = _session_status(final)
        total += 1
        if status == "correct":
            correct += 1
            lines.append(f"PASS: {label} correct")
        else:
            lines.append(f"FAIL: {label} status={status or 'unknown'}")

    rate = (correct / total) if total else 0.0
    ok = total > 0 and rate >= 0.5
    metrics = {
        "scenario": "logic-set",
        "kind": "arena",
        "model": model,
        "provider_id": provider_id or None,
        "prompt_tokens": None,
        "completion_tokens": None,
        "logic_correct": correct,
        "logic_total": total,
        "wins": 0,
        "losses": 0,
        "draws": 0,
        "illegal_moves": 0,
        "pass_rate": round(rate, 4),
        "capability_passed": ok,
    }
    return ok, metrics, lines


def run_connect4_smoke(
    hub: str,
    *,
    model: str,
    provider_id: str,
) -> tuple[bool, dict[str, Any], list[str]]:
    lines: list[str] = []
    session = _create_session(hub, "connect4")
    session_id = str(session.get("id") or "").strip()
    if not session_id:
        lines.append("FAIL: connect4-smoke missing session id")
        metrics = {
            "scenario": "connect4-smoke",
            "kind": "arena",
            "model": model,
            "provider_id": provider_id or None,
            "prompt_tokens": None,
            "completion_tokens": None,
            "logic_correct": 0,
            "logic_total": 0,
            "wins": 0,
            "losses": 0,
            "draws": 0,
            "illegal_moves": 0,
            "pass_rate": 0.0,
            "capability_passed": False,
            "connect4_completed": False,
        }
        return False, metrics, lines

    final = _match_run(
        hub,
        session_id,
        provider_id=provider_id,
        model=model,
        max_steps=CONNECT4_MAX_STEPS,
    )
    # Prefer fresh GET for final status.
    try:
        final = _get_session(hub, session_id)
    except ArenaUnavailable:
        pass

    status = _session_status(final)
    result = _session_result(final)
    completed = status in {"finished", "draw"} or result == "draw"
    wins = losses = draws = 0
    if status == "draw" or result == "draw":
        draws = 1
    elif completed:
        # Model played both seats; treat terminal game as a completed capability check.
        wins = 1 if result in {"red", "yellow", "white", "black"} else 0
        if wins == 0 and completed:
            draws = 1

    if completed:
        lines.append(f"PASS: connect4-smoke status={status} result={result or 'n/a'}")
    else:
        lines.append(f"FAIL: connect4-smoke status={status or 'unknown'} result={result or 'n/a'}")

    metrics = {
        "scenario": "connect4-smoke",
        "kind": "arena",
        "model": model,
        "provider_id": provider_id or None,
        "prompt_tokens": None,
        "completion_tokens": None,
        "logic_correct": 0,
        "logic_total": 0,
        "wins": wins,
        "losses": losses,
        "draws": draws,
        "illegal_moves": 0,
        "pass_rate": 1.0 if completed else 0.0,
        "capability_passed": completed,
        "connect4_completed": completed,
    }
    return completed, metrics, lines


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--hub", default=DEFAULT_HUB, help="Hub base URL")
    p.add_argument(
        "--scenario",
        required=True,
        help="logic-set | connect4-smoke | comma-separated list",
    )
    p.add_argument("--model", required=True, help="Model tag / id for match/run")
    p.add_argument(
        "--provider-id",
        default="",
        help='Provider id (default empty = hub default; often "ollama")',
    )
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    scenarios = [s.strip() for s in str(args.scenario).split(",") if s.strip()]
    if not scenarios:
        print("FAIL: --scenario required", file=sys.stderr)
        return 1

    all_ok = True
    combined: dict[str, Any] = {
        "kind": "arena",
        "model": args.model,
        "provider_id": args.provider_id or None,
        "prompt_tokens": None,
        "completion_tokens": None,
        "logic_correct": 0,
        "logic_total": 0,
        "wins": 0,
        "losses": 0,
        "draws": 0,
        "illegal_moves": 0,
        "scenarios": [],
        "capability_passed": False,
        "pass_rate": 0.0,
    }

    try:
        # Probe arena availability early.
        _request(args.hub, "GET", "/api/arena/challenges", timeout=10.0)

        for name in scenarios:
            if name == "logic-set":
                ok, metrics, lines = run_logic_set(
                    args.hub, model=args.model, provider_id=args.provider_id
                )
            elif name == "connect4-smoke":
                ok, metrics, lines = run_connect4_smoke(
                    args.hub, model=args.model, provider_id=args.provider_id
                )
            else:
                print(f"FAIL: unknown scenario {name!r}")
                all_ok = False
                continue

            for line in lines:
                print(line)
            combined["scenarios"].append(metrics)
            combined["logic_correct"] += int(metrics.get("logic_correct") or 0)
            combined["logic_total"] += int(metrics.get("logic_total") or 0)
            combined["wins"] += int(metrics.get("wins") or 0)
            combined["losses"] += int(metrics.get("losses") or 0)
            combined["draws"] += int(metrics.get("draws") or 0)
            combined["illegal_moves"] += int(metrics.get("illegal_moves") or 0)
            if not ok:
                all_ok = False

        # Aggregate capability / pass_rate.
        if combined["logic_total"]:
            combined["pass_rate"] = round(
                combined["logic_correct"] / combined["logic_total"], 4
            )
        elif combined["scenarios"]:
            rates = [float(s.get("pass_rate") or 0.0) for s in combined["scenarios"]]
            combined["pass_rate"] = round(sum(rates) / len(rates), 4)
        combined["capability_passed"] = all_ok
        combined["scenario"] = ",".join(scenarios)

    except ArenaUnavailable as exc:
        print(f"FAIL: arena unavailable ({exc})")
        fail_metrics = {
            "kind": "arena",
            "model": args.model,
            "provider_id": args.provider_id or None,
            "prompt_tokens": None,
            "completion_tokens": None,
            "logic_correct": 0,
            "logic_total": 0,
            "wins": 0,
            "losses": 0,
            "draws": 0,
            "illegal_moves": 0,
            "pass_rate": 0.0,
            "capability_passed": False,
            "error": "arena unavailable",
            "detail": str(exc),
            "scenario": args.scenario,
        }
        print("METRICS_JSON:" + json.dumps(fail_metrics, separators=(",", ":")))
        return 1

    print("METRICS_JSON:" + json.dumps(combined, separators=(",", ":")))
    return 0 if all_ok else 1


if __name__ == "__main__":
    sys.exit(main())
