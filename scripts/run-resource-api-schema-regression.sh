#!/usr/bin/env bash
# Run resource-api schema regression scenario (mirrors stuck collab fixes).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
REPO="${NEURAL_JUNKIE_SCENARIO_REPO:-$ROOT/scenarios/fixtures/minimal-repo}"

export NEURAL_JUNKIE_SCENARIO_REPO="$REPO"
export NJ_SCENARIO_ALLOW_FILE_FALLBACK=1

echo "Hub: $HUB"
echo "Workspace repo: $REPO"
if [[ ! -d "$REPO" ]]; then
  echo "ERROR: workspace not found: $REPO" >&2
  exit 1
fi

if ! curl -sf "$HUB/api/health" | grep -q '"status":"ok"'; then
  echo "ERROR: hub not healthy at $HUB (run: make build && make stop && make start-all)" >&2
  exit 1
fi

echo "Cancelling active collaborations on collab-scenarios..."
python3 << PY
import json, os, urllib.request
base = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")
req = urllib.request.Request(f"{base}/api/collaborations?include_terminal=false")
with urllib.request.urlopen(req, timeout=30) as r:
    collabs = json.load(r)
for c in collabs:
    if c.get("phase") in ("completed", "cancelled"):
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
    print(f"  cancelled {cid[:8]} ({c.get('phase')})")
PY

exec python3 "$ROOT/scripts/collab-scenarios.py" --scenario resource-api-schema-regression --verbose "$@"
