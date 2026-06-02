#!/usr/bin/env bash
# A/B test collaboration smart routing on execute-deliverable scenario.
# Note: light model downgrade for file deliverables is fixed in routing/plan.go;
# this matrix measures provider-level smart routing impact only.
set -euo pipefail
cd "$(dirname "$0")/.."

OUT_DIR="scenarios/out"
mkdir -p "$OUT_DIR"
STAMP="$(date +%Y%m%d-%H%M%S)"
REPORT="$OUT_DIR/routing-matrix-${STAMP}.md"
LOG_FILE="${NEURAL_JUNKIE_SERVER_LOG:-/tmp/chat-server.log}"

hub="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
code="$(curl -sf -o /dev/null -w '%{http_code}' "${hub}/api/health" 2>/dev/null || echo 000)"
if [ "$code" != "200" ]; then
  echo "FAIL: hub not healthy at ${hub} (HTTP ${code})" >&2
  echo "Start with: NEURAL_JUNKIE_RATE_LIMIT=0 make server" >&2
  exit 1
fi

set_smart_routing() {
  export HUB="$hub"
  export ENABLED="$1"
  export CFG="$(curl -sf "${hub}/api/settings")"
  python3 - <<'PY'
import json, os, sys, urllib.request
hub = os.environ["HUB"]
enabled = os.environ["ENABLED"] == "1"
cfg = json.loads(os.environ["CFG"])
cfg.setdefault("collaboration", {})["smart_routing_enabled"] = enabled
req = urllib.request.Request(
    f"{hub}/api/settings",
    data=json.dumps(cfg).encode(),
    headers={"Content-Type": "application/json"},
    method="PUT",
)
with urllib.request.urlopen(req, timeout=30) as resp:
    sys.exit(0 if resp.status == 200 else 1)
PY
}

echo "# Collab routing matrix — ${STAMP}" >"$REPORT"
echo "" >>"$REPORT"
echo "Hub: \`${hub}\`" >>"$REPORT"
echo "" >>"$REPORT"
echo "Scenario: \`execute-deliverable\`. Smart routing toggles **provider** selection only; file deliverable tasks keep the agent default model." >>"$REPORT"
echo "" >>"$REPORT"
echo "| smart routing | pass | seconds | routing log lines | note |" >>"$REPORT"
echo "|---------------|------|---------|-------------------|------|" >>"$REPORT"

failures=0
for variant in off on; do
  if [ "$variant" = "on" ]; then
    set_smart_routing 1
    enabled_label="true"
  else
    set_smart_routing 0
    enabled_label="false"
  fi

  tmp="$(mktemp)"
  start=$(date +%s)
  note=""
  if NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/collab-scenarios.py \
    --scenario execute-deliverable \
    --profile fast \
    >"$tmp" 2>&1; then
    pass="yes"
  else
    pass="no"
    failures=$((failures + 1))
    note="$(tail -n 3 "$tmp" | tr '\n' ' ' | sed 's/|/\\|/g' | cut -c1-80)"
  fi
  end=$(date +%s)
  elapsed=$((end - start))

  routing_lines=""
  if [ -f "$LOG_FILE" ]; then
    routing_lines="$(grep -c '\[collab-routing\]' "$LOG_FILE" 2>/dev/null || echo 0)"
  else
    routing_lines="n/a (no log at ${LOG_FILE})"
  fi

  echo "| ${enabled_label} | ${pass} | ${elapsed} | ${routing_lines} | ${note} |" >>"$REPORT"
  echo "matrix: smart_routing=${enabled_label} pass=${pass} (${elapsed}s) routing_lines=${routing_lines}"
  rm -f "$tmp"
done

echo "" >>"$REPORT"
echo "Log grep: \`grep '[collab-routing]' ${LOG_FILE}\`" >>"$REPORT"

echo ""
echo "Report: ${REPORT}"
if [ "$failures" -gt 0 ]; then
  echo "FAIL: ${failures} variant(s) failed" >&2
  exit 1
fi
echo "PASS: routing matrix complete"
