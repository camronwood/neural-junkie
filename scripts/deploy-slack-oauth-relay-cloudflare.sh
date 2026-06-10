#!/usr/bin/env bash
# Deploy nj-slack-oauth-relay to Cloudflare Workers (free *.workers.dev HTTPS).
#
# Prerequisites:
#   npm / npx
#   npx wrangler login   # once
#
# Usage:
#   ./scripts/deploy-slack-oauth-relay-cloudflare.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKER_DIR="${ROOT}/workers/slack-oauth-relay"

if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required" >&2
  exit 1
fi

cd "$WORKER_DIR"
if [[ ! -d node_modules ]]; then
  echo "==> npm install"
  npm install
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
  echo "  cd workers/slack-oauth-relay && npx wrangler login && npm run deploy" >&2
  echo "Or set CLOUDFLARE_API_TOKEN (see workers/slack-oauth-relay/README.md)." >&2
  exit "$DEPLOY_STATUS"
fi

# wrangler prints: https://nj-slack-oauth-relay.<account>.workers.dev
RELAY_BASE="$(echo "$DEPLOY_OUT" | grep -Eo 'https://[a-zA-Z0-9._-]+\.workers\.dev' | head -1 || true)"
if [[ -z "$RELAY_BASE" ]]; then
  RELAY_BASE="https://nj-slack-oauth-relay.$(npx wrangler whoami 2>/dev/null | grep -Eo '[a-z0-9-]+\.workers\.dev' | head -1 || echo 'YOUR_SUBDOMAIN.workers.dev')"
  RELAY_BASE="https://nj-slack-oauth-relay.${RELAY_BASE#https://}"
fi
RELAY_BASE="${RELAY_BASE%/}"

echo ""
echo "Relay deployed (or updated)."
echo "  Relay base: ${RELAY_BASE}"
echo ""
echo "Slack app → OAuth & Permissions → Redirect URLs:"
echo "  ${RELAY_BASE}/api/slack/oauth/callback"
echo "  ${RELAY_BASE}/api/slack/oauth/user-dm/callback"
echo ""
echo "Then set NJ vendor / CI:"
echo "  export SLACK_VENDOR_OAUTH_RELAY_BASE=${RELAY_BASE}"
echo "  gh secret set SLACK_VENDOR_OAUTH_RELAY_BASE --repo camronwood/neural-junkie"
echo ""
echo "Smoke test:"
echo "  curl -s ${RELAY_BASE}/healthz"
