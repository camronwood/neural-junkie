#!/usr/bin/env bash
# Smoke test for secondary analysis Phase 1–3 integration.
set -euo pipefail

NJ_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SAT_ROOT="${SECONDARY_ANALYSIS_TOOLS_PATH:-}"
if [[ -z "$SAT_ROOT" ]]; then
  for candidate in \
    "$NJ_ROOT/../secondary-analysis-tools" \
    "$NJ_ROOT/../../secondary-analysis-tools" \
    "/Users/camronwood/development/secondary-analysis-tools"; do
    if [[ -d "$candidate/cli" ]]; then
      SAT_ROOT="$(cd "$candidate" && pwd)"
      break
    fi
  done
fi
FIXTURE="${NJ_FIXTURE:-$NJ_ROOT/testdata/scan-analysis}"

echo "== Neural Junkie secondary analysis smoke =="
echo "NJ_ROOT=$NJ_ROOT"
echo "SAT_ROOT=$SAT_ROOT"
echo "FIXTURE=$FIXTURE"

if [[ ! -d "$FIXTURE/reports" ]]; then
  echo "SKIP: fixture missing at $FIXTURE"
  exit 0
fi

echo ""
echo "== Go: panel QC tests =="
(cd "$NJ_ROOT" && go test ./internal/scananalysis/... -count=1 -run 'TestRun12PlexQC|TestLLOQ')

echo ""
echo "== Desktop: panelQcUtils vitest =="
(cd "$NJ_ROOT/desktop" && npm run test -- --run src/utils/panelQcUtils.test.ts)

if [[ -d "$SAT_ROOT/cli" ]]; then
  echo ""
  echo "== Python: CLI smoke =="
  if [[ ! -d "$SAT_ROOT/.venv" ]]; then
    python3 -m venv "$SAT_ROOT/.venv"
    "$SAT_ROOT/.venv/bin/pip" install -q -r "$SAT_ROOT/requirements.txt"
  fi
  (cd "$SAT_ROOT" && .venv/bin/python -m pytest tests/tests/test_cli_smoke.py -q)
else
  echo "SKIP: secondary-analysis-tools not found at $SAT_ROOT"
fi

echo ""
echo "OK: secondary analysis smoke checks passed"
