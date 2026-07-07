#!/usr/bin/env bash
# Compose LoRA v2 LinkedIn covers (1200×627 + 1080×1080).
# Usage: ./scripts/compose-lora-v2-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT" <<'PY'
import sys
from pathlib import Path

root = Path(sys.argv[1])
sys.path.insert(0, str(root / "scripts" / "lib"))
from compose_lora_v2_graphics import render_article_cover, render_square_ad

assets = root / "assets"
covers = root / "docs" / "media" / "articles" / "covers"
gallery = root / "docs" / "media" / "gallery" / "ads"

render_article_cover(assets / "neural-junkie-lora-v2-1200.png", covers / "neural-junkie-lora-v2-1200.png")
render_square_ad(assets / "neural-junkie-lora-v2-ad-1080.png")
render_square_ad(gallery / "neural-junkie-lora-v2-ad-1080.png")
PY
