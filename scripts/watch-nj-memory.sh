#!/usr/bin/env bash
# Watch Neural Junkie (Tauri) + optional Go hub RSS and macOS memory_pressure snapshot.
# Usage: ./scripts/watch-nj-memory.sh [interval_seconds]
set -euo pipefail
INTERVAL="${1:-10}"

nj_pid() {
  pgrep -f '/Neural Junkie$' 2>/dev/null | head -1 || true
}

hub_pid() {
  pgrep -f 'go run cmd/server/main\.go' 2>/dev/null | head -1 || true
}

rss_mb() {
  local pid="$1"
  [[ -z "$pid" ]] && { echo "—"; return; }
  local kb
  kb="$(ps -p "$pid" -o rss= 2>/dev/null | tr -d ' ')" || { echo "—"; return; }
  [[ -z "$kb" || "$kb" == "RSS" ]] && { echo "—"; return; }
  awk -v k="$kb" 'BEGIN { printf "%.1f MB\n", k/1024 }'
}

echo "Interval: ${INTERVAL}s (Ctrl+C to stop)"
echo "Columns: time | Neural Junkie RSS | hub go RSS | memory_pressure free % (first line of Stats)"
echo ""

while true; do
  npid="$(nj_pid)"
  hpid="$(hub_pid)"
  nj_rss="$(rss_mb "$npid")"
  hub_rss="$(rss_mb "$hpid")"
  freepct=""
  if [[ -x /usr/bin/memory_pressure ]]; then
    freepct="$(/usr/bin/memory_pressure 2>/dev/null | awk -F': ' '/System-wide memory free percentage/ {print $2; exit}')"
  fi
  printf '%s | NJ %s | hub %s | system free %s\n' "$(date '+%H:%M:%S')" "$nj_rss" "$hub_rss" "${freepct:-n/a}"
  sleep "$INTERVAL"
done
