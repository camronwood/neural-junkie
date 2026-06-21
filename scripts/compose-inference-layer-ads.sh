#!/usr/bin/env bash
# Compose inference-layer square ads (1080×1080).
# Usage: ./scripts/compose-inference-layer-ads.sh [skip|gate|trust|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
VARIANT="${1:-all}"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" "$ROOT/scripts/lib/compose_inference_layer_graphics.py" "$VARIANT" --root "$ROOT"
