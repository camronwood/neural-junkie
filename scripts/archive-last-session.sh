#!/usr/bin/env bash
# Archive ~/.neural-junkie/last-session.json before clearing DMs or debugging bad threads.
#
# Examples:
#   ./scripts/archive-last-session.sh
#   ./scripts/archive-last-session.sh archive --label before-bio-clear --note "polluted DM"
#   ./scripts/archive-last-session.sh list
#   ./scripts/archive-last-session.sh summary --channel biologyexpert --tail 8
#   ./scripts/archive-last-session.sh analyze
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PY="${ROOT}/scripts/nj_last_session.py"
if [[ $# -eq 0 ]] || [[ "$1" == --label ]] || [[ "$1" == --note ]] || [[ "$1" == -q ]] || [[ "$1" == --quiet ]]; then
  exec python3 "${PY}" archive "$@"
fi
exec python3 "${PY}" "$@"
