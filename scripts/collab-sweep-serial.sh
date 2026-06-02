#!/usr/bin/env bash
# Run collab scenarios one at a time; advance to the next only after PASS.
#
# Usage:
#   make collab-sweep-serial              # all scenarios, stop on first FAIL
#   make collab-sweep-serial RESUME=1     # skip rows already PASS in matrix
#   ONLY=execution-no-stack-commands make collab-sweep-serial
#   RETRIES=3 make collab-sweep-serial    # up to 3 attempts per scenario (flakes)
#
# After a FAIL, fix product/harness then re-run the same scenario:
#   make collab-scenario SCENARIO=<name> VERBOSE=1
#   make collab-sweep-serial RESUME=1

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MATRIX="${MATRIX:-$ROOT/docs/testing/collab-matrix.tsv}"
LOG="${LOG:-/tmp/nj-collab-sweep-serial.log}"
HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
RETRIES="${RETRIES:-1}"
RESUME="${RESUME:-0}"
ONLY="${ONLY:-}"
VERBOSE_FLAG=""
if [ "${VERBOSE:-}" = "1" ]; then
  VERBOSE_FLAG="--verbose"
fi

mkdir -p "$(dirname "$MATRIX")"
if [ ! -f "$MATRIX" ]; then
  printf 'scenario\tstatus\tnotes\tupdated_at\n' >"$MATRIX"
fi

matrix_status() {
  local name="$1"
  awk -F'\t' -v s="$name" 'NR > 1 && $1 == s { print $2; exit }' "$MATRIX" 2>/dev/null || true
}

matrix_set() {
  local name="$1" status="$2" notes="${3:-}"
  local ts
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local tmp
  tmp="$(mktemp)"
  awk -F'\t' -v s="$name" 'NR == 1 || $1 != s' "$MATRIX" >"$tmp"
  printf '%s\t%s\t%s\t%s\n' "$name" "$status" "$notes" "$ts" >>"$tmp"
  mv "$tmp" "$MATRIX"
}

import_from_log() {
  local log_path="$1"
  [ -f "$log_path" ] || return 0
  while IFS= read -r line; do
    case "$line" in
      "=== PASS: "*)
        n="${line#=== PASS: }"
        n="${n% ===}"
        matrix_set "$n" PASS "imported from batch log"
        ;;
      "=== FAIL: "*)
        n="${line#=== FAIL: }"
        n="${n% ===}"
        matrix_set "$n" FAIL "imported from batch log"
        ;;
    esac
  done < <(grep -E '^=== (PASS|FAIL): ' "$log_path" || true)
}

run_one() {
  local scenario="$1"
  echo "" | tee -a "$LOG"
  echo "=== $(date -u +%H:%M:%SZ) scenario: $scenario ===" | tee -a "$LOG"
  PYTHONUNBUFFERED=1 NEURAL_JUNKIE_RATE_LIMIT=0 \
    python3 -u scripts/collab-scenarios.py --scenario "$scenario" $VERBOSE_FLAG 2>&1 | tee -a "$LOG"
}

echo "collab-sweep-serial → hub=$HUB matrix=$MATRIX log=$LOG" | tee "$LOG"
echo "preflight:" | tee -a "$LOG"
make collab-preflight 2>&1 | tee -a "$LOG"

if [ -f /tmp/nj-collab-sweep-2026-06-02.log ]; then
  import_from_log /tmp/nj-collab-sweep-2026-06-02.log
fi

SCENARIOS=()
while IFS= read -r _s; do
  [ -n "$_s" ] && SCENARIOS+=("$_s")
done < <(python3 scripts/collab-scenarios.py --list)

for scenario in "${SCENARIOS[@]}"; do
  [ -n "$scenario" ] || continue
  if [ -n "$ONLY" ] && [ "$scenario" != "$ONLY" ]; then
    continue
  fi
  if [ "$RESUME" = "1" ]; then
    st="$(matrix_status "$scenario")"
    if [ "$st" = "PASS" ]; then
      echo "skip (PASS): $scenario" | tee -a "$LOG"
      continue
    fi
  fi

  attempt=0
  while true; do
    attempt=$((attempt + 1))
    if run_one "$scenario"; then
      matrix_set "$scenario" PASS ""
      echo "✓ PASS: $scenario (attempt $attempt)" | tee -a "$LOG"
      break
    fi
    matrix_set "$scenario" FAIL "attempt $attempt"
    echo "✗ FAIL: $scenario (attempt $attempt/$RETRIES)" | tee -a "$LOG"
    if [ "$attempt" -ge "$RETRIES" ]; then
      echo "" >&2
      echo "Stopped on $scenario. Fix, then:" >&2
      echo "  make collab-scenario SCENARIO=$scenario VERBOSE=1" >&2
      echo "  make collab-sweep-serial RESUME=1   # continue after PASS" >&2
      exit 1
    fi
    echo "retrying $scenario in 15s..." | tee -a "$LOG"
    sleep 15
  done
done

echo "" | tee -a "$LOG"
echo "All scenarios PASS. Matrix: $MATRIX" | tee -a "$LOG"
