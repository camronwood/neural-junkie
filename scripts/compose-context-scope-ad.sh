#!/usr/bin/env bash
# Compose neural-junkie-context-scope-ad-1080.png — selective workspace context (no screenshots).
# Usage: ./scripts/compose-context-scope-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-context-scope-ad-1080.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$OUT" <<'PY'
import random
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

OUT = Path(sys.argv[1])

W, H = 1080, 1080
BG = (13, 13, 26)
PANEL = (18, 24, 44)
PANEL_EDGE = (48, 58, 88)
BOX_FILL = (24, 34, 58)
BOX_EDGE = (83, 52, 131)
ACCENT = (233, 69, 96)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
PINK = (199, 125, 255)
GREEN = (72, 199, 142)
NET_LINE = (42, 38, 72)
NET_NODE = (110, 70, 150)


def font(bold: bool, size: int):
    path = (
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf"
        if bold
        else "/System/Library/Fonts/Supplemental/Arial.ttf"
    )
    try:
        return ImageFont.truetype(path, size)
    except OSError:
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


def draw_network(draw: ImageDraw.ImageDraw, rng: random.Random):
    nodes = [(rng.randint(30, W - 30), rng.randint(24, 200)) for _ in range(14)]
    for i, (x1, y1) in enumerate(nodes):
        for x2, y2 in nodes[i + 1 :]:
            if (x1 - x2) ** 2 + (y1 - y2) ** 2 < 140**2:
                draw.line([(x1, y1), (x2, y2)], fill=NET_LINE, width=1)
    for x, y in nodes:
        r = 3
        draw.ellipse((x - r, y - r, x + r, y + r), fill=NET_NODE)


def draw_brand(draw: ImageDraw.ImageDraw):
    draw.text((W // 2, 44), "FOR DEVELOPERS", fill=MUTED, font=font(False, 13), anchor="mm")
    title = "NEURAL JUNKIE"
    ft = font(True, 52)
    tw = draw.textlength(title, font=ft)
    tx = (W - tw) // 2
    draw.text((tx, 72), title, fill=TEXT, font=ft)
    draw.rectangle((tx, 132, tx + tw, 136), fill=ACCENT)


def draw_scope_ladder(draw: ImageDraw.ImageDraw, y0: int):
    """Visual: none → hint → outline → focus → full"""
    labels = ["none", "hint", "outline", "focus", "full"]
    colors = [
        (60, 60, 80),
        (80, 90, 120),
        (100, 120, 160),
        (130, 100, 180),
        (199, 125, 255),
    ]
    bar_w = 720
    x0 = (W - bar_w) // 2
    step = bar_w // (len(labels) - 1)
    for i, (lab, col) in enumerate(zip(labels, colors)):
        x = x0 + i * step
        r = 14 if i < len(labels) - 1 else 18
        draw.ellipse((x - r, y0 - r, x + r, y0 + r), fill=col, outline=BOX_EDGE, width=1)
        if i < len(labels) - 1:
            draw.line([(x + r, y0), (x + step - r, y0)], fill=BOX_EDGE, width=3)
        draw.text((x, y0 + 28), lab, fill=MUTED if i < 3 else TEXT, font=font(False, 14), anchor="mm")
    draw.text((W // 2, y0 - 36), "context_scope tiers", fill=MUTED, font=font(False, 15), anchor="mm")


def draw_box(draw, xy, title: str, body: str):
    draw.rounded_rectangle(xy, radius=12, fill=BOX_FILL, outline=BOX_EDGE, width=1)
    x0, y0, x1, y1 = xy
    pad = 14
    draw.text((x0 + pad, y0 + pad), title, fill=PINK, font=font(True, 17))
    f = font(False, 14)
    lines = wrap(draw, body, f, x1 - x0 - 2 * pad)
    ty = y0 + pad + 24
    for line in lines[:4]:
        draw.text((x0 + pad, ty), line, fill=TEXT, font=f)
        ty += 18


canvas = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(canvas)
draw_network(draw, random.Random(19))
draw_brand(draw)

panel = (48, 208, 1032, 918)
draw.rounded_rectangle(panel, radius=18, fill=PANEL, outline=PANEL_EDGE, width=1)

cx = W // 2
draw.text((cx, 248), "Not every question needs your whole repo.", fill=MUTED, font=font(False, 22), anchor="mm")
draw.text((cx, 292), "Smart workspace context", fill=PINK, font=font(True, 38), anchor="mm")
fsub = font(False, 19)
sub = "Auto picks none · hint · outline · focus · full per message — or pin Always / Off in the composer"
for i, line in enumerate(wrap(draw, sub, fsub, panel[2] - panel[0] - 80)[:2]):
    draw.text((cx, 332 + i * 26), line, fill=TEXT, font=fsub, anchor="mm")

draw_scope_ladder(draw, 400)

margin, gap_b = 64, 16
inner_l, inner_r = panel[0] + 16, panel[2] - 16
bw = (inner_r - inner_l - gap_b) // 2
bh = 150
y0 = 458
boxes = [
    (
        (inner_l, y0, inner_l + bw, y0 + bh),
        "Auto (default)",
        "General chat stays light. Open files and @paths upgrade to focus or full only when you need them.",
    ),
    (
        (inner_l + bw + gap_b, y0, inner_r, y0 + bh),
        "Composer chip",
        "See the resolved scope before you send. Toggle workspace context: Auto · Always · Off.",
    ),
    (
        (inner_l, y0 + bh + gap_b, inner_l + bw, y0 + 2 * bh + gap_b),
        "Collab stays clean",
        "/collaborate no longer leaks open editors. Add --workspace for outline-only project tree during planning.",
    ),
    (
        (inner_l + bw + gap_b, y0 + bh + gap_b, inner_r, y0 + 2 * bh + gap_b),
        "Session-safe",
        "Slimmer collab payloads, smaller last-session.json, hub data consent — beta.6 stability bundle.",
    ),
]
for xy, title, body in boxes:
    draw_box(draw, xy, title, body)

draw.text((cx, 838), "v1.0.0-beta.6  ·  open source desktop beta", fill=GREEN, font=font(False, 16), anchor="mm")

draw.rounded_rectangle((120, 868, 960, 922), radius=14, fill=ACCENT)
draw.text(
    (cx, 895),
    "Download — macOS · Windows · Linux",
    fill=TEXT,
    font=font(True, 21),
    anchor="mm",
)
draw.text(
    (cx, 958),
    "github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.6",
    fill=MUTED,
    font=font(False, 15),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
