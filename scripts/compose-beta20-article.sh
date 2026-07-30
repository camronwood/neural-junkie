#!/usr/bin/env bash
# Compose neural-junkie-beta20-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-beta20-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" "$ROOT/scripts/lib/compose_beta20_graphics.py" article --root "$ROOT"
