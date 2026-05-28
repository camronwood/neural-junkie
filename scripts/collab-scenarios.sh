#!/usr/bin/env bash
# Run collaboration scenarios against a live hub.
set -euo pipefail
cd "$(dirname "$0")/.."
exec python3 scripts/collab-scenarios.py "$@"
