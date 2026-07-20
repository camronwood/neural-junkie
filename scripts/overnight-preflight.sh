#!/usr/bin/env bash
# Afternoon preflight before make overnight (release-prep).
# Fails fast on missing models, hub boot, Arena pack, and release-prep-ready smoke.
#
# Usage:
#   ./scripts/overnight-preflight.sh
#   make overnight-preflight
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

HUB_URL="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
SUITE="${BENCHMARK_SUITE:-release}"

# shellcheck disable=SC1091
source load-env.sh
export NEURAL_JUNKIE_RATE_LIMIT=0
export NEURAL_JUNKIE_HUB_URL="${HUB_URL}"
export BENCHMARK_SUITE="${SUITE}"

echo "=== overnight-preflight (suite=${SUITE}) ==="
echo ""

echo ">>> 1/4 Ollama models (warm + suite roster; pull missing)"
python3 scripts/ensure-ollama-models-ready.py \
  --suite "${SUITE}" \
  --warm \
  --smoke \
  --pull-missing \
  --keep-alive "${NJ_OVERNIGHT_KEEP_ALIVE:-24h}"

echo ""
echo ">>> 2/4 Boot regression hub + release-prep-ready smoke"
python3 <<PY
import sys
from pathlib import Path
sys.path.insert(0, "scripts")
from lib.regression_boot import boot_regression_stack
ok = boot_regression_stack(
    Path("."),
    "${HUB_URL}",
    label="overnight-preflight",
    clean=True,
    ready_smoke=True,
)
raise SystemExit(0 if ok else 1)
PY

echo ""
echo ">>> 3/4 Model Arena pack (HTTP challenges must be 200)"
python3 <<PY
import sys
sys.path.insert(0, "scripts")
from lib.arena_pack import ensure_model_arena_pack
ok, detail = ensure_model_arena_pack("${HUB_URL}")
print(detail)
raise SystemExit(0 if ok else 1)
PY

echo ""
echo ">>> 4/4 Arena logic-set smoke (one challenge round-trip)"
python3 <<PY
import sys
sys.path.insert(0, "scripts")
from lib import collab_hub as hub
code, data = hub.hub_request("${HUB_URL}", "GET", "/api/arena/challenges")
if code != 200:
    print(f"FAIL: GET /api/arena/challenges -> {code} {data}", file=sys.stderr)
    raise SystemExit(1)
challenges = (data or {}).get("challenges") if isinstance(data, dict) else None
n = len(challenges) if isinstance(challenges, list) else 0
print(f"OK: arena challenges listed ({n})")
PY

echo ""
echo "READY for overnight:"
echo "  make overnight NJ_OVERNIGHT_TARGET=release-prep"
echo "  # models already local — default NO_PULL=1 is fine"
echo "  # if you change machines: make overnight PULL=1"
