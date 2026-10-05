#!/usr/bin/env bash
# Wait until evening CT, then launch AWAY_GATE=full overnight with a future 05:00 deadline.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TARGET_HOUR="${OVERNIGHT_LAUNCH_HOUR_CT:-20}"
LOG="${OVERNIGHT_LAUNCH_LOG:-/tmp/nj-evening-overnight-launch.log}"

now_ct_hour() {
  TZ=America/Chicago date +%H
}

echo "[$(TZ=America/Chicago date '+%Y-%m-%d %H:%M:%S %Z')] waiting until ${TARGET_HOUR}:00 CT to launch overnight" | tee -a "$LOG"
while true; do
  h="$(now_ct_hour)"
  h=$((10#$h))
  if (( h >= TARGET_HOUR )); then
    break
  fi
  sleep 300
done

echo "[$(TZ=America/Chicago date '+%Y-%m-%d %H:%M:%S %Z')] launching AWAY_GATE=full overnight" | tee -a "$LOG"
# shellcheck disable=SC1091
source load-env.sh
export AWAY_GATE=full
export AWAY_DEADLINE_CT=05:00
export NJ_OVERNIGHT_TARGET=away
export NEURAL_JUNKIE_RATE_LIMIT=0
make overnight NJ_OVERNIGHT_TARGET=away 2>&1 | tee -a "$LOG"
echo "[$(TZ=America/Chicago date '+%Y-%m-%d %H:%M:%S %Z')] overnight finished exit=${PIPESTATUS[0]}" | tee -a "$LOG"
