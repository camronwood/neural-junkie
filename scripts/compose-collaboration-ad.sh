#!/usr/bin/env bash
# Compose neural-junkie-collaboration-ad-1080.png — collab feature (no screenshots).
# Usage: ./scripts/compose-collaboration-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-collaboration-ad-1080.png"
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
    draw.text((tx, 72), title, fill=TEXT, font=ft, anchor="lt")
    draw.rectangle((tx, 132, tx + tw, 138), fill=ACCENT)
    # Feature pill
    pill = "Collaboration"
    fp = font(True, 16)
    pw = draw.textlength(pill, font=fp) + 28
    px = (W - pw) // 2
    draw.rounded_rectangle((px, 152, px + pw, 182), radius=12, fill=BOX_FILL, outline=BOX_EDGE)
    draw.text((W // 2, 167), pill, fill=PINK, font=fp, anchor="mm")


def draw_workflow(draw: ImageDraw.ImageDraw, y: int):
    steps = ["Discuss", "Plan", "Approve", "Execute"]
    pill_w, arrow_w, gap = 158, 32, 12
    block = len(steps) * pill_w + (len(steps) - 1) * (arrow_w + gap)
    x = (W - block) // 2
    cy = y + 22
    for i, label in enumerate(steps):
        x1 = x + i * (pill_w + arrow_w + gap)
        draw.rounded_rectangle(
            (x1, y, x1 + pill_w, y + 44), radius=10, fill=BOX_FILL, outline=BOX_EDGE, width=1
        )
        draw.text((x1 + pill_w // 2, cy), label, fill=TEXT, font=font(True, 17), anchor="mm")
        if i < len(steps) - 1:
            ax0 = x1 + pill_w + 6
            ax1 = ax0 + arrow_w - 12
            draw.line([(ax0, cy), (ax1, cy)], fill=ACCENT, width=2)
            draw.polygon(
                [(ax1, cy), (ax1 - 7, cy - 5), (ax1 - 7, cy + 5)],
                fill=ACCENT,
            )


def draw_box(draw: ImageDraw.ImageDraw, xy, title: str, body: str):
    x0, y0, x1, y1 = xy
    pad = 20
    draw.rounded_rectangle(xy, radius=14, fill=BOX_FILL, outline=BOX_EDGE, width=1)
    draw.rounded_rectangle((x0 + pad, y0 + 16, x0 + pad + 4, y0 + 36), radius=2, fill=ACCENT)
    draw.text((x0 + pad + 12, y0 + 26), title, fill=TEXT, font=font(True, 18), anchor="lm")
    fbody = font(False, 14)
    max_w = x1 - x0 - 2 * pad
    y = y0 + 50
    for line in wrap(draw, body, fbody, max_w)[:4]:
        draw.text((x0 + pad, y), line, fill=MUTED, font=fbody, anchor="lt")
        y += 18


canvas = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(canvas)
draw_network(draw, random.Random(7))
draw_brand(draw)

# Main content panel (solid — no template bleed)
panel = (48, 208, 1032, 918)
draw.rounded_rectangle(panel, radius=18, fill=PANEL, outline=PANEL_EDGE, width=1)

cx = W // 2
draw.text((cx, 248), "Beyond @mentions and threads.", fill=MUTED, font=font(False, 24), anchor="mm")
draw.text(
    (cx, 288),
    "Run /collaborate",
    fill=PINK,
    font=font(True, 36),
    anchor="mm",
)
fsub = font(False, 20)
sub = "bounded multi-agent sessions with a plan you approve first"
sub_lines = wrap(draw, sub, fsub, panel[2] - panel[0] - 80)
sy = 322
for line in sub_lines[:2]:
    draw.text((cx, sy), line, fill=TEXT, font=fsub, anchor="mm")
    sy += 26

draw_workflow(draw, 368 if len(sub_lines) > 1 else 358)

margin, gap_b = 64, 16
inner_l, inner_r = panel[0] + 16, panel[2] - 16
bw = (inner_r - inner_l - gap_b) // 2
bh = 172
y0 = 428 if len(sub_lines) > 1 else 418
boxes = [
    (
        (inner_l, y0, inner_l + bw, y0 + bh),
        "Start a session",
        "/collaborate @GoExpert @SecurityExpert on a channel. Specialists join a bounded discussion with turn limits.",
    ),
    (
        (inner_l + bw + gap_b, y0, inner_r, y0 + bh),
        "Shared plan",
        "Agents produce one plan artifact. Use /revise-plan until you are ready to approve.",
    ),
    (
        (inner_l, y0 + bh + gap_b, inner_l + bw, y0 + 2 * bh + gap_b),
        "You stay in control",
        "/approve-plan gates execution. Confirm the on-disk sandbox before tasks are assigned.",
    ),
    (
        (inner_l + bw + gap_b, y0 + bh + gap_b, inner_r, y0 + 2 * bh + gap_b),
        "Safe execution",
        "Per-agent tasks, /collab-status, and file-change approvals — not unbounded agent chat.",
    ),
]
for xy, title, body in boxes:
    draw_box(draw, xy, title, body)

draw.text(
    (cx, 838),
    "/collaborate   /approve-plan   /revise-plan   /collab-status",
    fill=MUTED,
    font=font(False, 15),
    anchor="mm",
)

draw.rounded_rectangle((120, 868, 960, 922), radius=14, fill=ACCENT)
draw.text(
    (cx, 895),
    "Open source beta — macOS · Windows · Linux",
    fill=TEXT,
    font=font(True, 21),
    anchor="mm",
)
draw.text(
    (cx, 958),
    "github.com/camronwood/neural-junkie/releases",
    fill=MUTED,
    font=font(False, 16),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
