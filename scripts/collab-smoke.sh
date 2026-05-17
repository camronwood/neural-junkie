#!/usr/bin/env bash
# Collaboration smoke: in-process API test (default) or live hub (--live).
set -euo pipefail
cd "$(dirname "$0")/.."
exec python3 scripts/collab-smoke.py "$@"
