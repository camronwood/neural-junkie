#!/usr/bin/env bash
# Compose neural-junkie-beta-download-ad-1080.png from brand + screenshot.
# Usage: ./scripts/compose-beta-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-beta-download-ad-1080.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT" "$OUT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(sys.argv[1])
OUT = Path(sys.argv[2])
brand = ROOT / "assets/neural-junkie-social-ad-1080.png"
shot = next((ROOT / "assets/screenshots").glob("Screenshot 2026-05-13 at 12.35.40*"))

def font(path: str, size: int):
    try:
        return ImageFont.truetype(path, size)
    except OSError:
        return ImageFont.load_default()

canvas = Image.new("RGB", (1080, 1080), "#0d0d1a")
top = Image.open(brand).convert("RGBA").resize((1080, 420), Image.Resampling.LANCZOS)
canvas.paste(top, (0, 0), top)
s = Image.open(shot).convert("RGBA")
s.thumbnail((980, 520), Image.Resampling.LANCZOS)
border = 4
boxed = Image.new("RGBA", (s.width + 2 * border, s.height + 2 * border), "#533483")
boxed.paste(s, (border, border), s)
canvas.paste(boxed, ((1080 - boxed.width) // 2, 430), boxed)
draw = ImageDraw.Draw(canvas)
fb = font("/System/Library/Fonts/Supplemental/Arial Bold.ttf", 34)
fm = font("/System/Library/Fonts/Supplemental/Arial.ttf", 22)
fs = font("/System/Library/Fonts/Supplemental/Arial.ttf", 17)
draw.rounded_rectangle((220, 900, 860, 958), radius=14, fill="#e94560")
draw.text((540, 928), "NOW INSTALLABLE — OPEN BETA", fill="white", font=fb, anchor="mm")
draw.text((540, 978), "v1.0.0-beta.1  ·  macOS · Windows · Linux", fill="#c0c0d0", font=fm, anchor="mm")
draw.text((540, 1050), "github.com/camronwood/neural-junkie/releases", fill="#8888aa", font=fs, anchor="mm")
canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
