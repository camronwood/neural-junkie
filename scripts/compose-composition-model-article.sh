#!/usr/bin/env bash
# Compose neural-junkie-composition-model-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-composition-model-article.sh
#
# If a generated source PNG already exists in creatives/, resize/crop it to 1200×627.
# Otherwise draw the brand diagram poster.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/composition-model/creatives/neural-junkie-composition-model-1200.png"
GALLERY="$ROOT/docs/media/gallery/ads/neural-junkie-composition-model-1200.png"
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


def fit_cover(src: Path) -> Image.Image | None:
    if not src.exists():
        return None
    im = Image.open(src).convert("RGB")
    if im.size == (W, H):
        return im
    tw, th = W, H
    scale = max(tw / im.width, th / im.height)
    nw, nh = int(im.width * scale), int(im.height * scale)
    im = im.resize((nw, nh), Image.Resampling.LANCZOS)
    left = (nw - tw) // 2
    top = (nh - th) // 2
    return im.crop((left, top, left + tw, top + th))


existing = fit_cover(OUT)
if existing is not None and OUT.stat().st_size > 100_000:
    canvas = existing
else:
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (14, 10, 28), (8, 18, 32))
    draw = ImageDraw.Draw(canvas)

    badge = "OPEN SOURCE · NEURAL JUNKIE"
    bf = font(True, 11)
    bw = int(draw.textlength(badge, font=bf)) + 24
    bx0, by0 = MARGIN, 28
    draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 26), radius=8, fill=(28, 20, 32), outline=ACCENT, width=2)
    draw.text((bx0 + bw // 2, by0 + 13), badge, fill=ACCENT, font=bf, anchor="mm")

    head_y = 68
    draw.text((MARGIN, head_y), "BUNDLE. GRANT. TRACE.", fill=(255, 255, 255), font=font(True, 32), anchor="lt")
    draw.text((MARGIN, head_y + 40), "THE COMPOSITION MODEL.", fill=TEAL, font=font(True, 30), anchor="lt")

    sub_y = head_y + 88
    for line in wrap(
        draw,
        "Agents, tools, and runbooks as portable units — export, consented grant, provenance",
        font(False, 14),
        CONTENT_W,
    ):
        draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=font(False, 14), anchor="lt")
        sub_y += 18

    nodes = [
        ("Share Agent", "hydrate bundle", TEAL),
        ("Tool Wizard", "granted by name", GREEN),
        ("Runbooks", "export / import", BLUE),
        ("Provenance", "run → events", ACCENT),
    ]
    node_w = 220
    node_h = 78
    gap = (CONTENT_W - 4 * node_w) // 3
    ny = sub_y + 28
    for i, (title, sub, col) in enumerate(nodes):
        x0 = MARGIN + i * (node_w + gap)
        draw.rounded_rectangle((x0, ny, x0 + node_w, ny + node_h), radius=12, fill=(14, 20, 36), outline=col, width=2)
        draw.text((x0 + node_w // 2, ny + 28), title, fill=(255, 255, 255), font=font(True, 16), anchor="mm")
        draw.text((x0 + node_w // 2, ny + 52), sub, fill=col, font=font(False, 11, mono=True), anchor="mm")
        if i < len(nodes) - 1:
            ax0 = x0 + node_w + 6
            ax1 = x0 + node_w + gap - 6
            ay = ny + node_h // 2
            draw.line([(ax0, ay), (ax1, ay)], fill=MUTED, width=2)
            draw.polygon([(ax1, ay), (ax1 - 8, ay - 5), (ax1 - 8, ay + 5)], fill=MUTED)

    strip_y = ny + node_h + 24
    strip_h = 56
    draw.rounded_rectangle((MARGIN, strip_y, MARGIN + CONTENT_W, strip_y + strip_h), radius=10, fill=(12, 18, 32), outline=TEAL, width=1)
    items = ["export", "hydrate", "grant", "SSRF-safe", "provenance", "progress"]
    item_w = (CONTENT_W - 24 - 5 * 8) // 6
    for i, item in enumerate(items):
        ix0 = MARGIN + 12 + i * (item_w + 8)
        draw.rounded_rectangle((ix0, strip_y + 10, ix0 + item_w, strip_y + strip_h - 10), radius=6, fill=(16, 22, 40), outline=MUTED, width=1)
        draw.text((ix0 + item_w // 2, strip_y + strip_h // 2), item, fill=(220, 225, 235), font=font(False, 11), anchor="mm")

    draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=ACCENT)
    draw.text(
        (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
        "NEURAL JUNKIE · THE COMPOSITION MODEL",
        fill=(255, 255, 255),
        font=font(True, 13),
        anchor="mm",
    )

OUT.parent.mkdir(parents=True, exist_ok=True)
if isinstance(canvas, Image.Image) and canvas.size != (W, H):
    canvas = canvas.resize((W, H), Image.Resampling.LANCZOS)
canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
GALLERY.parent.mkdir(parents=True, exist_ok=True)
canvas.save(GALLERY, "PNG")
print(f"Wrote {GALLERY}")
PY
