#!/usr/bin/env bash
# Dump hub memory diagnostics (requires hub started with NEURAL_JUNKIE_DEBUG=1).
# Usage: NEURAL_JUNKIE_HUB_URL=http://127.0.0.1:18765 ./scripts/dump-hub-memory.sh
set -euo pipefail
BASE="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
curl -sS "${BASE%/}/api/debug/hub-memory" | python3 -m json.tool
