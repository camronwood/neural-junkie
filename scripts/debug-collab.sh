#!/usr/bin/env bash
# Wrapper for debug-collab.py (see --help).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec python3 "${ROOT}/scripts/debug-collab.py" "$@"
