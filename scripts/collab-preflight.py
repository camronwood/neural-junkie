#!/usr/bin/env python3
"""Fail-fast checks before live collab scenario sweeps."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts"))

from lib import collab_hub as hub  # noqa: E402

DEFAULT_ROSTER = [
    "Assistant",
    "BackendEngineer",
    "SoftwareArchitect",
    "PlatformEngineer",
]
GEMINI_SCENARIOS = ("resource-api-schema-planning",)
EXPECTED_SCENARIO_COUNT = 18
HUB_LOG = Path("/tmp/nj-hub.log")
OLLAMA_TAGS_URL = "http://127.0.0.1:11434/api/tags"


def _fail(msg: str) -> None:
    print(f"FAIL: {msg}", file=sys.stderr)


def _warn(msg: str) -> None:
    print(f"WARN: {msg}", file=sys.stderr)


def _ok(msg: str) -> None:
    print(f"OK: {msg}")


def load_expected_ollama_models() -> list[str]:
    env_local = ROOT / "env.local"
    models: list[str] = []
    if not env_local.is_file():
        return models
    for line in env_local.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("OLLAMA_MODEL=") or line.startswith("OLLAMA_CODE_MODEL="):
            _, _, val = line.partition("=")
            val = val.strip().strip('"').strip("'")
            if val and val not in models:
                models.append(val)
    return models


def check_hub(base: str) -> bool:
    try:
        health = hub.check_health(base)
    except urllib.error.URLError as exc:
        _fail(f"hub not reachable at {base} ({exc}); start: make server-regression")
        return False
    if not health:
        _fail(f"hub not healthy at {base} (start: make server-regression)")
        return False
    _ok(f"hub healthy at {base}")
    return True


def check_regression_log_hint() -> None:
    if not HUB_LOG.is_file():
        _warn(
            f"{HUB_LOG} missing — cannot verify regression env; use make server-regression"
        )
        return
    try:
        tail = HUB_LOG.read_text(encoding="utf-8", errors="replace")[-8000:]
    except OSError as exc:
        _warn(f"could not read {HUB_LOG}: {exc}")
        return
    has_rate = "NEURAL_JUNKIE_RATE_LIMIT=0" in tail or "RATE_LIMIT=0" in tail
    has_debug = "NEURAL_JUNKIE_DEBUG=1" in tail or "DEBUG=1" in tail
    if has_rate and has_debug:
        _ok("hub log suggests regression env (RATE_LIMIT=0, DEBUG=1)")
    else:
        _warn(
            "hub log may not be from make server-regression "
            "(expected NEURAL_JUNKIE_RATE_LIMIT=0 and NEURAL_JUNKIE_DEBUG=1 in /tmp/nj-hub.log)"
        )


def check_ollama(expected_models: list[str]) -> bool:
    try:
        with urllib.request.urlopen(OLLAMA_TAGS_URL, timeout=5) as resp:
            data = json.loads(resp.read().decode())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
        _fail(f"Ollama not reachable at 11434 ({exc}); run: ollama serve")
        return False

    names = {
        (m.get("name") or "").strip()
        for m in (data.get("models") or [])
        if isinstance(m, dict)
    }
    _ok(f"Ollama reachable ({len(names)} model(s) installed)")
    missing = [m for m in expected_models if m and m not in names]
    if missing:
        _warn(f"env.local models not pulled: {', '.join(missing)} — run: ollama pull <model>")
    return True


def check_agents(base: str, *, require_gemini: bool) -> bool:
    ok, missing = hub.verify_agents_online(base, DEFAULT_ROSTER)
    if not ok:
        _fail(f"agents offline: {', '.join(missing)}")
        return False
    _ok(f"default roster online: {', '.join(DEFAULT_ROSTER)}")

    if require_gemini:
        gok, gmissing = hub.verify_agents_online(base, ["Gemini"])
        if not gok:
            _fail("Gemini agent offline (required for resource-api-schema-planning)")
            return False
        _ok("Gemini online")
    return True


def check_scenario_list() -> bool:
    proc = subprocess.run(
        [sys.executable, str(ROOT / "scripts" / "collab-scenarios.py"), "--list"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        _fail(f"collab-scenarios.py --list failed: {proc.stderr.strip()}")
        return False
    names = [ln.strip() for ln in proc.stdout.splitlines() if ln.strip()]
    if len(names) != EXPECTED_SCENARIO_COUNT:
        _fail(f"expected {EXPECTED_SCENARIO_COUNT} scenarios, got {len(names)}")
        return False
    _ok(f"{len(names)} collab scenarios listed")
    return True


def main() -> int:
    p = argparse.ArgumentParser(description="Preflight for live collab sweeps")
    p.add_argument(
        "--hub",
        default=os.environ.get("NEURAL_JUNKIE_HUB_URL", hub.DEFAULT_HUB),
        help="Hub base URL",
    )
    p.add_argument(
        "--require-gemini",
        action="store_true",
        help="Require Gemini agent (resource-api-schema-planning)",
    )
    args = p.parse_args()
    base = args.hub.rstrip("/")

    print(f"Collab preflight → {base}")
    checks = [
        check_hub(base),
        check_ollama(load_expected_ollama_models()),
        check_agents(base, require_gemini=args.require_gemini),
        check_scenario_list(),
    ]
    check_regression_log_hint()

    if all(checks):
        print("Preflight passed.")
        return 0
    print("Preflight failed.", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
