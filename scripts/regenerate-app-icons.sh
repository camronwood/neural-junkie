#!/usr/bin/env bash
# Regenerate Neural Junkie app icons (transparent PNG + Tauri bundle + docs site assets).
#
# Uses Dickory Docs' knock_out_background.py (edge flood-fill; removes light squircle backgrounds).
# Requires: Python venv with Pillow+numpy (see dickory-docs/.venv-icon) or neural-junkie/.venv-icon.
#
# Usage (from repo root):
#   ./scripts/regenerate-app-icons.sh
#   ICON_BG=dark ./scripts/regenerate-app-icons.sh   # if source has dark/black backdrop

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SOURCE="${ICON_SOURCE:-assets/icons/Gemini_Generated_Image_7ofmua7ofmua7ofm.png}"
if [[ ! -f "$SOURCE" ]]; then
  SOURCE="assets/icons/app-icon.png"
fi

DICKORY="$(cd "$ROOT/../dickory-docs" 2>/dev/null && pwd || true)"
KNOCK_SCRIPT="${KNOCK_SCRIPT:-}"
if [[ -z "$KNOCK_SCRIPT" && -n "$DICKORY" && -f "$DICKORY/scripts/knock_out_background.py" ]]; then
  KNOCK_SCRIPT="$DICKORY/scripts/knock_out_background.py"
fi
if [[ -z "$KNOCK_SCRIPT" || ! -f "$KNOCK_SCRIPT" ]]; then
  echo "error: knock_out_background.py not found (set KNOCK_SCRIPT or clone dickory-docs beside neural-junkie)" >&2
  exit 1
fi

VENV_PY="$ROOT/.venv-icon/bin/python"
if [[ ! -x "$VENV_PY" ]]; then
  if [[ -x "${DICKORY:-}/.venv-icon/bin/python" ]]; then
    VENV_PY="${DICKORY}/.venv-icon/bin/python"
  else
    python3 -m venv "$ROOT/.venv-icon"
    "$ROOT/.venv-icon/bin/pip" install -q Pillow numpy
    VENV_PY="$ROOT/.venv-icon/bin/python"
  fi
fi

BG="${ICON_BG:-light}"
echo "Source: $SOURCE (bg=$BG)"
"$VENV_PY" "$KNOCK_SCRIPT" "$SOURCE" -o assets/icons/app-icon-transparent.png --bg "$BG"
cp assets/icons/app-icon-transparent.png assets/icons/app-icon-source.png

echo "Generating Tauri icon set..."
(cd desktop && npx @tauri-apps/cli icon ../assets/icons/app-icon-source.png)

mkdir -p docs/assets/icon
cp assets/icons/app-icon-transparent.png docs/assets/icon/og-image.png
cp desktop/src-tauri/icons/32x32.png docs/assets/icon/favicon-32.png
cp desktop/src-tauri/icons/128x128@2x.png docs/assets/icon/apple-touch-icon.png
cp desktop/src-tauri/icons/128x128.png docs/assets/icon/logo-128.png
cp assets/icons/app-icon-transparent.png desktop/public/app-icon.png

echo "Done. Icons: assets/icons/, desktop/src-tauri/icons/, docs/assets/icon/"
