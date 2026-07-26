#!/usr/bin/env bash
# Compose context-stack LinkedIn article cover (1200×627) + feed ad (1080×1080).
# Usage: ./scripts/compose-context-stack-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_HEADER="$ROOT/campaigns/context-stack/creatives/neural-junkie-context-stack-1200.png"
OUT_AD="$ROOT/campaigns/context-stack/creatives/neural-junkie-context-stack-ad-1080.png"
GALLERY_HEADER="$ROOT/docs/media/gallery/ads/neural-junkie-context-stack-1200.png"
GALLERY_AD="$ROOT/docs/media/gallery/ads/neural-junkie-context-stack-ad-1080.png"
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
BLUE = (96, 165, 250)


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
    w, h = canvas.size
    for y in range(h):
        t = y / max(h - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (w, y)], fill=(r, g, b))


def compose_header():
    W, H = 1200, 627
    MARGIN = 48
    CONTENT_W = W - 2 * MARGIN
    FOOTER_H = 46
    FOOTER_Y = H - MARGIN - FOOTER_H

    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (10, 16, 32), (8, 22, 28))
    draw = ImageDraw.Draw(canvas)

    badge = "OPEN SOURCE · NEURAL JUNKIE"
    bf = font(True, 11)
    bw = int(draw.textlength(badge, font=bf)) + 24
    bx0, by0 = MARGIN, 26
    draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 26), radius=8, fill=(20, 28, 36), outline=TEAL, width=2)
    draw.text((bx0 + bw // 2, by0 + 13), badge, fill=TEAL, font=bf, anchor="mm")

    head_y = 64
    draw.text((MARGIN, head_y), "BUILD. USE. SHARE.", fill=(255, 255, 255), font=font(True, 34), anchor="lt")
    draw.text((MARGIN, head_y + 40), "SCOPED CONTEXT — NOT A HIVE MIND.", fill=AMBER, font=font(True, 28), anchor="lt")

    sub_y = head_y + 86
    draw.text(
        (MARGIN, sub_y),
        "Conversation Context Stack · turn-by-turn prompts · explicit sharing",
        fill=(200, 205, 220),
        font=font(False, 14),
        anchor="lt",
    )

    # Six stack stages
    stages = [
        ("1 MODE", "chat / code", TEAL),
        ("2 INTENT", "casual→task", BLUE),
        ("3 MEMORY", "summary+tail", PINK),
        ("4 GROUND", "scope tiers", GREEN),
        ("5 PERSONA", "DM / room", AMBER),
        ("6 BUDGET", "caps + CCR", ACCENT),
    ]
    layer_y = sub_y + 28
    layer_h = 72
    gap = 10
    layer_w = (CONTENT_W - 5 * gap) // 6
    for i, (label, sub, accent) in enumerate(stages):
        x0 = MARGIN + i * (layer_w + gap)
        x1 = x0 + layer_w
        draw.rounded_rectangle((x0, layer_y, x1, layer_y + layer_h), radius=10, fill=(14, 20, 36), outline=accent, width=2)
        draw.text((x0 + layer_w // 2, layer_y + 22), label, fill=(255, 255, 255), font=font(True, 12), anchor="mm")
        draw.text((x0 + layer_w // 2, layer_y + 48), sub, fill=accent, font=font(False, 10), anchor="mm")
        if i < 5:
            ax = x1 + gap // 2
            draw.text((ax, layer_y + layer_h // 2), "→", fill=MUTED, font=font(True, 14), anchor="mm")

    # Share strip
    share_y = layer_y + layer_h + 18
    share_h = 88
    draw.rounded_rectangle(
        (MARGIN, share_y, MARGIN + CONTENT_W, share_y + share_h),
        radius=12,
        fill=(14, 20, 36),
        outline=MUTED,
        width=1,
    )
    draw.text((MARGIN + 16, share_y + 12), "SHARE PATHS", fill=MUTED, font=font(True, 10), anchor="lt")
    paths = [
        ("Channel", "same transcript", TEAL),
        ("Delegate", "DELEGATE_RESULTS", GREEN),
        ("Collab", "plan + tasks", PINK),
        ("Learnings", "user-confirmed", AMBER),
        ("Memory", "retrieve past", BLUE),
    ]
    path_w = (CONTENT_W - 32 - 4 * 10) // 5
    for i, (title, sub, col) in enumerate(paths):
        px0 = MARGIN + 16 + i * (path_w + 10)
        py0 = share_y + 32
        draw.rounded_rectangle((px0, py0, px0 + path_w, py0 + 44), radius=8, fill=(16, 24, 40), outline=col, width=1)
        draw.text((px0 + path_w // 2, py0 + 14), title, fill=(255, 255, 255), font=font(True, 12), anchor="mm")
        draw.text((px0 + path_w // 2, py0 + 32), sub, fill=col, font=font(False, 9), anchor="mm")

    draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
    draw.text(
        (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
        "NEURAL JUNKIE · CONTEXT-STACK-LINKEDIN.md",
        fill=(255, 255, 255),
        font=font(True, 13),
        anchor="mm",
    )
    return canvas


def compose_ad():
    W, H = 1080, 1080
    MARGIN = 56
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (10, 16, 32), (6, 24, 30))
    draw = ImageDraw.Draw(canvas)

    draw.text((W // 2, 64), "CONVERSATION CONTEXT STACK", fill=TEAL, font=font(True, 14), anchor="mm")

    y = 110
    for line in ["Build the right prompt.", "Use only what the turn needs.", "Share on purpose."]:
        draw.text((W // 2, y), line, fill=(255, 255, 255), font=font(True, 32), anchor="mm")
        y += 44

    # Vertical stack
    stages = [
        ("MODE", "chat · code · collab", TEAL),
        ("INTENT", "closure → task", BLUE),
        ("MEMORY", "summary + history", PINK),
        ("GROUNDING", "none → full scope", GREEN),
        ("PERSONA", "DM · channel · collab", AMBER),
        ("BUDGET", "caps + CCR refs", ACCENT),
    ]
    box_h = 64
    gap = 10
    stack_top = 280
    box_w = W - 2 * MARGIN
    for i, (label, sub, col) in enumerate(stages):
        y0 = stack_top + i * (box_h + gap)
        draw.rounded_rectangle((MARGIN, y0, MARGIN + box_w, y0 + box_h), radius=12, fill=(14, 20, 36), outline=col, width=2)
        draw.text((MARGIN + 28, y0 + box_h // 2), f"{i + 1}", fill=col, font=font(True, 22), anchor="lm")
        draw.text((MARGIN + 70, y0 + 18), label, fill=(255, 255, 255), font=font(True, 18), anchor="lt")
        draw.text((MARGIN + 70, y0 + 40), sub, fill=col, font=font(False, 14), anchor="lt")
        if i < len(stages) - 1:
            draw.text((W // 2, y0 + box_h + gap // 2), "↓", fill=MUTED, font=font(True, 14), anchor="mm")

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
