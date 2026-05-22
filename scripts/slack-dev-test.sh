#!/usr/bin/env bash
# Slack bridge dev helpers (hub on localhost:18765).
set -euo pipefail

HUB="${NEURAL_JUNKIE_HUB:-http://localhost:18765}"
AGENT_ID="${NEURAL_JUNKIE_SLACK_AGENT_ID:-7b7f2922-e4a5-4af4-984e-e50186ddac96}"

cmd="${1:-help}"
shift || true

case "$cmd" in
  status)
    curl -s "$HUB/api/slack/status" | python3 -m json.tool
    ;;
  diagnose)
    curl -s "$HUB/api/slack/diagnose" | python3 -m json.tool
    ;;
  channels)
    curl -s "$HUB/api/slack/channels" | python3 -c "
import json,sys
for ch in json.load(sys.stdin):
    priv = ' (private)' if ch.get('is_private') else ''
    mem = ' member' if ch.get('is_member') else ' NOT_MEMBER'
    print(f\"{ch['id']}  #{ch['name']}{priv}{mem}\")
"
    ;;
  restart)
    curl -s -X POST "$HUB/api/slack/restart" | python3 -m json.tool
    sleep 1
    curl -s "$HUB/api/slack/status" | python3 -m json.tool
    ;;
  bind)
    CHANNEL_ID="${1:?usage: $0 bind CHANNEL_ID [channel_name]}"
    NAME="${2:-}"
    BODY=$(python3 -c "import json; print(json.dumps({
      'slack_channel_id': '$CHANNEL_ID',
      'slack_channel_name': '$NAME' or None,
      'agent_id': '$AGENT_ID',
      'agent_name': 'Assistant',
      'policy': 'mention_only',
    }))")
    curl -s -X POST "$HUB/api/slack/bindings" -H 'Content-Type: application/json' -d "$BODY" | python3 -m json.tool
    ;;
  test-post)
    CHANNEL_ID="${1:?usage: $0 test-post CHANNEL_ID}"
    TEXT="${2:-Neural Junkie Slack test}"
    curl -s -X POST "$HUB/api/slack/test-post" \
      -H 'Content-Type: application/json' \
      -d "$(python3 -c "import json; print(json.dumps({'slack_channel_id':'$CHANNEL_ID','text':'''$TEXT'''}))")" \
      | python3 -m json.tool
    ;;
  help|*)
    cat <<EOF
Usage: $0 <command>

  status              Bridge status (socket_connected = real-time events)
  diagnose            Token format + apps.connections.open (why connection_error)
  channels            List Slack channels bot can see (use member + correct C…)
  restart             Restart bridge (uses long-lived context)
  bind CHANNEL_ID [name]  Create binding → Assistant
  test-post CHANNEL_ID [text]

Env: NEURAL_JUNKIE_HUB, NEURAL_JUNKIE_SLACK_AGENT_ID
EOF
    ;;
esac
