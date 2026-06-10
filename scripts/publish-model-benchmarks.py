#!/usr/bin/env python3
"""Merge model-benchmark JSON runs from docs/testing into docs/data/model-benchmarks.json for the website."""

from __future__ import annotations

import argparse
import json
import re
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TESTING_DIR = ROOT / "docs" / "testing"
DATA_PATH = ROOT / "docs" / "data" / "model-benchmarks.json"
RUN_RE = re.compile(r"^model-benchmark-(.+)-(\d{4}-\d{2}-\d{2}-\d{4})\.json$")

DEFAULT_CATALOG: dict = {
    "about": (
        "Live Neural Junkie scenario benchmarks on local Ollama models. "
        "Each run switches all in-process agents to one model tag, then executes "
        "the same implement + chat scenarios from scenarios/implement/ and scenarios/chat/."
    ),
    "methodology_url": "https://github.com/camronwood/neural-junkie/blob/main/docs/testing/MODEL_BENCHMARK.md",
    "suites": {
        "quick": {
            "description": "Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on 14B class).",
            "implement_scenarios": [
                "go-handler",
                "theme-toggle",
                "ask-mode-no-write",
            ],
            "chat_scenarios": [
                "dm-backend-workspace",
                "dm-backend-echo-followup",
            ],
        },
    },
    "runs": [],
}


def load_catalog() -> dict:
    if DATA_PATH.is_file():
        with DATA_PATH.open(encoding="utf-8") as f:
            return json.load(f)
    return json.loads(json.dumps(DEFAULT_CATALOG))


def run_id_from_path(path: Path) -> str | None:
    m = RUN_RE.match(path.name)
    if not m:
        return None
    return f"{m.group(1)}-{m.group(2)}"


def normalize_run(raw: dict, *, run_id: str, source_file: str) -> dict:
    return {
        "id": run_id,
        "suite": raw.get("suite") or "quick",
        "description": raw.get("description") or "",
        "generated_at": raw.get("generated_at") or "",
        "hub": raw.get("hub") or "",
        "hardware_note": raw.get("hardware_note") or "",
        "implement_scenarios": raw.get("implement_scenarios") or [],
        "chat_scenarios": raw.get("chat_scenarios") or [],
        "results": raw.get("results") or [],
        "source_file": source_file,
    }


def discover_runs(testing_dir: Path) -> list[tuple[str, Path]]:
    out: list[tuple[str, Path]] = []
    for path in sorted(testing_dir.glob("model-benchmark-*-*.json")):
        rid = run_id_from_path(path)
        if rid:
            out.append((rid, path))
    return out


def publish(*, testing_dir: Path, data_path: Path, only: str | None = None) -> int:
    catalog = load_catalog()
    if "suites" not in catalog:
        catalog["suites"] = DEFAULT_CATALOG["suites"]
    if "about" not in catalog:
        catalog["about"] = DEFAULT_CATALOG["about"]

    existing_ids = {r.get("id") for r in catalog.get("runs") or [] if isinstance(r, dict)}
    added = 0

    for run_id, path in discover_runs(testing_dir):
        if only and run_id != only and path.name != only:
            continue
        if run_id in existing_ids:
            continue
        with path.open(encoding="utf-8") as f:
            raw = json.load(f)
        catalog.setdefault("runs", []).append(
            normalize_run(raw, run_id=run_id, source_file=f"testing/{path.name}")
        )
        existing_ids.add(run_id)
        added += 1
        print(f"  + {run_id} from {path.name}")

    catalog["runs"] = sorted(
        catalog.get("runs") or [],
        key=lambda r: r.get("generated_at") or "",
        reverse=True,
    )
    catalog["updated_at"] = datetime.now(timezone.utc).isoformat()

    data_path.parent.mkdir(parents=True, exist_ok=True)
    data_path.write_text(json.dumps(catalog, indent=2) + "\n", encoding="utf-8")
    print(f"Published {added} new run(s) → {data_path} ({len(catalog['runs'])} total)")
    return added


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--testing-dir", type=Path, default=TESTING_DIR)
    p.add_argument("--out", type=Path, default=DATA_PATH)
    p.add_argument("--only", help="Import a single run id or filename")
    args = p.parse_args()
    publish(testing_dir=args.testing_dir, data_path=args.out, only=args.only)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
