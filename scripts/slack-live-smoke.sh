#!/usr/bin/env bash
# Optional pre-release Slack parity check (not CI). Requires a running regression hub and Slack bind.
set -euo pipefail

HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
HUB="${HUB%/}"

echo "slack-live-smoke: hub=$HUB"

if ! curl -sf "$HUB/api/health" >/dev/null; then
  echo "FAIL: hub not healthy — start with: make server-regression" >&2
  exit 1
fi

echo "Slack status:"
curl -sf "$HUB/api/slack/status" | python3 -m json.tool

if [[ -n "${SLACK_SMOKE_CHANNEL_ID:-}" ]]; then
  echo "Bound channel id: $SLACK_SMOKE_CHANNEL_ID (post a test @mention manually to verify agent replies)"
else
  echo "Set SLACK_SMOKE_CHANNEL_ID to the bound Slack channel id for a manual @Agent reply check."
fi

if [[ -z "${SLACK_BOT_TOKEN:-}" && -z "${SLACK_APP_TOKEN:-}" ]]; then
  echo "WARN: SLACK_BOT_TOKEN / SLACK_APP_TOKEN not set — live bridge may be inactive"
fi

echo "OK: slack-live-smoke hub checks passed"
