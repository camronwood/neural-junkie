#!/usr/bin/env bash
# Smoke-test the public Slack OAuth relay (Cloudflare Worker).
#
# Usage:
#   ./scripts/verify-slack-oauth-relay.sh
#   SLACK_OAUTH_RELAY_BASE=https://nj-slack-oauth-relay.subdomain.workers.dev ./scripts/verify-slack-oauth-relay.sh
set -euo pipefail

BASE="${SLACK_OAUTH_RELAY_BASE:-${SLACK_VENDOR_OAUTH_RELAY_BASE:-${NEURAL_JUNKIE_SLACK_OAUTH_RELAY_BASE:-}}}"
BASE="${BASE%/}"

if [[ -z "$BASE" ]]; then
  echo "Set SLACK_OAUTH_RELAY_BASE (or SLACK_VENDOR_OAUTH_RELAY_BASE)" >&2
  exit 1
fi

echo "==> healthz ${BASE}/healthz"
BODY="$(python3 -c "import urllib.request; print(urllib.request.urlopen('${BASE}/healthz', timeout=15).read().decode())")"
echo "$BODY"
echo "$BODY" | grep -q '"ok":true' || { echo "healthz failed" >&2; exit 1; }

echo "==> root"
python3 -c "import urllib.request; print(urllib.request.urlopen('${BASE}/', timeout=15).read().decode())"

echo "OK: relay reachable at ${BASE}"
