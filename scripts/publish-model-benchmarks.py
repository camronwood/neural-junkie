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
PROFILES_PATH = ROOT / "docs" / "data" / "model-capability-profiles.json"
EMBED_PROFILES_PATH = ROOT / "internal" / "routing" / "capabilities" / "data" / "capability_profiles.json"
RUN_RE = re.compile(r"^model-benchmark-(.+)-(\d{4}-\d{2}-\d{2}-\d{4})\.json$")

import sys

sys.path.insert(0, str(ROOT / "scripts"))
from lib.model_benchmark import (  # noqa: E402
    build_scenario_catalog,
    derive_capability_profiles,
    load_models,
    load_scenario_meta,
    write_capability_profiles,
)

DEFAULT_CATALOG: dict = {
    "about": (
        "Live Neural Junkie scenario benchmarks on local Ollama models. "
        "Each run switches all in-process agents to one model tag, then executes "
        "the same implement + chat scenarios from scenarios/implement/ and scenarios/chat/."
    ),
    "methodology_url": "https://github.com/camronwood/neural-junkie/blob/main/docs/testing/MODEL_BENCHMARK.md",
    "suites": {
        "quick": {
            "description": "Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).",
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
    implement = raw.get("implement_scenarios") or []
    chat = raw.get("chat_scenarios") or []
    hardware = raw.get("hardware") if isinstance(raw.get("hardware"), dict) else {}
    hw_note = str(raw.get("hardware_note") or "").strip()
    if not hw_note and hardware.get("total_memory_gb"):
        hw_note = f"{hardware['total_memory_gb']} GB RAM ({hardware.get('tier', '?')} tier)"
    catalog = raw.get("scenario_catalog")
    if not isinstance(catalog, list) or not catalog:
        catalog = build_scenario_catalog(
            [str(x) for x in implement if str(x).strip()],
            [str(x) for x in chat if str(x).strip()],
        )
    return {
        "id": run_id,
        "suite": raw.get("suite") or "quick",
        "description": raw.get("description") or "",
        "generated_at": raw.get("generated_at") or "",
        "hub": raw.get("hub") or "",
        "hardware": hardware,
        "hardware_note": hw_note,
        "judge_provider": str(raw.get("judge_provider") or "").strip(),
        "scenario_catalog": catalog,
        "implement_scenarios": implement,
        "chat_scenarios": chat,
        "results": raw.get("results") or [],
        "source_file": source_file,
    }


def build_scenario_index() -> dict[str, dict]:
    """All scenario metadata keyed by kind/name for website client-side enrichment."""
    index: dict[str, dict] = {}
    for kind, subdir in (("implement", "implement"), ("chat", "chat")):
        scenario_dir = ROOT / "scenarios" / subdir
        if not scenario_dir.is_dir():
            continue
        for path in sorted(scenario_dir.glob("*.json")):
            meta = load_scenario_meta(kind, path.stem)
            index[f"{kind}/{path.stem}"] = meta
    return index


def refresh_run_metadata(run: dict) -> dict:
    """Fill scenario catalog and hardware note for runs imported before enrichment existed."""
    out = dict(run)
    implement = [str(x) for x in (out.get("implement_scenarios") or []) if str(x).strip()]
    chat = [str(x) for x in (out.get("chat_scenarios") or []) if str(x).strip()]
    if not out.get("scenario_catalog"):
        out["scenario_catalog"] = build_scenario_catalog(implement, chat)
    hardware = out.get("hardware") if isinstance(out.get("hardware"), dict) else {}
    if not out.get("hardware_note") and hardware.get("total_memory_gb"):
        out["hardware_note"] = f"{hardware['total_memory_gb']} GB RAM ({hardware.get('tier', '?')} tier)"
    judged = [s for s in (out.get("scenario_catalog") or []) if isinstance(s, dict) and s.get("llm_judge")]
    if judged and not str(out.get("judge_provider") or "").strip():
        out["judge_provider"] = "gemini"
    return out


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
    catalog["runs"] = [refresh_run_metadata(r) for r in catalog.get("runs") or [] if isinstance(r, dict)]
    catalog["scenario_index"] = build_scenario_index()
    catalog["updated_at"] = datetime.now(timezone.utc).isoformat()

    data_path.parent.mkdir(parents=True, exist_ok=True)
    data_path.write_text(json.dumps(catalog, indent=2) + "\n", encoding="utf-8")
    print(f"Published {added} new run(s) → {data_path} ({len(catalog['runs'])} total)")

    try:
        profiles = derive_capability_profiles(catalog, load_models())
        write_capability_profiles(
            profiles,
            docs_path=PROFILES_PATH,
            embed_path=EMBED_PROFILES_PATH,
        )
        print(
            f"  capability profiles → {PROFILES_PATH.name} "
            f"(source {profiles.get('source_run_id')})"
        )
    except ValueError as exc:
        print(f"  capability profiles skipped: {exc}")

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
