#!/usr/bin/env bash
# Compose neural-junkie-byom-ad-1080.png — Bring Your Own Model / agent IDE.
# Usage: ./scripts/compose-byom-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/campaigns/byom/creatives/neural-junkie-byom-ad-1080.png"
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
MARGIN = 48

TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
PINK = (199, 125, 255)
ACCENT = (233, 69, 96)
GREEN = (72, 199, 142)
MUTED = (168, 176, 184)
DIM = (136, 140, 168)
IDE_BG = (14, 18, 36)
SLOT_BG = (22, 28, 52)


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


def draw_model_chip(draw, cx, cy, label: str, accent):
    f = font(True, 13)
    tw = int(draw.textlength(label, font=f)) + 24
    ph = 34
    x0, y0 = cx - tw // 2, cy - ph // 2
    draw.rounded_rectangle(
        (x0, y0, x0 + tw, y0 + ph),
        radius=8,
        fill=(18, 24, 44),
        outline=accent,
        width=2,
    )
    draw.text((cx, cy), label, fill=(255, 255, 255), font=f, anchor="mm")


def draw_ide_mock(draw, box):
    x0, y0, x1, y1 = box
    draw.rounded_rectangle(box, radius=14, fill=IDE_BG, outline=(83, 52, 131), width=2)
    bar_h = 36
    draw.rectangle((x0, y0, x1, y0 + bar_h), fill=(28, 36, 58))
    draw.text((x0 + 14, y0 + bar_h // 2), "IDE layout · my-repo", fill=(220, 224, 232), font=font(False, 12), anchor="lm")
    draw.text((x1 - 14, y0 + bar_h // 2), "⇧⌘M Model library", fill=TEAL, font=font(True, 10), anchor="rm")

    tree_w = 88
    inner_y = y0 + bar_h
    draw.rectangle((x0, inner_y, x0 + tree_w, y1), fill=(18, 22, 40))
    draw.line([(x0 + tree_w, inner_y), (x0 + tree_w, y1)], fill=(48, 58, 88), width=1)
    ty = inner_y + 14
    for path in ["src/", "  main.go", "pkg/", "README"]:
        draw.text((x0 + 10, ty), path, fill=MUTED, font=font(False, 10, mono=True), anchor="lt")
        ty += 18

    ed_x0 = x0 + tree_w + 1
    ed_x1 = x1 - 4
    draw.rectangle((ed_x0, inner_y, ed_x1, y1 - 120), fill=(12, 16, 30))
    draw.text((ed_x0 + 12, inner_y + 12), "handler.go", fill=PINK, font=font(True, 11), anchor="lt")
    code_lines = [
        "func Serve(w http.ResponseWriter, r *http.Request) {",
        "  // @GoExpert — review this",
        "  auth := middleware.FromContext(r)",
    ]
    cy = inner_y + 32
    for line in code_lines:
        draw.text((ed_x0 + 12, cy), line, fill=(180, 190, 210), font=font(False, 10, mono=True), anchor="lt")
        cy += 16

    chat_y0 = y1 - 116
    draw.rectangle((ed_x0, chat_y0, ed_x1, y1 - 8), fill=(20, 26, 48), outline=(48, 58, 88))
    draw.text((ed_x0 + 10, chat_y0 + 8), "#general", fill=MUTED, font=font(True, 10), anchor="lt")
    draw.rounded_rectangle(
        (ed_x0 + 10, chat_y0 + 26, ed_x1 - 10, chat_y0 + 52),
        radius=6,
        fill=(38, 52, 88),
    )
    draw.text(
        (ed_x0 + 16, chat_y0 + 38),
        "@SecurityReviewer on this diff?",
        fill=(240, 242, 248),
        font=font(False, 11),
        anchor="lm",
    )
    draw.text(
        (ed_x0 + 10, chat_y0 + 58),
        "Pending changes · you approve",
        fill=GREEN,
        font=font(True, 10),
        anchor="lt",
    )


canvas = Image.new("RGB", (W, H), (10, 14, 28))
gradient_bg(canvas, (16, 14, 36), (10, 22, 32))
draw = ImageDraw.Draw(canvas)

badge = "AGENT IDE · LOCAL-FIRST"
bf = font(True, 11)
bw = int(draw.textlength(badge, font=bf)) + 24
bx0 = (W - bw) // 2
by0 = 40
draw.rounded_rectangle(
    (bx0, by0, bx0 + bw, by0 + 30),
    radius=8,
    fill=(20, 28, 48),
    outline=TEAL,
    width=2,
)
draw.text((W // 2, by0 + 15), badge, fill=TEAL, font=bf, anchor="mm")

y = 86
draw.text((W // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
y += 38
draw.text((W // 2, y), "BRING YOUR", fill=(255, 255, 255), font=font(True, 44), anchor="mm")
y += 48
draw.text((W // 2, y), "OWN MODEL", fill=TEAL, font=font(True, 48), anchor="mm")
y += 44
sub = "An agent IDE — files, specialists, approvals. You pick the brain."
fsub = font(False, 16)
for line in wrap(draw, sub, fsub, W - 2 * MARGIN):
    draw.text((W // 2, y), line, fill=(200, 205, 220), font=fsub, anchor="mm")
    y += 20

y += 16
split_y = y
panel_h = 420
left_box = (MARGIN, split_y, MARGIN + 340, split_y + panel_h)
right_box = (MARGIN + 356, split_y, W - MARGIN, split_y + panel_h)

# Left — model slot
x0, y0, x1, y1 = left_box
draw.rounded_rectangle(left_box, radius=14, fill=SLOT_BG, outline=AMBER, width=2)
draw.text((x0 + 20, y0 + 18), "YOUR MODEL", fill=AMBER, font=font(True, 14), anchor="lt")
draw.text((x0 + 20, y0 + 40), "Plug in · swap · per agent", fill=DIM, font=font(False, 12), anchor="lt")

slot_inner = (x0 + 16, y0 + 68, x1 - 16, y1 - 16)
draw.rounded_rectangle(slot_inner, radius=10, fill=(12, 16, 32), outline=(60, 70, 100), width=1)
ix0, iy0, ix1, iy1 = slot_inner
cx_slot = (ix0 + ix1) // 2
draw.text((cx_slot, iy0 + 36), "?", fill=(60, 68, 90), font=font(True, 72), anchor="mm")
draw.text((cx_slot, iy0 + 88), "You choose", fill=MUTED, font=font(True, 16), anchor="mm")

chips = [
    ("Ollama", GREEN),
    ("Claude", PINK),
    ("GPT", TEAL),
    ("Hugging Face", AMBER),
    ("OpenAI-compat", (180, 160, 255)),
]
chip_y = iy0 + 130
for i, (lab, col) in enumerate(chips):
    row, col_idx = i // 2, i % 2
    cx = ix0 + 90 + col_idx * 118
    cy = chip_y + row * 44
    draw_model_chip(draw, cx, cy, lab, col)

draw.text(
    (cx_slot, iy1 - 28),
    "Different model per specialist",
    fill=DIM,
    font=font(False, 11),
    anchor="mm",
)

# Arrow
ax = left_box[2] + 8
draw.text((ax + 14, split_y + panel_h // 2), "→", fill=ACCENT, font=font(True, 32), anchor="mm")

# Right — IDE mock
draw_ide_mock(draw, right_box)

y = split_y + panel_h + 20
bullets = [
    "Model library ⇧⌘M — browse Ollama & Hugging Face",
    "Per-agent provider routing — local + cloud in one app",
    "File proposals — nothing writes without your OK",
]
for b in bullets:
    draw.text((MARGIN + 8, y), "✓", fill=GREEN, font=font(True, 14), anchor="lt")
    draw.text((MARGIN + 28, y), b, fill=(210, 214, 224), font=font(False, 14), anchor="lt")
    y += 24

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
    "We're the workbench. You bring the brain.",
    fill=DIM,
    font=font(False, 14),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
