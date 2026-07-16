#!/usr/bin/env bash
# Compose personal-learning LinkedIn article cover (1200×627) + feed ad (1080×1080).
# Usage: ./scripts/compose-personal-learning-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_HEADER="$ROOT/campaigns/personal-learning/creatives/neural-junkie-personal-learning-1200.png"
OUT_AD="$ROOT/campaigns/personal-learning/creatives/neural-junkie-personal-learning-ad-1080.png"
GALLERY_HEADER="$ROOT/docs/media/gallery/ads/neural-junkie-personal-learning-1200.png"
GALLERY_AD="$ROOT/docs/media/gallery/ads/neural-junkie-personal-learning-ad-1080.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$OUT_HEADER" "$OUT_AD" "$GALLERY_HEADER" "$GALLERY_AD" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

OUT_HEADER = Path(sys.argv[1])
OUT_AD = Path(sys.argv[2])
GALLERY_HEADER = Path(sys.argv[3])
GALLERY_AD = Path(sys.argv[4])

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
            "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold
            else "/System/Library/Fonts/Supplemental/Arial.ttf",
        ]
    for path in paths:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def gradient_bg(canvas: Image.Image, top, bottom):
    draw = ImageDraw.Draw(canvas)
    w, h = canvas.size
    for y in range(h):
        t = y / max(h - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (w, y)], fill=(r, g, b))


def draw_scope_box(draw, x0, y0, x1, y1, label, example, accent):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=(16, 22, 40), outline=accent, width=2)
    pad = 12
    draw.text((x0 + pad, y0 + pad), label, fill=(255, 255, 255), font=font(True, 13), anchor="lt")
    for i, line in enumerate(example.split("\n")):
        draw.text((x0 + pad, y0 + pad + 22 + i * 16), line, fill=accent, font=font(False, 10), anchor="lt")


