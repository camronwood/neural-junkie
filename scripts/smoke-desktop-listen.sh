#!/usr/bin/env bash
# Smoke Desktop Listen against a running hub (local/dev).
# Requires: hub on HUB_URL (default http://127.0.0.1:18765), NJ_WHISPER_CMD stub or real whisper.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HUB_URL="${HUB_URL:-http://127.0.0.1:18765}"
STUB="${ROOT}/scripts/whisper-stub.sh"
export NJ_WHISPER_CMD="${NJ_WHISPER_CMD:-$STUB}"

echo "== listen status =="
curl -sf "${HUB_URL}/api/assistant/listen/status" | tee /tmp/nj-listen-status.json
echo

echo "== listen start =="
START=$(curl -sf -X POST "${HUB_URL}/api/assistant/listen/start" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Smoke listen"}')
echo "$START"
SESSION=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["session_id"])' <<<"$START")

echo "== listen chunk =="
# Minimal WAV-like payload (stub ignores content)
printf 'RIFF....WAVEfmt ' > /tmp/nj-listen-chunk.wav
curl -sf -X POST "${HUB_URL}/api/assistant/listen/chunk?session_id=${SESSION}" \
  -H 'Content-Type: application/octet-stream' \
  --data-binary @/tmp/nj-listen-chunk.wav | tee /tmp/nj-listen-chunk.json
echo

echo "== listen stop =="
curl -sf -X POST "${HUB_URL}/api/assistant/listen/stop" \
  -H 'Content-Type: application/json' \
  -d "{\"session_id\":\"${SESSION}\"}" | tee /tmp/nj-listen-stop.json
echo
echo "OK desktop listen smoke"
