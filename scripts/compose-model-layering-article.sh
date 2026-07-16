#!/usr/bin/env bash
# Compose neural-junkie-model-layering-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-model-layering-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/model-layering/creatives/neural-junkie-model-layering-1200.png"
GALLERY="$ROOT/docs/media/gallery/ads/neural-junkie-model-layering-1200.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$OUT" "$GALLERY" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

OUT = Path(sys.argv[1])
GALLERY = Path(sys.argv[2])
W, H = 1200, 627
MARGIN = 48
CONTENT_W = W - 2 * MARGIN
FOOTER_H = 46
FOOTER_Y = H - MARGIN - FOOTER_H

TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
PINK = (199, 125, 255)
ACCENT = (233, 69, 96)
GREEN = (72, 199, 142)
MUTED = (168, 176, 184)
BLUE = (96, 165, 250)
BIO = (120, 220, 160)


def font(bold: bool, size: int, mono: bool = False):
    if mono:
        paths = ["/System/Library/Fonts/Menlo.ttc"]
    else:
        paths = [
            "/System/Library/Fonts/Supplemental/Arial Bold.ttf"
            if bold
            else "/System/Library/Fonts/Supplemental/Arial.ttf",
        ]
    for path in paths:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def wrap(draw, text: str, fnt, max_w: int):
    words = text.split()
    lines, cur = [], ""
    for w in words:
        test = f"{cur} {w}".strip()
        if draw.textlength(test, font=fnt) <= max_w:
            cur = test
        else:
            if cur:
                lines.append(cur)
            cur = w
    if cur:
        lines.append(cur)
    return lines


def gradient_bg(canvas: Image.Image, top, bottom):
    draw = ImageDraw.Draw(canvas)
    for y in range(H):
        t = y / max(H - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (W, y)], fill=(r, g, b))


def draw_layer_box(draw, x0, y0, x1, y1, num, label, accent, tag):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=(16, 22, 40), outline=accent, width=2)
    pad = 10
    draw.text((x0 + pad, y0 + pad), f"L{num}", fill=accent, font=font(True, 11), anchor="lt")
    draw.text((x0 + pad + 28, y0 + pad - 1), label, fill=(255, 255, 255), font=font(True, 13), anchor="lt")
    draw.text((x0 + pad, y1 - pad - 12), tag, fill=accent, font=font(True, 9, mono=True), anchor="lt")


def draw_bio_split(draw, x0, y0, x1, y1):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=12, fill=(14, 28, 24), outline=BIO, width=2)
    pad = 14
    draw.text((x0 + pad, y0 + pad), "EXAMPLE: SMART BIO SPECIALIST", fill=BIO, font=font(True, 10), anchor="lt")
    mid = (x0 + x1) // 2
    half_h = y1 - y0
    draw.line([(mid, y0 + 28), (mid, y1 - 10)], fill=(40, 60, 55), width=1)

    # OpenBio side
    draw.text((x0 + pad, y0 + pad + 22), "OpenBio 8B", fill=GREEN, font=font(True, 14), anchor="lt")
    for i, line in enumerate(["Domain chat", "Med reasoning", "Assay context"]):
        draw.text((x0 + pad, y0 + pad + 44 + i * 14), f"• {line}", fill=MUTED, font=font(False, 10), anchor="lt")

    # Qwen side
    draw.text((mid + pad, y0 + pad + 22), "Qwen 2.5 7B", fill=TEAL, font=font(True, 14), anchor="lt")
    for i, line in enumerate(["MCP tool loop", "fold_protein", "analyze_sequence"]):
        draw.text((mid + pad, y0 + pad + 44 + i * 14), f"• {line}", fill=MUTED, font=font(False, 10), anchor="lt")

    draw.text(((x0 + x1) // 2, y1 - pad - 4), "One BiologyExpert · hub picks per turn", fill=BIO, font=font(False, 10), anchor="mm")


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (12, 14, 30), (6, 20, 28))
draw = ImageDraw.Draw(canvas)

badge = "OPEN SOURCE · NEURAL JUNKIE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0, by0 = MARGIN, 28
draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 26), radius=8, fill=(28, 32, 20), outline=TEAL, width=2)
draw.text((bx0 + bw // 2, by0 + 13), badge, fill=TEAL, font=bf, anchor="mm")

head_y = 68
draw.text((MARGIN, head_y), "WE DON'T USE ONE MODEL.", fill=(255, 255, 255), font=font(True, 32), anchor="lt")
draw.text((MARGIN, head_y + 40), "WE LAYER THEM.", fill=AMBER, font=font(True, 36), anchor="lt")

sub_y = head_y + 88
for line in wrap(draw, "Context · Weights · Routing · Orchestration", font(False, 14), CONTENT_W):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=font(False, 14), anchor="lt")
    sub_y += 18

# Four layer boxes in a row
layer_y = sub_y + 8
layer_h = 72
gap = 12
layer_w = (CONTENT_W - 3 * gap) // 4
layers = [
    ("1", "CONTEXT", TEAL, "intent · budget"),
    ("2", "WEIGHTS", GREEN, "14b · 7b · LoRA"),
    ("3", "ROUTING", PINK, "collab · delegate"),
    ("4", "ORCH", AMBER, "summaries · gates"),
]
for i, (num, label, accent, tag) in enumerate(layers):
    x0 = MARGIN + i * (layer_w + gap)
    x1 = x0 + layer_w
    draw_layer_box(draw, x0, layer_y, x1, layer_y + layer_h, num, label, accent, tag)
    if i < 3:
        ax = x1 + gap // 2
        draw.text((ax, layer_y + layer_h // 2), "→", fill=MUTED, font=font(True, 16), anchor="mm")

# Bio example panel
bio_y = layer_y + layer_h + 14
bio_h = 118
draw_bio_split(draw, MARGIN, bio_y, MARGIN + CONTENT_W, bio_y + bio_h)

draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "NEURAL JUNKIE · MODEL-LAYERING-LINKEDIN.md",
    fill=(255, 255, 255),
    font=font(True, 13),
    anchor="mm",
)

OUT.parent.mkdir(parents=True, exist_ok=True)
canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
GALLERY.parent.mkdir(parents=True, exist_ok=True)
canvas.save(GALLERY, "PNG")
print(f"Wrote {GALLERY}")
PY
