#!/usr/bin/env bash
# Compose LoRA v2 LinkedIn covers (1200×627 + 1080×1080).
# Usage: ./scripts/compose-lora-v2-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_HEADER="$ROOT/assets/neural-junkie-lora-v2-1200.png"
OUT_AD="$ROOT/assets/neural-junkie-lora-v2-ad-1080.png"
GALLERY_HEADER="$ROOT/docs/media/articles/covers/neural-junkie-lora-v2-1200.png"
GALLERY_AD="$ROOT/docs/media/gallery/ads/neural-junkie-lora-v2-ad-1080.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

mkdir -p "$(dirname "$GALLERY_HEADER")" "$(dirname "$GALLERY_AD")"
exec "$PY" - "$OUT_HEADER" "$OUT_AD" "$GALLERY_HEADER" "$GALLERY_AD" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

OUT_HEADER = Path(sys.argv[1])
OUT_AD = Path(sys.argv[2])
GALLERY_HEADER = Path(sys.argv[3])
GALLERY_AD = Path(sys.argv[4])

TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
PINK = (199, 125, 255)
GREEN = (72, 199, 142)
MUTED = (168, 176, 184)


def font(bold: bool, size: int, mono: bool = False):
    if mono:
        paths = ["/System/Library/Fonts/Menlo.ttc"]
    else:
        paths = [
            "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold
            else "/System/Library/Fonts/Supplemental/Arial.ttf",
        ]
    for path in paths:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def gradient(canvas, top, bottom):
    draw = ImageDraw.Draw(canvas)
    w, h = canvas.size
    for y in range(h):
        t = y / max(h - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (w, y)], fill=(r, g, b))


def compose_header(path: Path) -> None:
    W, H = 1200, 627
    M = 48
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient(canvas, (14, 16, 32), (8, 22, 30))
    draw = ImageDraw.Draw(canvas)
    draw.text((M, 36), "OPEN SOURCE · NEURAL JUNKIE · LORA V2", fill=TEAL, font=font(True, 11))
    draw.text((M, 72), "TRAIN ONCE.", fill=(255, 255, 255), font=font(True, 40))
    draw.text((M, 118), "COMPOUND FOREVER.", fill=GREEN, font=font(True, 42))
    draw.text((M, 175), "v1 → v2: refresh, profiles, routing, MLX, team sharing", fill=MUTED, font=font(False, 15))

    v1_x, v2_x, y0, bw, bh = M, 620, 220, 240, 200
    draw.rounded_rectangle((v1_x, y0, v1_x + bw, y0 + bh), radius=12, fill=(18, 24, 44), outline=MUTED, width=2)
    draw.text((v1_x + 16, y0 + 14), "v1", fill=MUTED, font=font(True, 16))
    for i, line in enumerate(["Import", "Train once", "Assign tag"]):
        draw.text((v1_x + 16, y0 + 48 + i * 28), line, fill=(210, 214, 224), font=font(False, 14))

    draw.text((v1_x + bw + 50, y0 + bh // 2), "→", fill=PINK, font=font(True, 48))

    draw.rounded_rectangle((v2_x, y0, v2_x + bw, y0 + bh), radius=12, fill=(18, 24, 44), outline=GREEN, width=2)
    draw.text((v2_x + 16, y0 + 14), "v2", fill=GREEN, font=font(True, 16))
    for i, line in enumerate(["Refresh", "Profiles", "Routing", "MLX + Eval", "MCP + HF"]):
        draw.text((v2_x + 16, y0 + 44 + i * 24), line, fill=(210, 214, 224), font=font(False, 13))

    fy = H - M - 46
    draw.rounded_rectangle((M, fy, W - M, fy + 46), radius=12, fill=PINK)
    draw.text((W // 2, fy + 23), "github.com/camronwood/neural-junkie · LORA_V2.md", fill=(255, 255, 255), font=font(True, 14), anchor="mm")
    canvas.save(path, "PNG")


def compose_ad(path: Path) -> None:
    S = 1080
    M = 56
    canvas = Image.new("RGB", (S, S), (10, 14, 28))
    gradient(canvas, (14, 16, 32), (8, 22, 30))
    draw = ImageDraw.Draw(canvas)
    draw.text((M, M), "LORA V2", fill=TEAL, font=font(True, 14))
    draw.text((M, M + 36), "Your repo expert compounds", fill=(255, 255, 255), font=font(True, 44))
    pillars = ["Incremental refresh", "Dual-tag profiles", "Per-turn routing", "MLX on Mac", "Team adapters"]
    for i, p in enumerate(pillars):
        draw.text((M, M + 120 + i * 52), "▸ " + p, fill=GREEN if i % 2 == 0 else TEAL, font=font(False, 28))
    canvas.save(path, "PNG")


compose_header(OUT_HEADER)
compose_header(GALLERY_HEADER)
compose_ad(OUT_AD)
compose_ad(GALLERY_AD)
print(f"Wrote {OUT_HEADER}")
print(f"Wrote {OUT_AD}")
PY
