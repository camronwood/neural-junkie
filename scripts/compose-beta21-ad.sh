#!/usr/bin/env bash
# Compose neural-junkie-beta21-ad-1200.png — LinkedIn cover (1200×627).
# Usage: ./scripts/compose-beta21-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-beta21-ad-1200.png"
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
    paths = ["/System/Library/Fonts/Menlo.ttc"] if mono else [
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold else "/System/Library/Fonts/Supplemental/Arial.ttf",
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


def draw_pillar(draw, x0, y0, x1, y1, title: str, sub: str, accent, mono: str):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=(18, 24, 44), outline=accent, width=2)
    draw.text((x0 + 18, y0 + 16), title, fill=(255, 255, 255), font=font(True, 18), anchor="lt")
    fs = font(False, 12)
    sy = y0 + 42
    for line in wrap(draw, sub, fs, x1 - x0 - 36)[:3]:
        draw.text((x0 + 18, sy), line, fill=MUTED, font=fs, anchor="lt")
        sy += 16
    draw.text((x0 + 18, y1 - 26), mono, fill=accent, font=font(True, 10, mono=True), anchor="lt")


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (14, 16, 32), (8, 22, 30))
draw = ImageDraw.Draw(canvas)

badge = "v1.0.0-beta.21 · OPEN SOURCE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0, by0 = MARGIN, 32
draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 28), radius=8, fill=(28, 32, 20), outline=TEAL, width=2)
draw.text((bx0 + bw // 2, by0 + 14), badge, fill=TEAL, font=bf, anchor="mm")

head_y = 76
draw.text((MARGIN, head_y), "SLACK INBOX.", fill=(255, 255, 255), font=font(True, 40), anchor="lt")
draw.text((MARGIN, head_y + 46), "CHAT YOU CAN TRUST.", fill=GREEN, font=font(True, 42), anchor="lt")
fsub = font(False, 15)
sub_y = head_y + 98
for line in wrap(draw, "Context model v2, workspace visibility answers, and live scenario regressions before you ship.", fsub, CONTENT_W):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=fsub, anchor="lt")
    sub_y += 20

pillars_y = sub_y + 24
pillar_h = 118
gap = 14
pillar_w = (CONTENT_W - 2 * gap) // 3
pillars = [
    ("Slack → hub", "DM the NJ bot, forward channels with rules, optional away-mode replies in your DMs.", TEAL, "docs/SLACK_INTEGRATION.md"),
    ("Context v2", "Chat/Code mode, turn intent, DM persona, thread history, ~32KB budget.", AMBER, "docs/CONTEXT_MODEL.md"),
    ("Test harness", "make test-all · chat-scenarios · collab-scenario-regression", PINK, "docs/TESTING.md"),
]
for i, (title, sub, accent, mono) in enumerate(pillars):
    x0 = MARGIN + i * (pillar_w + gap)
    draw_pillar(draw, x0, pillars_y, x0 + pillar_w, pillars_y + pillar_h, title, sub, accent, mono)

draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21",
    fill=(255, 255, 255),
    font=font(True, 14),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
