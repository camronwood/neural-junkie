#!/usr/bin/env bash
# Compose neural-junkie-two-tier-lora-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-two-tier-lora-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/two-tier-lora/creatives/neural-junkie-two-tier-lora-1200.png"
GALLERY="$ROOT/docs/media/gallery/ads/neural-junkie-two-tier-lora-ad-1200.png"
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


def draw_tier_box(draw, x0, y0, x1, y1, title, accent, bullets, mono_tags):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=(18, 24, 44), outline=accent, width=2)
    pad = 16
    draw.text((x0 + pad, y0 + pad), title, fill=accent, font=font(True, 13), anchor="lt")
    y = y0 + pad + 22
    bf = font(False, 11)
    for line in bullets:
        for wl in wrap(draw, line, bf, x1 - x0 - pad * 2)[:2]:
            draw.text((x0 + pad, y), f"• {wl}", fill=MUTED, font=bf, anchor="lt")
            y += 14
    y = y1 - pad - 14 * len(mono_tags)
    for tag in mono_tags:
        draw.text((x0 + pad, y), tag, fill=accent, font=font(True, 10, mono=True), anchor="lt")
        y += 14


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (14, 16, 32), (8, 22, 30))
draw = ImageDraw.Draw(canvas)

badge = "OPEN SOURCE · NEURAL JUNKIE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0, by0 = MARGIN, 32
draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 28), radius=8, fill=(28, 32, 20), outline=TEAL, width=2)
draw.text((bx0 + bw // 2, by0 + 14), badge, fill=TEAL, font=bf, anchor="mm")

head_y = 76
draw.text((MARGIN, head_y), "INFERENCE ON QWEN.", fill=(255, 255, 255), font=font(True, 34), anchor="lt")
draw.text((MARGIN, head_y + 42), "LORA ON LLAMA.", fill=GREEN, font=font(True, 38), anchor="lt")
sub_y = head_y + 92
for line in wrap(draw, "Two tiers. One hub. Composable specialists.", font(False, 15), CONTENT_W):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=font(False, 15), anchor="lt")
    sub_y += 20

tier_y = sub_y + 10
tier_h = 168
half_w = (CONTENT_W - 20) // 2
left_x0 = MARGIN
left_x1 = left_x0 + half_w
right_x0 = left_x1 + 20
right_x1 = MARGIN + CONTENT_W

draw_tier_box(
    draw, left_x0, tier_y, left_x1, tier_y + tier_h,
    "INFERENCE TIER",
    TEAL,
    ["Default specialist chat & tools", "Software development pack"],
    ["qwen2.5-coder:14b"],
)
draw_tier_box(
    draw, right_x0, tier_y, right_x1, tier_y + tier_h,
    "LORA TIER",
    GREEN,
    ["Train · compose · bootstrap presets", "Ollama safetensors ADAPTER"],
    ["llama3.1:8b", "llama3.2:3b · mistral:7b", "→ nj-security:14b"],
)

arrow_y = tier_y + tier_h // 2
draw.text(((left_x1 + right_x0) // 2, arrow_y), "⇄", fill=AMBER, font=font(True, 28), anchor="mm")
draw.text((MARGIN + CONTENT_W // 2, tier_y + tier_h + 12), "Assign composed tags when you want domain-tuned weights", fill=MUTED, font=font(False, 12), anchor="mm")

draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "NEURAL JUNKIE · TWO-TIER-LORA-LINKEDIN.md · LORA_ADAPTERS.md",
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
