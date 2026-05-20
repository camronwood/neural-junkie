#!/usr/bin/env bash
# Analyze ~/.neural-junkie/last-session.json (collab + conversation focus).
# Archive first: ./scripts/archive-last-session.sh --label before-debug
# Quick summary:  ./scripts/archive-last-session.sh summary --channel biologyexpert --tail 10
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec "${ROOT}/scripts/archive-last-session.sh" analyze "$@"
