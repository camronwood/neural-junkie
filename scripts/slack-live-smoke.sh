#!/usr/bin/env bash
# Maintainer live Slack smoke (synthetic inbound by default — no Slack messages sent).
# Optional outbound: SLACK_SMOKE_ALLOW=1 SLACK_SMOKE_OUTBOUND=1 + private #nj-smoke-test channel.
set -euo pipefail

HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
HUB="${HUB%/}"

OUTBOUND="${SLACK_SMOKE_OUTBOUND:-0}"
CHANNEL_ID="${SLACK_SMOKE_CHANNEL_ID:-}"

echo "slack-live-smoke: hub=$HUB outbound=$OUTBOUND"

if ! curl -sf "$HUB/api/health" >/dev/null; then
  echo "FAIL: hub not healthy — start the hub first" >&2
  exit 1
fi

echo "Diagnose:"
DIAG=$(curl -sf "$HUB/api/slack/diagnose")
python3 -m json.tool <<<"$DIAG"

if ! python3 -c "import json,sys; d=json.load(sys.stdin); sys.exit(0 if d.get('auth_test_ok') and d.get('socket_open_ok') else 1)" <<<"$DIAG"; then
  echo "FAIL: diagnose auth_test_ok and socket_open_ok required" >&2
  exit 1
fi

BODY=$(python3 -c "import json,os; print(json.dumps({
  'channel_id': os.environ.get('SLACK_SMOKE_CHANNEL_ID','') or None,
  'outbound': os.environ.get('SLACK_SMOKE_OUTBOUND','0') == '1',
  'allow_outbound': os.environ.get('SLACK_SMOKE_OUTBOUND','0') == '1',
}))")

echo "Smoke run (outbound=${OUTBOUND}):"
SMOKE=$(curl -sf -X POST "$HUB/api/slack/smoke/run" \
  -H 'Content-Type: application/json' \
  -d "$BODY") || {
  echo "FAIL: smoke/run request failed" >&2
  exit 1
}
python3 -m json.tool <<<"$SMOKE"

if ! python3 -c "import json,sys; d=json.load(sys.stdin); sys.exit(0 if d.get('ok') else 1)" <<<"$SMOKE"; then
  echo "FAIL: smoke checks did not pass" >&2
  exit 1
fi

if [[ "$OUTBOUND" == "1" ]]; then
  echo "OK: live smoke passed (including gated outbound to test channel)"
else
  echo "OK: live smoke passed (synthetic inbound only — no Slack messages sent)"
  if [[ -z "$CHANNEL_ID" ]]; then
    echo "Tip: set SLACK_SMOKE_CHANNEL_ID or save a channel binding for synthetic inbound routing"
  fi
fi
