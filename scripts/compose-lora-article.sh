#!/usr/bin/env bash
# Compose neural-junkie-lora-ad-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-lora-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-lora-ad-1200.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$OUT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

OUT = Path(sys.argv[1])
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


def draw_box(draw, x0, y0, x1, y1, title, sub, accent, mono=None):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=12, fill=(18, 24, 44), outline=accent, width=2)
    pad = 14
    inner_w = x1 - x0 - pad * 2
    draw.text((x0 + pad, y0 + pad), title, fill=(255, 255, 255), font=font(True, 15), anchor="lt")
    fs = font(False, 11)
    for i, line in enumerate(wrap(draw, sub, fs, inner_w)[:2]):
        draw.text((x0 + pad, y0 + pad + 24 + i * 14), line, fill=MUTED, font=fs, anchor="lt")
    if mono:
        draw.text((x0 + pad, y1 - pad - 12), mono, fill=accent, font=font(True, 10, mono=True), anchor="lt")


def draw_op(draw, cx, cy, symbol, color):
    draw.text((cx, cy), symbol, fill=color, font=font(True, 24), anchor="mm")


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (14, 16, 32), (8, 22, 30))
draw = ImageDraw.Draw(canvas)

# Badge
badge = "OPEN SOURCE · NEURAL JUNKIE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0 = MARGIN
by0 = 32
draw.rounded_rectangle(
    (bx0, by0, bx0 + bw, by0 + 28),
    radius=8,
    fill=(28, 32, 20),
    outline=TEAL,
    width=2,
)
draw.text((bx0 + bw // 2, by0 + 14), badge, fill=TEAL, font=bf, anchor="mm")

# Headline block
head_y = 76
draw.text((MARGIN, head_y), "ONE BASE MODEL.", fill=(255, 255, 255), font=font(True, 38), anchor="lt")
draw.text((MARGIN, head_y + 44), "MANY SPECIALISTS.", fill=GREEN, font=font(True, 42), anchor="lt")
fsub = font(False, 15)
sub = "Import, compose, or train LoRA adapters — then assign them to your agents."
sub_y = head_y + 96
for line in wrap(draw, sub, fsub, CONTENT_W):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=fsub, anchor="lt")
    sub_y += 20

# Pipeline + paths share a 3-column grid
COL_GAP = 14
COL_W = (CONTENT_W - 2 * COL_GAP) // 3
col_x = [MARGIN + i * (COL_W + COL_GAP) for i in range(3)]

pipe_h = 90
pipe_y = sub_y + 14

boxes_pipe = [
    ("Base model", "Full weights in Ollama", TEAL, "qwen2.5-coder:14b"),
    ("LoRA adapter", "Small HF delta · tens of MB", AMBER, "adapter_model.safetensors"),
    ("Composed tag", "ollama create · assign to agent", GREEN, "nj-security:14b"),
]
for i, (title, sub, accent, mono) in enumerate(boxes_pipe):
    x0 = col_x[i]
    draw_box(draw, x0, pipe_y, x0 + COL_W, pipe_y + pipe_h, title, sub, accent, mono)

# Operators centered in gaps between columns
mid_y = pipe_y + pipe_h // 2
draw_op(draw, col_x[0] + COL_W + COL_GAP // 2, mid_y, "+", AMBER)
draw_op(draw, col_x[1] + COL_W + COL_GAP // 2, mid_y, "=", PINK)

# Three path boxes — same column positions
path_h = 94
paths_y = pipe_y + pipe_h + 18
paths = [
    ("Import", "Hugging Face → Model Library → Compose", TEAL, "Download · Compose"),
    ("Pack presets", "Security, biology — one API call", PINK, "install-loras"),
    ("Train yours", "Chat · collab · repo → Unsloth → Ollama", ACCENT, "Train LoRA tab"),
]
for i, (title, sub, accent, mono) in enumerate(paths):
    x0 = col_x[i]
    draw_box(draw, x0, paths_y, x0 + COL_W, paths_y + path_h, title, sub, accent, mono)

# Data strip — full content width
strip_h = 64
strip_y = paths_y + path_h + 16
draw.rounded_rectangle(
    (MARGIN, strip_y, MARGIN + CONTENT_W, strip_y + strip_h),
    radius=12,
    fill=(20, 28, 40),
    outline=GREEN,
    width=2,
)
draw.text(
    (MARGIN + 18, strip_y + 14),
    "Training data you already have",
    fill=GREEN,
    font=font(True, 14),
    anchor="lt",
)
draw.text(
    (MARGIN + 18, strip_y + 38),
    "Channel / DM transcripts  ·  Collaboration task outputs  ·  Repo agent history",
    fill=(210, 214, 224),
    font=font(False, 13),
    anchor="lt",
)

# Footer
draw.rounded_rectangle(
    (MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H),
    radius=12,
    fill=ACCENT,
)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "github.com/camronwood/neural-junkie  ·  LORA_ADAPTERS.md  ·  LORA_TRAINING.md",
    fill=(255, 255, 255),
    font=font(True, 14),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
