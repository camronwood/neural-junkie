#!/usr/bin/env bash
# Compose neural-junkie-collab-craft-ad-1080.png — collab pipeline / engineering rigor.
# Usage: ./scripts/compose-collab-craft-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/collab-craft/creatives/neural-junkie-collab-craft-ad-1080.png"
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
W, H = 1080, 1080
MARGIN = 56

TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
PINK = (199, 125, 255)
ACCENT = (233, 69, 96)
GREEN = (72, 199, 142)
MUTED = (168, 176, 184)
DIM = (136, 140, 168)


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


def draw_pipeline_stage(draw, cx, cy, box_w, box_h, title: str, sub: str, accent, check: bool):
    x0, y0 = cx - box_w // 2, cy - box_h // 2
    x1, y1 = x0 + box_w, y0 + box_h
    draw.rounded_rectangle((x0, y0, x1, y1), radius=12, fill=(18, 24, 44), outline=accent, width=2)
    if check:
        r = 14
        gx, gy = x1 - 18, y0 + 16
        draw.ellipse((gx - r, gy - r, gx + r, gy + r), fill=GREEN, outline=GREEN)
        draw.line([(gx - 5, gy), (gx - 1, gy + 5), (gx + 6, gy - 4)], fill=(255, 255, 255), width=2)
    draw.text((cx, y0 + 22), title, fill=(255, 255, 255), font=font(True, 16), anchor="mm")
    fs = font(False, 11)
    for i, line in enumerate(wrap(draw, sub, fs, box_w - 20)[:2]):
        draw.text((cx, y0 + 42 + i * 16), line, fill=MUTED, font=fs, anchor="mm")


def draw_arrow(draw, x0, y, x1, color):
    mid_y = y
    draw.line([(x0, mid_y), (x1 - 8, mid_y)], fill=color, width=3)
    draw.polygon([(x1, mid_y), (x1 - 10, mid_y - 5), (x1 - 10, mid_y + 5)], fill=color)


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (14, 16, 32), (8, 20, 28))
draw = ImageDraw.Draw(canvas)

badge = "BETA.18 · COLLABORATION HARDENING"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0 = (W - bw) // 2
by0 = 44
draw.rounded_rectangle(
    (bx0, by0, bx0 + bw, by0 + 30),
    radius=8,
    fill=(28, 32, 20),
    outline=AMBER,
    width=2,
)
draw.text((W // 2, by0 + 15), badge, fill=AMBER, font=bf, anchor="mm")

y = 92
draw.text((W // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 26), anchor="mm")
y += 44
draw.text((W // 2, y), "COLLAB THAT", fill=(255, 255, 255), font=font(True, 46), anchor="mm")
y += 50
draw.text((W // 2, y), "HOLDS UP", fill=GREEN, font=font(True, 52), anchor="mm")
y += 48
sub = "Planning → review → execute — tested, gated, bounded."
fsub = font(False, 17)
for line in wrap(draw, sub, fsub, W - 2 * MARGIN):
    draw.text((W // 2, y), line, fill=(200, 205, 220), font=fsub, anchor="mm")
    y += 22

y += 20
draw.text((MARGIN, y), "The pipeline (not a demo)", fill=MUTED, font=font(True, 13), anchor="lt")
y += 32

# Horizontal pipeline — 5 stages
stages = [
    ("Planning", "Bounded turns · collab channel", TEAL, True),
    ("Review", "You approve the plan", AMBER, True),
    ("Gate", "Workspace ack · Continue", PINK, True),
    ("Execute", "DAG tasks · ready waves", ACCENT, True),
    ("Deliver", "Sandbox or worktree", GREEN, True),
]
n = len(stages)
box_w, box_h = 168, 88
gap = 12
total_w = n * box_w + (n - 1) * gap
start_x = (W - total_w) // 2 + box_w // 2
cy = y + box_h // 2 + 8
prev_right = None
for i, (title, sub, accent, check) in enumerate(stages):
    cx = start_x + i * (box_w + gap)
    draw_pipeline_stage(draw, cx, cy, box_w, box_h, title, sub, accent, check)
    if prev_right is not None:
        draw_arrow(draw, prev_right + 4, cy, cx - box_w // 2 - 4, (80, 88, 110))
    prev_right = cx + box_w // 2

y = cy + box_h // 2 + 28
# beta.18 fixes strip
strip_y = y
strip_h = 96
draw.rounded_rectangle(
    (MARGIN, strip_y, W - MARGIN, strip_y + strip_h),
    radius=14,
    fill=(20, 28, 40),
    outline=TEAL,
    width=2,
)
draw.text(
    (MARGIN + 20, strip_y + 18),
    "What we're shipping in beta.18",
    fill=TEAL,
    font=font(True, 15),
    anchor="lt",
)
fixes = [
    "Source repo binding — real git root, not wrong sandboxes",
    "Task parsing from plans · multi-collab phase routing",
    "make collab-smoke — planning → review → execute with live agents",
]
fy = strip_y + 42
ff = font(False, 14)
for fix in fixes:
    draw.text((MARGIN + 28, fy), "✓", fill=GREEN, font=font(True, 14), anchor="lt")
    draw.text((MARGIN + 48, fy), fix, fill=(210, 214, 224), font=ff, anchor="lt")
    fy += 22

y = strip_y + strip_h + 24
draw.text(
    (W // 2, y),
    "If collab broke on you before — try beta.18 and tell us what still fails.",
    fill=DIM,
    font=font(False, 13),
    anchor="mm",
)

draw.rounded_rectangle(
    (MARGIN, 868, W - MARGIN, 868 + 56),
    radius=14,
    fill=ACCENT,
)
draw.text(
    (W // 2, 868 + 28),
    "github.com/camronwood/neural-junkie/releases",
    fill=(255, 255, 255),
    font=font(True, 20),
    anchor="mm",
)
draw.text(
    (W // 2, 944),
    "make collab-smoke · open source · macOS · Windows · Linux",
    fill=DIM,
    font=font(False, 14),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
