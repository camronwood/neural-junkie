#!/usr/bin/env bash
# Serial repro for collab scenarios that failed in release-prep / collab-scenarios-all.
#
# Usage:
#   ./scripts/collab-failure-repro.sh              # all known failure scenarios
#   ./scripts/collab-failure-repro.sh planning-two-agent
#   RESTART_HUB=1 ./scripts/collab-failure-repro.sh
#
# After each FAIL, inspect hub state:
#   ./scripts/debug-collab.py live --include-terminal

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
RESTART_HUB="${RESTART_HUB:-0}"
VERBOSE_FLAG=""
if [ "${VERBOSE:-}" = "1" ]; then
  VERBOSE_FLAG="VERBOSE=1"
fi

DEFAULT_SCENARIOS=(
  plan-dependency-prose-regression
  planning-two-agent
  plan-phoenix-combined-regression
  plan-findings-task-regression
  plan-distinct-deliverables-same-agent
  resource-api-schema-planning
  resource-api-schema-regression
  solo-vs-collab-parity
  collab-conversation-quality-regression
  collab-no-edit-after-cancel
)

if [ "$#" -gt 0 ]; then
  SCENARIOS=("$@")
else
  SCENARIOS=("${DEFAULT_SCENARIOS[@]}")
fi

echo "collab-failure-repro → hub=$HUB scenarios=${#SCENARIOS[@]}"

if [ "$RESTART_HUB" = "1" ]; then
  echo "restarting regression hub..."
  make stop 2>/dev/null || true
  sleep 2
  make server-regression
  for _ in $(seq 1 60); do
    if curl -sf "$HUB/api/health" >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
fi

make collab-preflight

failed=0
for scenario in "${SCENARIOS[@]}"; do
  echo ""
  echo "=== repro: $scenario ==="
  if eval "$VERBOSE_FLAG make collab-scenario SCENARIO=$scenario"; then
    echo "✓ PASS: $scenario"
  else
    echo "✗ FAIL: $scenario"
    failed=$((failed + 1))
    if [ -x "$ROOT/scripts/debug-collab.py" ]; then
      python3 "$ROOT/scripts/debug-collab.py" live --include-terminal || true
    fi
    if [ "${STOP_ON_FAIL:-1}" = "1" ]; then
      echo "Stopped on $scenario (set STOP_ON_FAIL=0 to continue)"
      exit 1
    fi
  fi
  sleep 3
done

if [ "$failed" -gt 0 ]; then
  echo "$failed scenario(s) failed"
  exit 1
fi

echo "All repro scenarios PASS"
