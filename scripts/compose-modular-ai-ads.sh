#!/usr/bin/env bash
# Compose modular-AI square ads (1080×1080).
# Usage: ./scripts/compose-modular-ai-ads.sh [router|observe|compose|stacks|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
VARIANT="${1:-all}"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" "$ROOT/scripts/lib/compose_modular_ai_graphics.py" "$VARIANT" --root "$ROOT"
