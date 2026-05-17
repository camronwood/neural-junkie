#!/usr/bin/env bash
# Analyze ~/.neural-junkie/last-session.json (collab + conversation focus).
# Prefer: ./scripts/debug-collab.py session
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SESSION="${1:-${HOME}/.neural-junkie/last-session.json}"
exec python3 "${ROOT}/scripts/debug-collab.py" session --session "${SESSION}" "${@:2}"
