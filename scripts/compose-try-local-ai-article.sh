#!/usr/bin/env bash
# Compose neural-junkie-try-local-ai-1200.png — LinkedIn / article cover (1200×627).
# Usage: ./scripts/compose-try-local-ai-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-try-local-ai-1200.png"
GALLERY="$ROOT/docs/media/articles/covers/neural-junkie-try-local-ai-1200.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$OUT" "$GALLERY" <<'PY'
import shutil
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
ACCENT = (233, 69, 96)
GREEN = (72, 199, 142)
MUTED = (168, 176, 184)


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


def draw_benefit_box(draw, x0, y0, x1, y1, title, detail, accent):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=12, fill=(18, 24, 44), outline=accent, width=2)
    pad = 14
    draw.text((x0 + pad, y0 + pad), title, fill=(255, 255, 255), font=font(True, 20), anchor="lt")
    for i, line in enumerate(wrap(draw, detail, font(False, 11), x1 - x0 - 2 * pad)):
        draw.text((x0 + pad, y0 + pad + 30 + i * 15), line, fill=accent, font=font(False, 11), anchor="lt")


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (14, 16, 32), (8, 22, 30))
draw = ImageDraw.Draw(canvas)

badge = "OPEN SOURCE · NEURAL JUNKIE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0 = MARGIN
by0 = 32
draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 28), radius=8, fill=(28, 32, 20), outline=TEAL, width=2)
draw.text((bx0 + bw // 2, by0 + 14), badge, fill=TEAL, font=bf, anchor="mm")

head_y = 76
draw.text((MARGIN, head_y), "RUN AI LOCALLY.", fill=(255, 255, 255), font=font(True, 38), anchor="lt")
draw.text((MARGIN, head_y + 44), "ON YOUR MACHINE.", fill=GREEN, font=font(True, 42), anchor="lt")
draw.text((MARGIN, head_y + 88), "TRY IT TODAY.", fill=AMBER, font=font(True, 34), anchor="lt")

sub_y = head_y + 132
fsub = font(False, 14)
for line in wrap(
    draw,
    "Install Neural Junkie, pull a model once, and chat with a multi-agent team — no cloud required.",
    fsub,
    CONTENT_W,
):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=fsub, anchor="lt")
    sub_y += 18

# Three-step flow
bar_y = sub_y + 12
bar_h = 36
step_w = (CONTENT_W - 28) // 3
steps = [
    ("1. Download", "~15 MB app", GREEN),
    ("2. Pull once", "Ollama bundled", TEAL),
    ("3. Chat local", "Specialists collaborate", AMBER),
]
for i, (label, detail, accent) in enumerate(steps):
    x0 = MARGIN + i * (step_w + 14)
    x1 = x0 + step_w
    draw.rounded_rectangle((x0, bar_y, x1, bar_y + bar_h), radius=8, fill=(18, 24, 44), outline=accent, width=2)
    draw.text((x0 + step_w // 2, bar_y + 11), label, fill=(255, 255, 255), font=font(True, 12), anchor="mm")
    draw.text((x0 + step_w // 2, bar_y + 26), detail, fill=accent, font=font(True, 10), anchor="mm")

# Three benefit boxes
COL_GAP = 14
COL_W = (CONTENT_W - 2 * COL_GAP) // 3
col_x = [MARGIN + i * (COL_W + COL_GAP) for i in range(3)]
tier_y = bar_y + bar_h + 20
tier_h = 88
benefits = [
    ("Your data stays", "Code and prompts on your hardware", TEAL),
    ("No token bill", "Local inference by default", GREEN),
    ("Hybrid when you want", "Add cloud keys for burst capacity", AMBER),
]
for i, (title, detail, accent) in enumerate(benefits):
    draw_benefit_box(draw, col_x[i], tier_y, col_x[i] + COL_W, tier_y + tier_h, title, detail, accent)

note_y = tier_y + tier_h + 16
draw.rounded_rectangle((MARGIN, note_y, MARGIN + CONTENT_W, note_y + 52), radius=12, fill=(20, 28, 40), outline=MUTED, width=1)
draw.text(
    (MARGIN + 18, note_y + 16),
    "macOS · Windows · Linux — open beta · Ollama runtime bundled on macOS",
    fill=(210, 214, 224),
    font=font(False, 13),
    anchor="lt",
)

draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "github.com/camronwood/neural-junkie/releases  ·  TRY LOCAL AI",
    fill=(255, 255, 255),
    font=font(True, 14),
    anchor="mm",
)

canvas.save(OUT, "PNG")
GALLERY.parent.mkdir(parents=True, exist_ok=True)
shutil.copy2(OUT, GALLERY)
print(f"Wrote {OUT}")
print(f"Wrote {GALLERY}")
PY
