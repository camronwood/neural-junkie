#!/usr/bin/env bash
# Compose neural-junkie-fix-growth-loops-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-fix-growth-loops-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-fix-growth-loops-1200.png"
GALLERY="$ROOT/docs/media/articles/covers/neural-junkie-fix-growth-loops-1200.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

mkdir -p "$(dirname "$GALLERY")"
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


def gradient_bg(canvas, top, bottom):
    draw = ImageDraw.Draw(canvas)
    for y in range(H):
        t = y / max(H - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (W, y)], fill=(r, g, b))


def draw_loop_box(draw, x0, y0, x1, y1, num: str, title: str, sub: str, accent, mono_line: str):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=(18, 24, 44), outline=accent, width=2)
    badge_r = 17
    badge_cx = x0 + 28
    badge_cy = y0 + (y1 - y0) // 2
    draw.ellipse(
        (badge_cx - badge_r, badge_cy - badge_r, badge_cx + badge_r, badge_cy + badge_r),
        fill=accent,
    )
    draw.text((badge_cx, badge_cy), num, fill=(255, 255, 255), font=font(True, 15), anchor="mm")
    text_x = x0 + 56
    text_w = x1 - text_x - 16
    draw.text((text_x, y0 + 16), title, fill=(255, 255, 255), font=font(True, 17), anchor="lt")
    fs = font(False, 11)
    for i, line in enumerate(wrap(draw, sub, fs, text_w)[:2]):
        draw.text((text_x, y0 + 38 + i * 15), line, fill=MUTED, font=fs, anchor="lt")
    draw.text((text_x, y1 - 22), mono_line, fill=accent, font=font(True, 10, mono=True), anchor="lt")


def compose(path: Path) -> None:
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (14, 16, 32), (8, 22, 30))
    draw = ImageDraw.Draw(canvas)

    badge = "OPEN SOURCE · NEURAL JUNKIE · RELEASE ENGINEERING"
    bf = font(True, 11)
    draw.text((MARGIN, 32), badge, fill=TEAL, font=bf, anchor="lt")

    head_y = 68
    draw.text((MARGIN, head_y), "GREEN TESTS", fill=(255, 255, 255), font=font(True, 40), anchor="lt")
    draw.text((MARGIN, head_y + 46), "AREN'T ENOUGH", fill=GREEN, font=font(True, 42), anchor="lt")
    fsub = font(False, 15)
    sub_y = head_y + 98
    for line in wrap(draw, "Gate → fix → grow: release loops that keep an agent platform shippable.", fsub, CONTENT_W):
        draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sub_y += 20

    box_h = 86
    gap = 10
    layers_y = FOOTER_Y - 16 - (3 * box_h + 2 * gap)
    box_w = CONTENT_W

    loops = [
        ("1", "layer-gate", "Know what broke — one surface at a time", TEAL, "make layer-gate LAYER=collab-full"),
        ("2", "layer-fix-loop", "Repair product code — agent brief → commit → rerun", AMBER, "make layer-fix-loop LAYER=chat"),
        ("3", "test-growth-loop", "Strengthen the contract — discover gaps → add tests", PINK, "make test-growth-loop"),
    ]
    for i, (num, title, sub, accent, mono) in enumerate(loops):
        y0 = layers_y + i * (box_h + gap)
        draw_loop_box(draw, MARGIN, y0, MARGIN + box_w, y0 + box_h, num, title, sub, accent, mono)

    draw.rounded_rectangle(
        (MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H),
        radius=12,
        fill=ACCENT,
    )
    draw.text(
        (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
        "github.com/camronwood/neural-junkie  ·  TESTING.md  ·  layer-gate · fix-loop · test-growth",
        fill=(255, 255, 255),
        font=font(True, 13),
        anchor="mm",
    )
    canvas.save(path, "PNG")


compose(OUT)
compose(GALLERY)
print(f"Wrote {OUT}")
print(f"Wrote {GALLERY}")
PY
