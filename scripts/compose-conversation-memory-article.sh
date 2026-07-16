#!/usr/bin/env bash
# Compose conversation-memory LinkedIn article cover (1200×627) + feed ad (1080×1080).
# Usage: ./scripts/compose-conversation-memory-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_HEADER="$ROOT/campaigns/conversation-memory/creatives/neural-junkie-conversation-memory-1200.png"
OUT_AD="$ROOT/campaigns/conversation-memory/creatives/neural-junkie-conversation-memory-ad-1080.png"
GALLERY_HEADER="$ROOT/docs/media/gallery/ads/neural-junkie-conversation-memory-1200.png"
GALLERY_AD="$ROOT/docs/media/gallery/ads/neural-junkie-conversation-memory-ad-1080.png"
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


def draw_layer_box(draw, x0, y0, x1, y1, label, sub, accent, highlight=False):
    fill = (20, 32, 28) if highlight else (16, 22, 40)
    width = 3 if highlight else 2
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=fill, outline=accent, width=width)
    pad = 10
    draw.text((x0 + pad, y0 + pad), label, fill=(255, 255, 255), font=font(True, 13), anchor="lt")
    draw.text((x0 + pad, y1 - pad - 12), sub, fill=accent, font=font(False, 10), anchor="lt")


def compose_header():
    W, H = 1200, 627
    MARGIN = 48
    CONTENT_W = W - 2 * MARGIN
    FOOTER_H = 46
    FOOTER_Y = H - MARGIN - FOOTER_H

    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (12, 14, 30), (6, 20, 28))
    draw = ImageDraw.Draw(canvas)

    badge = "OPEN SOURCE · NEURAL JUNKIE"
    bf = font(True, 11)
    bw = int(draw.textlength(badge, font=bf)) + 24
    bx0, by0 = MARGIN, 28
    draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 26), radius=8, fill=(28, 32, 20), outline=TEAL, width=2)
    draw.text((bx0 + bw // 2, by0 + 13), badge, fill=TEAL, font=bf, anchor="mm")

    head_y = 68
    draw.text((MARGIN, head_y), "AGENTS FORGET.", fill=(255, 255, 255), font=font(True, 34), anchor="lt")
    draw.text((MARGIN, head_y + 42), "WE RETRIEVE.", fill=AMBER, font=font(True, 38), anchor="lt")

    sub_y = head_y + 92
    draw.text((MARGIN, sub_y), "Tail history · Session summary · Conversation memory", fill=(200, 205, 220), font=font(False, 14), anchor="lt")

    layer_y = sub_y + 28
    layer_h = 78
    gap = 14
    layer_w = (CONTENT_W - 2 * gap) // 3
    layers = [
        ("TAIL", "Last 2–10 msgs", TEAL, False),
        ("SUMMARY", "Rolling 2KB", PINK, False),
        ("MEMORY", "Retrieve past context", GREEN, True),
    ]
    for i, (label, sub, accent, hi) in enumerate(layers):
        x0 = MARGIN + i * (layer_w + gap)
        x1 = x0 + layer_w
        draw_layer_box(draw, x0, layer_y, x1, layer_y + layer_h, label, sub, accent, hi)
        if i < 2:
            ax = x1 + gap // 2
            draw.text((ax, layer_y + layer_h // 2), "→", fill=MUTED, font=font(True, 16), anchor="mm")

    flow_y = layer_y + layer_h + 20
    flow_h = 100
    draw.rounded_rectangle((MARGIN, flow_y, MARGIN + CONTENT_W, flow_y + flow_h), radius=12, fill=(14, 20, 36), outline=MUTED, width=1)
    steps = ["User question", "Ollama embed", "memory.db", "top-k chunks", "prompt"]
    step_w = CONTENT_W // len(steps)
    for i, step in enumerate(steps):
        cx = MARGIN + step_w * i + step_w // 2
        cy = flow_y + flow_h // 2
        draw.text((cx, cy - 10), step, fill=GREEN if i == 4 else TEAL, font=font(True, 11), anchor="mm")
        if i < len(steps) - 1:
            nx = MARGIN + step_w * (i + 1)
            draw.text(((cx + nx - step_w // 2) // 2 + step_w // 4, cy + 14), "→", fill=MUTED, font=font(True, 14), anchor="mm")

    draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
    draw.text(
        (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
        "NEURAL JUNKIE · CONVERSATION-MEMORY-LINKEDIN.md",
        fill=(255, 255, 255),
        font=font(True, 13),
        anchor="mm",
    )
    return canvas


def compose_ad():
    W, H = 1080, 1080
    MARGIN = 56
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (12, 14, 30), (6, 24, 32))
    draw = ImageDraw.Draw(canvas)

    draw.text((W // 2, 72), "CONVERSATION MEMORY", fill=TEAL, font=font(True, 14), anchor="mm")
    headline = "What did we decide\nabout auth yesterday?"
    y = 120
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=(255, 255, 255), font=font(True, 40), anchor="mm")
        y += 48

    split_y = 260
    half = (W - 2 * MARGIN - 20) // 2
    left = (MARGIN, split_y, MARGIN + half, split_y + 420)
    right = (MARGIN + half + 20, split_y, W - MARGIN, split_y + 420)

    draw.rounded_rectangle(left, radius=14, fill=(18, 22, 38), outline=MUTED, width=2)
    draw.text((left[0] + 20, left[1] + 16), "TAIL HISTORY (faded)", fill=MUTED, font=font(True, 12), anchor="lt")
    faded = [
        "… message 38",
        "… message 39",
        "User: thanks",
        "Agent: you're welcome",
        "User: auth question?",
    ]
    ty = left[1] + 48
    for i, line in enumerate(faded):
        col = MUTED if i < 3 else (200, 205, 220)
        draw.text((left[0] + 20, ty), line, fill=col, font=font(False, 16), anchor="lt")
        ty += 30

    draw.rounded_rectangle(right, radius=14, fill=(14, 32, 28), outline=GREEN, width=3)
    draw.text((right[0] + 20, right[1] + 16), "RETRIEVED CHUNK", fill=GREEN, font=font(True, 12), anchor="lt")
    excerpt = (
        "[Camron] Agreed: JWT + refresh\n"
        "rotation in auth middleware.\n"
        "Tests in internal/auth/…"
    )
    ty = right[1] + 52
    for line in excerpt.split("\n"):
        draw.text((right[0] + 20, ty), line, fill=(220, 240, 230), font=font(False, 17), anchor="lt")
        ty += 32

    chips = ["Local embed", "SQLite", "Collab artifacts"]
    cx = W // 2
    ty = split_y + 440
    for chip in chips:
        tw = int(draw.textlength(chip, font=font(True, 12))) + 28
        draw.rounded_rectangle((cx - tw // 2, ty, cx + tw // 2, ty + 30), radius=8, fill=(24, 32, 48), outline=TEAL, width=1)
        draw.text((cx, ty + 15), chip, fill=TEAL, font=font(True, 12), anchor="mm")
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