def compose_header():
    W, H = 1200, 627
    MARGIN = 48
    CONTENT_W = W - 2 * MARGIN
    FOOTER_H = 46
    FOOTER_Y = H - MARGIN - FOOTER_H

    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (14, 12, 32), (8, 22, 30))
    draw = ImageDraw.Draw(canvas)

    badge = "OPT-IN · USER-CONFIRMED"
    bf = font(True, 11)
    bw = int(draw.textlength(badge, font=bf)) + 24
    bx0, by0 = MARGIN, 28
    draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 26), radius=8, fill=(28, 24, 36), outline=PINK, width=2)
    draw.text((bx0 + bw // 2, by0 + 13), badge, fill=PINK, font=bf, anchor="mm")

    head_y = 68
    draw.text((MARGIN, head_y), "RECALL THE PAST.", fill=MUTED, font=font(True, 28), anchor="lt")
    draw.text((MARGIN, head_y + 40), "SHAPE THE FUTURE.", fill=AMBER, font=font(True, 38), anchor="lt")

    sub_y = head_y + 92
    draw.text((MARGIN, sub_y), "Personal learning · Scoped notes · Local embed retrieval", fill=(200, 205, 220), font=font(False, 14), anchor="lt")

    scope_y = sub_y + 32
    scope_h = 96
    gap = 16
    scope_w = (CONTENT_W - 2 * gap) // 3
    scopes = [
        ("THIS EXPERT", "Tabs not spaces\nin Go files", TEAL),
        ("ALL EXPERTS", "pnpm not npm\nfor this repo", PINK),
        ("THIS COLLAB", "JWT + refresh\nin middleware", GREEN),
    ]
    for i, (label, example, accent) in enumerate(scopes):
        x0 = MARGIN + i * (scope_w + gap)
        draw_scope_box(draw, x0, scope_y, x0 + scope_w, scope_y + scope_h, label, example, accent)

    gate_y = scope_y + scope_h + 22
    gate_h = 72
    draw.rounded_rectangle((MARGIN, gate_y, MARGIN + CONTENT_W, gate_y + gate_h), radius=12, fill=(18, 20, 38), outline=AMBER, width=2)
    draw.text((MARGIN + 20, gate_y + 14), "GATES", fill=AMBER, font=font(True, 12), anchor="lt")
    steps = ["Specialist tuning pack", "Settings opt-in", "You approve every save"]
    step_w = CONTENT_W // len(steps)
    for i, step in enumerate(steps):
        cx = MARGIN + step_w * i + step_w // 2
        draw.text((cx, gate_y + 44), step, fill=(220, 225, 235), font=font(True, 11), anchor="mm")
        if i < len(steps) - 1:
            nx = MARGIN + step_w * (i + 1)
            draw.text(((cx + nx - step_w // 2) // 2 + step_w // 4, gate_y + 58), "→", fill=MUTED, font=font(True, 14), anchor="mm")

    draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
    draw.text(
        (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
        "NEURAL JUNKIE · PERSONAL-LEARNING-LINKEDIN.md",
        fill=(255, 255, 255),
        font=font(True, 13),
        anchor="mm",
    )
    return canvas


def compose_ad():
    W, H = 1080, 1080
    MARGIN = 56
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (14, 12, 32), (8, 24, 32))
    draw = ImageDraw.Draw(canvas)

    draw.text((W // 2, 72), "PERSONAL LEARNING", fill=PINK, font=font(True, 14), anchor="mm")
    headline = "Remember that we use\npnpm, not npm."
    y = 120
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=(255, 255, 255), font=font(True, 38), anchor="mm")
        y += 46

    modal_y = 250
    modal_h = 280
    draw.rounded_rectangle((MARGIN, modal_y, W - MARGIN, modal_y + modal_h), radius=16, fill=(18, 22, 42), outline=TEAL, width=3)
    draw.text((MARGIN + 28, modal_y + 24), "Save learning?", fill=TEAL, font=font(True, 16), anchor="lt")
    draft = (
        "Content: Always use pnpm for this monorepo.\n"
        "Scope: All experts\n"
        "Category: workflow"
    )
    ty = modal_y + 64
    for line in draft.split("\n"):
        draw.text((MARGIN + 28, ty), line, fill=(210, 215, 225), font=font(False, 17), anchor="lt")
        ty += 34

    btn_y = modal_y + modal_h - 56
    approve_w = 140
    cancel_w = 100
    cx = W // 2
    draw.rounded_rectangle((cx - approve_w - 12, btn_y, cx - 12, btn_y + 40), radius=8, fill=GREEN)
    draw.text((cx - approve_w // 2 - 12, btn_y + 20), "Approve", fill=(255, 255, 255), font=font(True, 13), anchor="mm")
    draw.rounded_rectangle((cx + 12, btn_y, cx + 12 + cancel_w, btn_y + 40), radius=8, fill=(32, 36, 52), outline=MUTED, width=1)
    draw.text((cx + 12 + cancel_w // 2, btn_y + 20), "Cancel", fill=MUTED, font=font(True, 13), anchor="mm")

    chips = ["Nothing auto-saved", "Local embed", "→ LoRA optional"]
    ty = modal_y + modal_h + 36
    for chip in chips:
        tw = int(draw.textlength(chip, font=font(True, 12))) + 28
        draw.rounded_rectangle((W // 2 - tw // 2, ty, W // 2 + tw // 2, ty + 30), radius=8, fill=(24, 28, 48), outline=PINK, width=1)
        draw.text((W // 2, ty + 15), chip, fill=PINK, font=font(True, 12), anchor="mm")
        ty += 40

    draw.rounded_rectangle((MARGIN, H - MARGIN - 52, W - MARGIN, H - MARGIN), radius=12, fill=ACCENT)
    draw.text((W // 2, H - MARGIN - 26), "NEURAL JUNKIE · OPEN SOURCE", fill=(255, 255, 255), font=font(True, 15), anchor="mm")
    return canvas


header = compose_header()
ad = compose_ad()

for out, img in [(OUT_HEADER, header), (OUT_AD, ad), (GALLERY_HEADER, header), (GALLERY_AD, ad)]:
    out.parent.mkdir(parents=True, exist_ok=True)
    img.save(out, "PNG")
    print(f"Wrote {out}")
PY
