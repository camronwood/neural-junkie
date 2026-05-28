#!/usr/bin/env bash
# Run a full Phoenix resource-api schema collab against the local hub.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PHOENIX="${NEURAL_JUNKIE_SCENARIO_REPO:-/Users/camronwood/development/Phoenix}"
HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"

export NEURAL_JUNKIE_SCENARIO_REPO="$PHOENIX"
export NJ_SCENARIO_ALLOW_FILE_FALLBACK=1

echo "Hub: $HUB"
echo "Phoenix repo: $PHOENIX"
if [[ ! -d "$PHOENIX" ]]; then
  echo "ERROR: Phoenix path not found: $PHOENIX" >&2
  exit 1
fi

if ! curl -sf "$HUB/api/health" | grep -q '"status":"ok"'; then
  echo "ERROR: hub not healthy at $HUB (run make start-all)" >&2
  exit 1
fi

# Stuck gemini-cli from a prior collab can block the next run (394% CPU, no discussion post).
if pgrep -f "/opt/homebrew/bin/gemini" >/dev/null 2>&1; then
  echo "Stopping stuck gemini-cli processes..."
  pkill -f "/opt/homebrew/bin/gemini" 2>/dev/null || true
  sleep 2
fi

echo "Cancelling active collaborations on collab-scenarios and stuck Phoenix collabs..."
python3 << PY
import json, os, urllib.request
base = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")
req = urllib.request.Request(f"{base}/api/collaborations?include_terminal=false")
with urllib.request.urlopen(req, timeout=30) as r:
    collabs = json.load(r)
for c in collabs:
    phase = c.get("phase")
    if phase in ("completed", "cancelled"):
        continue
    cid = c["id"]
    ch = c.get("channel") or "general"
    body = json.dumps({
        "channel": ch,
        "content": f"/cancel-plan {cid[:8]}",
        "type": "question",
        "from": {"name": "CollabRunner", "type": "human"},
    }).encode()
    urllib.request.urlopen(
        urllib.request.Request(
            f"{base}/api/send",
            data=body,
            headers={"Content-Type": "application/json"},
            method="POST",
        ),
        timeout=60,
    )
    print(f"  cancelled {cid[:8]} ({phase})")
PY

echo "Running phoenix-resource-api-e2e scenario..."
exec python3 "$ROOT/scripts/collab-scenarios.py" --scenario phoenix-resource-api-e2e --verbose
