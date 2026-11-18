#!/usr/bin/env bash
# Deploy nj-presence to Cloudflare Workers (free *.workers.dev HTTPS + KV).
#
# Prerequisites:
#   npm / npx
#   npx wrangler login   # once
#
# Usage:
#   ./scripts/deploy-presence-cloudflare.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKER_DIR="${ROOT}/workers/presence"
WRANGLER_TOML="${WORKER_DIR}/wrangler.toml"

if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required" >&2
  exit 1
fi

cd "$WORKER_DIR"
if [[ ! -d node_modules ]]; then
  echo "==> npm install"
  npm install
fi

# Ensure a real KV namespace id is wired (placeholder zeros are not deployable).
if grep -q 'id = "00000000000000000000000000000000"' "$WRANGLER_TOML"; then
  echo "==> creating Cloudflare KV namespace PRESENCE_KV"
  CREATE_OUT="$(npx wrangler kv namespace create PRESENCE_KV 2>&1 || true)"
  echo "$CREATE_OUT"
  KV_ID="$(echo "$CREATE_OUT" | grep -Eo 'id = "[a-f0-9]{32}"' | head -1 | sed -E 's/.*id = "([a-f0-9]{32})".*/\1/' || true)"
  if [[ -z "$KV_ID" ]]; then
    KV_ID="$(echo "$CREATE_OUT" | grep -Eo '[a-f0-9]{32}' | head -1 || true)"
  fi
  if [[ -z "$KV_ID" ]]; then
    echo "Could not parse KV namespace id. Create manually and set [[kv_namespaces]].id in wrangler.toml" >&2
    exit 1
  fi
  # Portable in-place replace for macOS/Linux
  if sed --version >/dev/null 2>&1; then
    sed -i "s/id = \"00000000000000000000000000000000\"/id = \"${KV_ID}\"/" "$WRANGLER_TOML"
    sed -i "s/preview_id = \"00000000000000000000000000000000\"/preview_id = \"${KV_ID}\"/" "$WRANGLER_TOML"
  else
    sed -i '' "s/id = \"00000000000000000000000000000000\"/id = \"${KV_ID}\"/" "$WRANGLER_TOML"
    sed -i '' "s/preview_id = \"00000000000000000000000000000000\"/preview_id = \"${KV_ID}\"/" "$WRANGLER_TOML"
  fi
  echo "Wrote KV id ${KV_ID} into wrangler.toml"
fi

echo "==> wrangler deploy"
set +e
DEPLOY_OUT="$(npm run deploy 2>&1)"
DEPLOY_STATUS=$?
set -e
echo "$DEPLOY_OUT"
if [[ $DEPLOY_STATUS -ne 0 ]]; then
  echo "" >&2
  echo "Deploy failed. For first-time setup run interactively:" >&2
  echo "  cd workers/presence && npx wrangler login && npm run deploy" >&2
  echo "Or set CLOUDFLARE_API_TOKEN (see workers/presence/README.md)." >&2
  exit "$DEPLOY_STATUS"
fi

PRESENCE_BASE="$(echo "$DEPLOY_OUT" | grep -Eo 'https://[a-zA-Z0-9._-]+\.workers\.dev' | head -1 || true)"
PRESENCE_BASE="${PRESENCE_BASE%/}"

echo ""
echo "Presence worker deployed (or updated)."
if [[ -n "$PRESENCE_BASE" ]]; then
  echo "  Base: ${PRESENCE_BASE}"
  echo ""
  echo "Desktop release builds:"
  echo "  export VITE_PRESENCE_URL=${PRESENCE_BASE}"
  echo ""
  echo "Set stats token (once):"
  echo "  cd workers/presence && npx wrangler secret put PRESENCE_STATS_TOKEN"
  echo ""
  echo "Smoke:"
  echo "  curl -s ${PRESENCE_BASE}/healthz"
  echo "  curl -s -X POST ${PRESENCE_BASE}/v1/ping -H 'content-type: application/json' -d '{\"id\":\"550e8400-e29b-41d4-a716-446655440000\"}'"
  echo "  curl -s -H \"Authorization: Bearer \$PRESENCE_STATS_TOKEN\" ${PRESENCE_BASE}/v1/stats"
fi
