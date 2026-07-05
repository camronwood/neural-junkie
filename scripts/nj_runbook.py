#!/usr/bin/env python3
"""CLI for runbook definitions (v2.2).

Usage:
  python3 scripts/nj_runbook.py list
  python3 scripts/nj_runbook.py run health-check-alert --agent Assistant --input health_url=https://example.com/health
  python3 scripts/nj_runbook.py history health-check-alert
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.request

HUB = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")


def api(method: str, path: str, body: dict | None = None) -> dict:
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(f"{HUB}{path}", data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            raw = resp.read().decode()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        print(e.read().decode(), file=sys.stderr)
        raise SystemExit(1)


def cmd_list(_: argparse.Namespace) -> None:
    rows = api("GET", "/api/runbook-definitions")
    for row in rows:
        print(f"{row.get('id')}\t{row.get('source')}\tv{row.get('version')}\t{row.get('title')}")


def cmd_run(args: argparse.Namespace) -> None:
    agent_ids = []
    agents_resp = api("GET", "/api/agents")
    by_name = {a.get("name", "").lower(): a.get("id") for a in agents_resp if isinstance(a, dict)}
    for name in args.agent:
        aid = by_name.get(name.lower())
        if not aid:
            print(f"unknown agent: {name}", file=sys.stderr)
            raise SystemExit(1)
        agent_ids.append(aid)
    inputs = {}
    for item in args.input or []:
        if "=" not in item:
            continue
        k, v = item.split("=", 1)
        inputs[k] = v
    inst = api("POST", f"/api/runbook-definitions/{args.definition_id}/instantiate", {
        "channel": args.channel,
        "created_by": "cli",
        "agent_ids": agent_ids,
        "inputs": inputs,
    })
    collab_id = inst["collaboration_id"]
    api("POST", f"/api/runbooks/{collab_id}/submit", {})
    snap = api("POST", f"/api/runbooks/{collab_id}/start", {"inputs": inputs})
    print(json.dumps({"collaboration_id": collab_id, "phase": snap.get("phase")}, indent=2))


def cmd_history(args: argparse.Namespace) -> None:
    qs = f"?definition_id={args.definition_id}" if args.definition_id else ""
    rows = api("GET", f"/api/runbook-runs{qs}")
    for row in rows:
        print(f"#{row.get('run_number')}\t{row.get('id')[:8]}\t{row.get('phase')}\t{row.get('started_at')}")


def main() -> None:
    p = argparse.ArgumentParser(description="Neural Junkie runbook CLI")
    sub = p.add_subparsers(dest="cmd", required=True)
    sub.add_parser("list").set_defaults(func=cmd_list)
    run_p = sub.add_parser("run")
    run_p.add_argument("definition_id")
    run_p.add_argument("--agent", action="append", required=True)
    run_p.add_argument("--input", action="append")
    run_p.add_argument("--channel", default="general")
    run_p.set_defaults(func=cmd_run)
    hist_p = sub.add_parser("history")
    hist_p.add_argument("definition_id", nargs="?")
    hist_p.set_defaults(func=cmd_history)
    args = p.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
