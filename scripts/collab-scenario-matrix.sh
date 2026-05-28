#!/usr/bin/env bash
# Sweep planning-two-agent with different agent profiles and discussion budgets.
set -euo pipefail
cd "$(dirname "$0")/.."

OUT_DIR="scenarios/out"
mkdir -p "$OUT_DIR"
STAMP="$(date +%Y%m%d-%H%M%S)"
REPORT="$OUT_DIR/matrix-${STAMP}.md"

hub="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
code="$(curl -sf -o /dev/null -w '%{http_code}' "${hub}/api/health" 2>/dev/null || echo 000)"
if [ "$code" != "200" ]; then
  echo "FAIL: hub not healthy at ${hub} (HTTP ${code})" >&2
  echo "Start with: make gui  or  make server" >&2
  exit 1
fi

# profile:rounds:messages
VARIANTS=(
  "fast:1:2"
  "fast:2:4"
  "realistic:1:2"
  "realistic:2:4"
)

echo "# Collab scenario matrix — ${STAMP}" >"$REPORT"
echo "" >>"$REPORT"
echo "Hub: \`${hub}\`" >>"$REPORT"
echo "" >>"$REPORT"
echo "| profile | rounds | messages | pass | seconds | note |" >>"$REPORT"
echo "|---------|--------|----------|------|---------|------|" >>"$REPORT"

failures=0
for v in "${VARIANTS[@]}"; do
  IFS=: read -r profile rounds messages <<<"$v"
  tmp="$(mktemp)"
  start=$(date +%s)
  note=""
  if NJ_SCENARIO_PROFILE="$profile" \
    NJ_SCENARIO_ROUNDS="$rounds" \
    NJ_SCENARIO_MESSAGES="$messages" \
    python3 scripts/collab-scenarios.py \
    --scenario planning-two-agent \
    --profile "$profile" \
    >"$tmp" 2>&1; then
    pass="yes"
  else
    pass="no"
    failures=$((failures + 1))
    note="$(tail -n 3 "$tmp" | tr '\n' ' ' | sed 's/|/\\|/g' | cut -c1-80)"
  fi
  end=$(date +%s)
  elapsed=$((end - start))
  echo "| ${profile} | ${rounds} | ${messages} | ${pass} | ${elapsed} | ${note} |" >>"$REPORT"
  echo "matrix: profile=${profile} rounds=${rounds} messages=${messages} pass=${pass} (${elapsed}s)"
  rm -f "$tmp"
done

echo "" >>"$REPORT"
echo "Template: \`planning-two-agent\` with \`NJ_SCENARIO_ROUNDS\` / \`NJ_SCENARIO_MESSAGES\` overrides." >>"$REPORT"

echo ""
echo "Report: ${REPORT}"
if [ "$failures" -gt 0 ]; then
  echo "FAIL: ${failures} variant(s) failed" >&2
  exit 1
fi
echo "PASS: all matrix variants"
