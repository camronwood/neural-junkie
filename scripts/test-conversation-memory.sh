#!/usr/bin/env bash
# Smoke-test conversation memory index + retrieval API.
# Usage: ./scripts/test-conversation-memory.sh [hub_base_url]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASE="${1:-http://127.0.0.1:8080}"

echo "== memory stats =="
curl -sf "${BASE}/api/memory/stats" | python3 -m json.tool

echo "== memory query (channel required for meaningful results) =="
CHANNEL="${TEST_MEMORY_CHANNEL:-general}"
QUERY="${TEST_MEMORY_QUERY:-auth middleware JWT}"
curl -sf "${BASE}/api/memory/query?channel=${CHANNEL}&q=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "${QUERY}")&limit=5" \
  | python3 -m json.tool

echo "OK — check non-empty results after seeding ${CHANNEL} with 30+ messages referencing: ${QUERY}"
