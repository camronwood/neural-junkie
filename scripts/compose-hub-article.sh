#!/usr/bin/env bash
# Compose neural-junkie-hub-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-hub-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-hub-1200.png"
GALLERY="$ROOT/docs/media/gallery/ads/neural-junkie-hub-1200.png"
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
HUB = (233, 69, 96)


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


def draw_client_node(draw, cx, cy, label, sub, accent):
    w, h = 108, 52
    x0, y0 = cx - w // 2, cy - h // 2
    draw.rounded_rectangle((x0, y0, x0 + w, y0 + h), radius=8, fill=(14, 20, 36), outline=accent, width=2)
    draw.text((cx, cy - 8), label, fill=(255, 255, 255), font=font(True, 11), anchor="mm")
    draw.text((cx, cy + 10), sub, fill=accent, font=font(False, 8, mono=True), anchor="mm")


def draw_hub_core(draw, x0, y0, x1, y1):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=16, fill=(20, 12, 28), outline=HUB, width=3)
    cx = (x0 + x1) // 2
    draw.text((cx, y0 + 18), "GO HUB", fill=HUB, font=font(True, 13), anchor="mm")
    draw.text((cx, y0 + 36), "127.0.0.1:18765", fill=MUTED, font=font(False, 9, mono=True), anchor="mm")

    modules = [
        ("channels", TEAL),
        ("agents", GREEN),
        ("collab", PINK),
        ("files", AMBER),
        ("routing", BLUE),
    ]
    mw = (x1 - x0 - 24) // 5
    my = y0 + 52
    mh = 36
    for i, (name, col) in enumerate(modules):
        mx0 = x0 + 12 + i * mw
        mx1 = mx0 + mw - 4
        draw.rounded_rectangle((mx0, my, mx1, my + mh), radius=6, fill=(12, 16, 30), outline=col, width=1)
        draw.text(((mx0 + mx1) // 2, my + mh // 2), name, fill=col, font=font(True, 8), anchor="mm")

    draw.text((cx, y1 - 14), "orchestrator · state · policy", fill=MUTED, font=font(False, 9), anchor="mm")


def draw_arrow(draw, x0, y0, x1, y1, color=MUTED):
    draw.line([(x0, y0), (x1, y1)], fill=color, width=2)
  # arrowhead
    if abs(x1 - x0) > abs(y1 - y0):
        dx = 6 if x1 > x0 else -6
        draw.polygon([(x1, y1), (x1 - dx, y1 - 4), (x1 - dx, y1 + 4)], fill=color)
    else:
        dy = 6 if y1 > y0 else -6
        draw.polygon([(x1, y1), (x1 - 4, y1 - dy), (x1 + 4, y1 - dy)], fill=color)


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (14, 10, 28), (8, 18, 32))
draw = ImageDraw.Draw(canvas)

badge = "OPEN SOURCE · NEURAL JUNKIE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0, by0 = MARGIN, 28
draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 26), radius=8, fill=(28, 20, 32), outline=HUB, width=2)
draw.text((bx0 + bw // 2, by0 + 13), badge, fill=HUB, font=bf, anchor="mm")

head_y = 68
draw.text((MARGIN, head_y), "THE HUB IS THE PRODUCT.", fill=(255, 255, 255), font=font(True, 30), anchor="lt")
draw.text((MARGIN, head_y + 38), "NOT ANOTHER CHAT UI.", fill=TEAL, font=font(True, 32), anchor="lt")

sub_y = head_y + 84
for line in wrap(draw, "One Go server · many clients · local-first orchestration", font(False, 14), CONTENT_W):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=font(False, 14), anchor="lt")
    sub_y += 18

# Diagram area
diag_y = sub_y + 6
hub_x0 = MARGIN + 200
hub_y0 = diag_y + 10
hub_x1 = MARGIN + CONTENT_W - 200
hub_y1 = diag_y + 130
draw_hub_core(draw, hub_x0, hub_y0, hub_x1, hub_y1)
hub_cx = (hub_x0 + hub_x1) // 2
hub_cy = (hub_y0 + hub_y1) // 2

clients = [
    (MARGIN + 90, hub_y0 + 20, "Desktop", "Tauri", TEAL),
    (MARGIN + 90, hub_cy, "Browser", "chat", GREEN),
    (MARGIN + 90, hub_y1 - 20, "Slack", "bridge", PINK),
    (MARGIN + CONTENT_W - 90, hub_y0 + 20, "CLI", "HTTP", AMBER),
    (MARGIN + CONTENT_W - 90, hub_cy, "Terminal", "WS", BLUE),
    (MARGIN + CONTENT_W - 90, hub_y1 - 20, "Agents", "in-proc", GREEN),
]
for cx, cy, label, sub, col in clients:
    draw_client_node(draw, cx, cy, label, sub, col)
    if cx < hub_cx:
        draw_arrow(draw, cx + 54, cy, hub_x0 - 4, cy, col)
    else:
        draw_arrow(draw, hub_x1 + 4, cy, cx - 54, cy, col)

# Bottom strip: what hub owns
strip_y = hub_y1 + 16
strip_h = 56
draw.rounded_rectangle((MARGIN, strip_y, MARGIN + CONTENT_W, strip_y + strip_h), radius=10, fill=(12, 18, 32), outline=TEAL, width=1)
items = ["@mentions", "file approvals", "collab phases", "model routing", "workspaces", "session state"]
gap = 8
item_w = (CONTENT_W - 24 - 5 * gap) // 6
for i, item in enumerate(items):
    ix0 = MARGIN + 12 + i * (item_w + gap)
    draw.rounded_rectangle((ix0, strip_y + 10, ix0 + item_w, strip_y + strip_h - 10), radius=6, fill=(16, 22, 40), outline=MUTED, width=1)
    draw.text((ix0 + item_w // 2, strip_y + strip_h // 2), item, fill=(220, 225, 235), font=font(False, 9), anchor="mm")

draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "NEURAL JUNKIE · HUB-LINKEDIN.md",
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
