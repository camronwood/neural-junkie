#!/usr/bin/env bash
# Compose neural-junkie-community-ad-1080.png — community / early hive invitation.
# Usage: ./scripts/compose-community-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/community/creatives/neural-junkie-community-ad-1080.png"
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


def draw_issue_card(draw, box, label: str, title: str, meta: str, accent):
    x0, y0, x1, y1 = box
    pad = 16
    draw.rounded_rectangle(box, radius=12, fill=(18, 24, 44), outline=(48, 58, 88), width=1)
    draw.rounded_rectangle(
        (x0 + pad, y0 + pad, x0 + pad + 8, y1 - pad),
        radius=4,
        fill=accent,
    )
    lx = x0 + pad + 18
    draw.text((lx, y0 + pad), label, fill=accent, font=font(True, 10), anchor="lt")
    tf = font(True, 15)
    lines = wrap(draw, title, tf, x1 - lx - pad)[:2]
    ty = y0 + pad + 16
    for line in lines:
        draw.text((lx, ty), line, fill=(240, 242, 248), font=tf, anchor="lt")
        ty += 20
    draw.text((lx, y1 - pad - 14), meta, fill=DIM, font=font(False, 11), anchor="lt")


def draw_cta_pill(draw, cx, cy, icon: str, label: str, accent):
    f = font(True, 14)
    text = f"{icon}  {label}"
    tw = int(draw.textlength(text, font=f))
    pw, ph = tw + 28, 44
    x0, y0 = cx - pw // 2, cy - ph // 2
    draw.rounded_rectangle(
        (x0, y0, x0 + pw, y0 + ph),
        radius=10,
        fill=(24, 30, 52),
        outline=accent,
        width=2,
    )
    draw.text((cx, cy), text, fill=(255, 255, 255), font=f, anchor="mm")


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (12, 18, 36), (8, 22, 32))
draw = ImageDraw.Draw(canvas)

# OPEN BETA badge
badge = "OPEN BETA · BUILD IN PUBLIC"
bf = font(True, 12)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0 = (W - bw) // 2
by0 = 44
draw.rounded_rectangle(
    (bx0, by0, bx0 + bw, by0 + 32),
    radius=8,
    fill=(20, 36, 48),
    outline=TEAL,
    width=2,
)
draw.text((W // 2, by0 + 16), badge, fill=TEAL, font=bf, anchor="mm")

y = 96
draw.text((W // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 26), anchor="mm")
y += 44
draw.text((W // 2, y), "YOUR WORKFLOW", fill=(255, 255, 255), font=font(True, 48), anchor="mm")
y += 52
draw.text((W // 2, y), "OUR ROADMAP", fill=AMBER, font=font(True, 48), anchor="mm")
y += 48
sub = "Open beta. Star it, break it, tell us what you're building."
fsub = font(False, 17)
for line in wrap(draw, sub, fsub, W - 2 * MARGIN):
    draw.text((W // 2, y), line, fill=(200, 205, 220), font=fsub, anchor="mm")
    y += 22

y += 16
draw.text(
    (MARGIN, y),
    "What builders are talking about",
    fill=MUTED,
    font=font(True, 13),
    anchor="lt",
)
y += 28

card_h = 72
gap = 10
cards = [
    ("feedback", "Slack + local Ollama on Windows — does your setup work?", "issue · 12 comments", TEAL),
    ("idea", "Runbook template for vendor review (non-dev workflow)", "discussion · welcome", AMBER),
    ("question", "Who uses BiologyExpert daily? Domain pack feedback", "life sciences", GREEN),
    ("show-and-tell", "Collab channel screenshot — multi-agent planning IRL", "community", PINK),
]
for label, title, meta, accent in cards:
    draw_issue_card(draw, (MARGIN, y, W - MARGIN, y + card_h), label, title, meta, accent)
    y += card_h + gap

y += 8
pill_y = y + 28
pill_gap = 16
labels = [
    ("★", "Star", AMBER),
    ("💬", "Share workflow", TEAL),
    ("↓", "Try beta.18", GREEN),
]
# measure total width for centering
pill_widths = []
for icon, lab, _ in labels:
    text = f"{icon}  {lab}"
    pill_widths.append(int(draw.textlength(text, font=font(True, 14))) + 28)
total = sum(pill_widths) + 2 * pill_gap
x = (W - total) // 2
for (icon, lab, accent), pw in zip(labels, pill_widths):
    cx = x + pw // 2
    draw_cta_pill(draw, cx, pill_y, icon, lab, accent)
    x += pw + pill_gap

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
    "MIT · local-first · macOS · Windows · Linux",
    fill=DIM,
    font=font(False, 14),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
