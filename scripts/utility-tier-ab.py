#!/usr/bin/env python3
"""Compare utility-tier candidates (qwen3.5:9b vs gemma3:12b) for classifier + hub smoke.

Usage:
  ./scripts/utility-tier-ab.py                    # rules baseline + hub classify if debug hub up
  HUB=http://127.0.0.1:18765 ./scripts/utility-tier-ab.py --live

Writes: docs/testing/utility-tier-ab-<stamp>.md
"""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = ROOT / "scripts"
DOCS = ROOT / "docs" / "testing"

CLASSIFIER_QUERIES = [
    ("security oauth", "security"),
    ("Fix React component state bug", "frontend"),
    ("Design REST API endpoint for users", "backend"),
    ("fix typo in README", "cheap"),
    ("Explain this protein pathway", "biology"),
    ("Kubernetes deployment rollout", "devops"),
]


def stamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")


def run_go_routing_tests() -> tuple[bool, str]:
    proc = subprocess.run(
        ["go", "test", "./internal/routing/...", "-count=1"],
        cwd=ROOT,
        capture_output=True,
        text=True,
    )
    tail = (proc.stdout + proc.stderr)[-2000:]
    return proc.returncode == 0, tail


def fetch_routing_classify(hub: str, q: str, agent_type: str = "", *, timeout: float = 3.0) -> dict | None:
    qs = urllib.parse.urlencode({"q": q, "agent_type": agent_type})
    url = f"{hub.rstrip('/')}/api/debug/routing-classify?{qs}"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        if e.code == 404:
            return None
        return None
    except (urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        return None


def hub_classify_matrix(hub: str) -> list[dict]:
    rows: list[dict] = []
    for q, label in CLASSIFIER_QUERIES:
        dec = fetch_routing_classify(hub, q)
        rows.append(
            {
                "query": q,
                "label": label,
                "ok": dec is not None,
                "decision": dec,
            }
        )
    return rows


def main() -> int:
    p = argparse.ArgumentParser(description="Utility tier A/B report generator")
    p.add_argument("--hub", default=os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765"))
    p.add_argument("--live", action="store_true", help="Require live hub debug classify")
    args = p.parse_args()

    st = stamp()
    out_path = DOCS / f"utility-tier-ab-{st}.md"

    rules_ok, rules_tail = run_go_routing_tests()
    live_rows = hub_classify_matrix(args.hub) if args.live or True else []
    live_ok = all(r["ok"] for r in live_rows) if live_rows else False
    hub_reachable = live_rows and live_rows[0]["ok"]

    lines = [
        f"# Utility tier A/B — {st} UTC",
        "",
        "Candidates: `qwen3.5:9b` (current `UtilityOllamaModel`) vs `gemma3:12b` (release smoke co-winner).",
        "",
        "## Release smoke benchmark (reference)",
        "",
        "- `gemma3:12b`: 5/5 in `model-benchmark-release-2026-06-21-0301` (3m20s)",
        "- `qwen2.5-coder:14b`: 5/5 (coding specialists — not utility)",
        "- `qwen3.5:9b`: not in release suite; quick suite often 4/5",
        "",
        "## Rules classifier baseline (model-agnostic)",
        "",
        f"- `go test ./internal/routing/...`: **{'PASS' if rules_ok else 'FAIL'}**",
        "",
        "```text",
        rules_tail.strip() or "(no output)",
        "```",
        "",
        "## Live LLM classifier (`GET /api/debug/routing-classify`)",
        "",
        f"- Hub: `{args.hub}`",
        f"- Reachable: **{'yes' if hub_reachable else 'no'}** (needs `NEURAL_JUNKIE_DEBUG=1` on hub)",
        "",
    ]

    if live_rows:
        lines.append("| Query | Expected | OK | domain | cost_tier | source |")
        lines.append("|-------|----------|----|--------|-----------|--------|")
        for row in live_rows:
            dec = row.get("decision") or {}
            lines.append(
                f"| {row['query'][:40]} | {row['label']} | {'✓' if row['ok'] else '✗'} | "
                f"{dec.get('domain', '—')} | {dec.get('cost_tier', '—')} | {dec.get('source', '—')} |"
            )
        lines.append("")
    else:
        lines.append("_No live classify results._")
        lines.append("")

    lines.extend(
        [
            "## Decision (manual)",
            "",
            "| Outcome | Action |",
            "|---------|--------|",
            "| Gemma ≥ Qwen on classifier + tools | Switch `UtilityOllamaModel` → `gemma3:12b` |",
            "| Gemma wins smoke only | Keep Qwen utility; document Gemma as optional Assistant upgrade |",
            "| Mixed | Split roles if config supports |",
            "",
            "## Next steps for full A/B",
            "",
            "1. Run hub with `NEURAL_JUNKIE_DEBUG=1` and Ollama models pulled.",
            "2. Temporarily set `ClassifierModel` / Assistant model to each candidate; rerun this script.",
            "3. Compare session summary quality and Bio/CAD tool loop manually or via harness.",
            "",
        ]
    )

    out_path.write_text("\n".join(lines), encoding="utf-8")
    print(f"Wrote {out_path}")
    if args.live and not hub_reachable:
        print("Live hub classify unavailable", file=sys.stderr)
        return 1
    if not rules_ok:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
