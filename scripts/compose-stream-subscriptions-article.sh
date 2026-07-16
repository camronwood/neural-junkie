#!/usr/bin/env bash
# Compose neural-junkie-stream-subscriptions-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-stream-subscriptions-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/stream-subscriptions/creatives/neural-junkie-stream-subscriptions-1200.png"
GALLERY="$ROOT/docs/media/articles/covers/neural-junkie-stream-subscriptions-1200.png"
ADS="$ROOT/docs/media/gallery/ads/neural-junkie-stream-subscriptions-1200.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$OUT" "$GALLERY" "$ADS" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

OUT = Path(sys.argv[1])
GALLERY = Path(sys.argv[2])
ADS = Path(sys.argv[3])
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
MQTT = (72, 180, 200)
KAFKA = (255, 193, 94)


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


def box(draw, x0, y0, x1, y1, title, subtitle, accent):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=(14, 20, 36), outline=accent, width=2)
    cx = (x0 + x1) // 2
    cy = (y0 + y1) // 2
    draw.text((cx, cy - 8), title, fill=(255, 255, 255), font=font(True, 13), anchor="mm")
    draw.text((cx, cy + 12), subtitle, fill=accent, font=font(False, 10, mono=True), anchor="mm")


def arrow(draw, x0, y0, x1, y1, color=MUTED):
    draw.line([(x0, y0), (x1, y1)], fill=color, width=2)
    dx = 6 if x1 > x0 else -6
    draw.polygon([(x1, y1), (x1 - dx, y1 - 4), (x1 - dx, y1 + 4)], fill=color)


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (12, 18, 34), (8, 14, 26))
draw = ImageDraw.Draw(canvas)

badge = "OPEN SOURCE · NEURAL JUNKIE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0, by0 = MARGIN, 28
draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 26), radius=8, fill=(28, 20, 32), outline=ACCENT, width=2)
draw.text((bx0 + bw // 2, by0 + 13), badge, fill=ACCENT, font=bf, anchor="mm")

head_y = 68
draw.text((MARGIN, head_y), "STREAMS IN. AGENTS OUT.", fill=(255, 255, 255), font=font(True, 30), anchor="lt")
draw.text((MARGIN, head_y + 38), "MQTT · KAFKA → RUNBOOKS", fill=TEAL, font=font(True, 28), anchor="lt")

sub_y = head_y + 82
for line in wrap(draw, "Long-lived consumers. Match once. Fire a runbook, channel post, or webhook.", font(False, 14), CONTENT_W):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=font(False, 14), anchor="lt")
    sub_y += 18

diag_y = sub_y + 14
# Sources
box(draw, MARGIN, diag_y, MARGIN + 160, diag_y + 70, "MQTT", "broker", MQTT)
box(draw, MARGIN, diag_y + 90, MARGIN + 160, diag_y + 160, "Kafka", "topic/group", KAFKA)

# Manager
mx0 = MARGIN + 210
mx1 = MARGIN + 430
box(draw, mx0, diag_y + 40, mx1, diag_y + 120, "Stream manager", "match · debounce", PINK)
arrow(draw, MARGIN + 160, diag_y + 35, mx0, diag_y + 70, MQTT)
arrow(draw, MARGIN + 160, diag_y + 125, mx0, diag_y + 90, KAFKA)

# Actions
ax0 = MARGIN + 480
aw = 200
ah = 48
gap = 12
actions = [
    ("Runbook", "Step Functions-style", GREEN),
    ("Hub channel", "always-on agents", BLUE),
    ("Webhook", "outbound HTTP", AMBER),
]
for i, (title, sub, col) in enumerate(actions):
    y0 = diag_y + i * (ah + gap)
    box(draw, ax0, y0, ax0 + aw, y0 + ah, title, sub, col)
    arrow(draw, mx1, diag_y + 80, ax0, y0 + ah // 2, col)

# Right callout
rx0 = ax0 + aw + 24
draw.rounded_rectangle((rx0, diag_y, MARGIN + CONTENT_W, diag_y + 160), radius=12, fill=(16, 22, 40), outline=TEAL, width=1)
lines = [
    "Settings → Streams",
    "connector + subscription",
    "test fire without broker",
    "cap-aware runbook starts",
]
for i, line in enumerate(lines):
    draw.text((rx0 + 16, diag_y + 22 + i * 34), "▸  " + line, fill=(220, 225, 235), font=font(False, 13), anchor="lt")

draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "NEURAL JUNKIE · STREAM-SUBSCRIPTIONS-LINKEDIN.md",
    fill=(255, 255, 255),
    font=font(True, 13),
    anchor="mm",
)

for path in (OUT, GALLERY, ADS):
    path.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(path, "PNG")
    print(f"Wrote {path}")
PY
