#!/usr/bin/env bash
# Run Playwright UI journeys against the desktop Vite shell (+ live hub).
# Overnight: hub should already be up (SKIP_BOOT=1). Set NJ_E2E_HUB_URL if needed.
#
# Optional Tauri WebDriver path (macOS overnight box):
#   NJ_E2E_TAURI=1  — requires `tauri-driver` on PATH; see docs/DESKTOP_E2E.md
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}/desktop"

HUB="${NJ_E2E_HUB_URL:-${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}}"
export NJ_E2E_HUB_URL="${HUB}"

STAMP="$(date +%Y%m%d-%H%M)"
REPORT_DIR="${ROOT}/docs/testing"
mkdir -p "${REPORT_DIR}"
REPORT="${REPORT_DIR}/desktop-e2e-${STAMP}.md"

{
  echo "# desktop-e2e — ${STAMP}"
  echo
  echo "- Hub: \`${HUB}\`"
  echo "- Mode: \`\${NJ_E2E_TAURI:-web}\`"
  echo
} >"${REPORT}"

if [[ ! -d node_modules/@playwright/test ]]; then
  echo "Installing Playwright..."
  npm install --no-save @playwright/test@1.51.0
  npx playwright install chromium
fi

set +e
npx playwright test --config=playwright.config.ts "$@"
rc=$?
set -e

{
  echo "## Result"
  echo
  if [[ "${rc}" -eq 0 ]]; then
    echo "**PASS**"
  else
    echo "**FAIL** (exit ${rc})"
    echo
    echo "Traces/screenshots under \`desktop/test-results/\` and \`desktop/e2e-report/\`."
  fi
} >>"${REPORT}"

echo "Wrote ${REPORT}"
exit "${rc}"
