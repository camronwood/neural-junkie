#!/usr/bin/env python3
"""Run chat workspace + collab conversation quality live scenarios."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS_DIR = ROOT / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.hub_regression import wait_for_hub  # noqa: E402
from lib.fixture_cleanup import preflight_regression_run  # noqa: E402

CHAT_SCENARIOS = [
    # Workspace visibility (all specialist DMs + public)
    "dm-backend-workspace",
    "public-backend-theme-workspace",
    "dm-frontend-workspace",
    "dm-architect-workspace",
    "dm-security-workspace",
    "dm-code-reviewer-workspace",
    "dm-platform-workspace",
    "dm-database-workspace",
    # Multi-turn conversation quality
    "dm-backend-echo-followup",
    "thanks-closure",
    "already-said-closure",
    "public-frontend-theme-continuation",
    # IDE / routing
    "dm-ide-route-backend",
    # Multi-turn conversation flow (continuation, topic switch, interject)
    "dm-backend-deep-continuation",
    "dm-topic-switch",
    "dm-assistant-continue-after-closure",
    "dm-backend-interject-resume",
]

COLLAB_SCENARIOS = [
    "collab-conversation-quality-regression",
    "collab-no-edit-after-cancel",
    "collab-generation-error-resilience",
    "collab-participation-two-agent-strict",
    "collab-participation-three-agent",
    "collab-human-planning-interject",
]


def run_cmd(args: list[str]) -> int:
    print(f"\n>>> {' '.join(args)}")
    env = os.environ.copy()
    env["NEURAL_JUNKIE_RATE_LIMIT"] = "0"
    return subprocess.call(args, cwd=str(ROOT), env=env)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--verbose", action="store_true")
    parser.add_argument("--chat-only", action="store_true")
    parser.add_argument("--collab-only", action="store_true")
    args = parser.parse_args()

    hub_url = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")
    print(f"Waiting for hub at {hub_url}...")
    if not wait_for_hub(hub_url, timeout_s=120.0):
        print("FAIL: hub not reachable (start with: make server-regression)", file=sys.stderr)
        return 1
    print("OK: hub ready")
    preflight_regression_run(ROOT, hub_url, label="conversation-scenarios preflight")

    env_prefix: list[str] = []
    verbose = ["--verbose"] if args.verbose else []
    failed: list[str] = []

    if not args.collab_only:
        print("\n=== Chat workspace scenarios ===")
        for name in CHAT_SCENARIOS:
            code = run_cmd(
                ["python3", "scripts/chat-scenarios.py", "--scenario", name] + verbose
            )
            if code != 0:
                failed.append(f"chat:{name}")

    if not args.chat_only:
        print("\n=== Collab conversation scenarios ===")
        for name in COLLAB_SCENARIOS:
            code = run_cmd(
                ["python3", "scripts/collab-scenarios.py", "--scenario", name] + verbose
            )
            if code != 0:
                failed.append(f"collab:{name}")

    print("\n=== Summary ===")
    total = (0 if args.collab_only else len(CHAT_SCENARIOS)) + (
        0 if args.chat_only else len(COLLAB_SCENARIOS)
    )
    passed = total - len(failed)
    print(f"PASS {passed}/{total}")
    if failed:
        print("FAILED:", ", ".join(failed), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
