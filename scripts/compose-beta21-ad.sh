#!/usr/bin/env bash
# Deprecated: use ./scripts/compose-beta21-ads.sh [slack|chat|test|all]
# Wrapper regenerates the Slack ad only (replaces old combined creative).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
exec "$ROOT/scripts/compose-beta21-ads.sh" "${1:-slack}"
