#!/usr/bin/env bash
# Compose neural-junkie-mcp-lora-ad-1200.png — LinkedIn article cover (1200×627).
# Usage: ./scripts/compose-mcp-lora-article.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/mcp-lora/creatives/neural-junkie-mcp-lora-ad-1200.png"
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


def draw_panel(draw, x0, y0, x1, y1, layer_label, title, bullets, accent, mono):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=(16, 22, 40), outline=accent, width=2)
    pad = 16
    inner_w = x1 - x0 - pad * 2
    draw.text((x0 + pad, y0 + pad), layer_label, fill=accent, font=font(True, 10), anchor="lt")
    draw.text((x0 + pad, y0 + pad + 18), title, fill=(255, 255, 255), font=font(True, 22), anchor="lt")
    bf = font(False, 11)
    y = y0 + pad + 50
    for line in bullets:
        for wrapped in wrap(draw, "• " + line, bf, inner_w)[:2]:
            draw.text((x0 + pad, y), wrapped, fill=MUTED, font=bf, anchor="lt")
            y += 15
        y += 4
    draw.text((x0 + pad, y1 - pad - 14), mono, fill=accent, font=font(True, 10, mono=True), anchor="lt")


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (14, 16, 32), (8, 22, 30))
draw = ImageDraw.Draw(canvas)

# Badge
badge = "OPEN SOURCE · NEURAL JUNKIE"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0 = MARGIN
by0 = 28
draw.rounded_rectangle(
    (bx0, by0, bx0 + bw, by0 + 28),
    radius=8,
    fill=(28, 32, 20),
    outline=TEAL,
    width=2,
)
draw.text((bx0 + bw // 2, by0 + 14), badge, fill=TEAL, font=bf, anchor="mm")

# Headline
head_y = 68
draw.text((MARGIN, head_y), "TWO LAYERS.", fill=(255, 255, 255), font=font(True, 40), anchor="lt")
draw.text((MARGIN, head_y + 46), "ONE SPECIALIST.", fill=GREEN, font=font(True, 42), anchor="lt")
fsub = font(False, 14)
sub = "MCP export shares knowledge. LoRA tunes behavior. Same repo expert — complementary, not interchangeable."
sub_y = head_y + 98
for line in wrap(draw, sub, fsub, CONTENT_W):
    draw.text((MARGIN, sub_y), line, fill=(200, 205, 220), font=fsub, anchor="lt")
    sub_y += 18

# Two columns
COL_GAP = 20
COL_W = (CONTENT_W - COL_GAP) // 2
left_x0 = MARGIN
right_x0 = MARGIN + COL_W + COL_GAP
panel_y = sub_y + 12
panel_h = 198

draw_panel(
    draw,
    left_x0,
    panel_y,
    left_x0 + COL_W,
    panel_y + panel_h,
    "CONTEXT LAYER",
    "MCP export",
    [
        "Snapshot indexed repo knowledge",
        "Architecture, files, prompts, system persona",
        "Portable JSON — any MCP client",
    ],
    TEAL,
    "/export-agent-mcp · import · resource server",
)

draw_panel(
    draw,
    right_x0,
    panel_y,
    right_x0 + COL_W,
    panel_y + panel_h,
    "MODEL LAYER",
    "LoRA adapters",
    [
        "Fine-tune small deltas on a shared base",
        "Import, pack presets, or train from chat",
        "Composed Ollama tags — local inference",
    ],
    AMBER,
    "nj-repo-myapp:14b · Train LoRA tab",
)

# Center connector
mid_x = MARGIN + CONTENT_W // 2
mid_y = panel_y + panel_h // 2
draw.rounded_rectangle(
    (mid_x - 28, mid_y - 28, mid_x + 28, mid_y + 28),
    radius=14,
    fill=(24, 30, 52),
    outline=PINK,
    width=2,
)
draw.text((mid_x, mid_y - 6), "repo", fill=PINK, font=font(True, 11), anchor="mm")
draw.text((mid_x, mid_y + 10), "expert", fill=PINK, font=font(True, 11), anchor="mm")

# Bottom strip — why both
strip_h = 58
strip_y = panel_y + panel_h + 14
draw.rounded_rectangle(
    (MARGIN, strip_y, MARGIN + CONTENT_W, strip_y + strip_h),
    radius=12,
    fill=(20, 28, 40),
    outline=GREEN,
    width=2,
)
draw.text(
    (MARGIN + 18, strip_y + 12),
    "Why both?",
    fill=GREEN,
    font=font(True, 13),
    anchor="lt",
)
draw.text(
    (MARGIN + 18, strip_y + 32),
    "Share context across tools (MCP)  ·  Bake session patterns into weights (LoRA)  ·  Prompts + indexing stay unchanged",
    fill=(210, 214, 224),
    font=font(False, 12),
    anchor="lt",
)

# Footer
draw.rounded_rectangle(
    (MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H),
    radius=12,
    fill=ACCENT,
)
draw.text(
    (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
    "github.com/camronwood/neural-junkie  ·  MCP_EXPORTS.md  ·  LORA_ADAPTERS.md",
    fill=(255, 255, 255),
    font=font(True, 13),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
