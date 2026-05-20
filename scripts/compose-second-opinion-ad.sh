#!/usr/bin/env bash
# Compose neural-junkie-nondev-second-opinion-ad-1080.png — second LLM check (non-dev).
# Usage: ./scripts/compose-second-opinion-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-nondev-second-opinion-ad-1080.png"
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


def draw_cta(draw, y_btn: int):
    btn_h = 56
    draw.rounded_rectangle(
        (MARGIN, y_btn, W - MARGIN, y_btn + btn_h),
        radius=14,
        fill=(233, 69, 96),
    )
    draw.text(
        (W // 2, y_btn + btn_h // 2),
        "Download — macOS · Windows · Linux",
        fill=(255, 255, 255),
        font=font(True, 22),
        anchor="mm",
    )
    draw.text(
        (W // 2, y_btn + btn_h + 28),
        "github.com/camronwood/neural-junkie/releases",
        fill=(136, 140, 168),
        font=font(False, 15),
        anchor="mm",
    )


def draw_tab_chaos(draw, box, label: str):
    """Two browser tabs — the old habit."""
    x0, y0, x1, y1 = box
    draw.rounded_rectangle(box, radius=14, fill=(28, 32, 52), outline=(90, 100, 130), width=2)
    draw.text((x0 + 16, y0 + 12), label, fill=(120, 128, 150), font=font(True, 11), anchor="lt")
    tw, th, gap = 118, 28, 8
    tx = x0 + 16
    ty = y0 + 36
    for i, name in enumerate(["ChatGPT", "Claude"]):
        txi = tx + i * (tw + gap)
        col = (233, 69, 96) if i == 0 else (199, 125, 255)
        draw.rounded_rectangle((txi, ty, txi + tw, ty + th), radius=6, fill=(18, 22, 40), outline=col, width=2)
        draw.text((txi + tw // 2, ty + th // 2), name, fill=(168, 176, 184), font=font(False, 11), anchor="mm")
        cx = txi + tw - 14
        cy = ty + th // 2
        draw.line([(cx - 4, cy - 4), (cx + 4, cy + 4)], fill=col, width=2)
        draw.line([(cx + 4, cy - 4), (cx - 4, cy + 4)], fill=col, width=2)
    draw.text(
        (x0 + 16, y1 - 36),
        "Copy · paste · re-explain yourself",
        fill=(233, 69, 96),
        font=font(True, 13),
        anchor="lt",
    )


def draw_thread_mock(draw, box):
    """One app — ask one expert, second reviews in the same thread."""
    x0, y0, x1, y1 = box
    draw.rounded_rectangle(box, radius=14, fill=(14, 18, 36), outline=(83, 52, 131), width=2)
    bar_h = 32
    draw.rectangle((x0, y0, x1, y0 + bar_h), fill=(28, 36, 58))
    draw.text(
        (x0 + 14, y0 + bar_h // 2),
        "DM · Trip + budget planning",
        fill=(220, 224, 232),
        font=font(False, 12),
        anchor="lm",
    )
    badge = "same thread"
    bf = font(True, 10)
    bw = int(draw.textlength(badge, font=bf)) + 14
    bx1 = x1 - 12
    draw.rounded_rectangle(
        (bx1 - bw, y0 + 6, bx1, y0 + bar_h - 6),
        radius=6,
        fill=(24, 34, 58),
        outline=(72, 199, 142),
    )
    draw.text((bx1 - bw // 2, y0 + bar_h // 2), badge, fill=(72, 199, 142), font=bf, anchor="mm")

    pad = 14
    chat_l = x0 + pad
    chat_r = x1 - pad
    chat_w = chat_r - chat_l
    y = y0 + bar_h + pad

    bubbles = [
        ((52, 38, 78), "You", "Plan a 10-day Italy trip under $4k — flights included?", 0, 44),
        ((38, 52, 88), "Trip planner", "Fly into Rome, train north — I'll draft a day-by-day…", 0, 50),
        ((78, 52, 38), "You", "@Budget buddy — does this math hold up?", 0, 42),
        ((38, 72, 58), "Budget buddy", "Rome-first saves ~$400 vs Milan start. Watch shoulder-season fares.", 0, 52),
    ]
    colors = {
        "You": (168, 176, 184),
        "Trip planner": (199, 125, 255),
        "Budget buddy": (72, 199, 142),
    }
    for fill, who, line, indent, bh in bubbles:
        x = chat_l + indent
        w = chat_w - indent
        draw.rounded_rectangle((x, y, x + w, y + bh), radius=8, fill=fill, outline=(48, 58, 88))
        draw.text((x + 10, y + 6), who, fill=colors.get(who, (255, 255, 255)), font=font(True, 11))
        draw.text((x + 10, y + 22), line, fill=(240, 242, 248), font=font(False, 12))
        y += bh + 10

    draw.text(
        (chat_l, y + 4),
        "Reply in thread → @mention another expert for a second opinion",
        fill=(136, 140, 168),
        font=font(False, 11),
        anchor="lt",
    )


canvas = Image.new("RGB", (W, H), (10, 12, 28))
gradient_bg(canvas, (8, 10, 26), (20, 16, 42))
draw = ImageDraw.Draw(canvas)

y = 48
draw.text((W // 2, y), "NEURAL JUNKIE", fill=(199, 125, 255), font=font(True, 28), anchor="mm")
y += 40
draw.text((W // 2, y), "TWO AIs", fill=(120, 128, 150), font=font(True, 24), anchor="mm")
y += 38
draw.text((W // 2, y), "ONE ROOM", fill=(255, 255, 255), font=font(True, 62), anchor="mm")
y += 56
sub = "Ask one expert. Have another check the answer — no new browser tab."
fsub = font(False, 18)
for line in wrap(draw, sub, fsub, W - 2 * MARGIN):
    draw.text((W // 2, y), line, fill=(200, 205, 220), font=fsub, anchor="mm")
    y += 24

y += 20
col_w = (W - 2 * MARGIN - 20) // 2
left = (MARGIN, y, MARGIN + col_w, y + 200)
right = (MARGIN + col_w + 20, y, W - MARGIN, y + 200)
draw_tab_chaos(draw, left, "THE OLD WAY")
draw_thread_mock(draw, right)

# Arrow between columns
ax = MARGIN + col_w + 10
mid_y = y + 100
draw.text((ax, mid_y), "→", fill=(233, 69, 96), font=font(True, 28), anchor="mm")

y += 220
steps = [
    ("1", "Pick your experts", "Writing coach, trip planner, budget buddy — your names"),
    ("2", "Get an answer", "Same DM or channel you already use"),
    ("3", "Ask for a second look", "Reply in the thread and @mention another expert"),
]
row_h = 72
for num, title, sub in steps:
    cy = y + row_h // 2
    draw.ellipse(
        (MARGIN, y + 12, MARGIN + 40, y + 52),
        fill=(233, 69, 96),
    )
    draw.text((MARGIN + 20, cy), num, fill=(255, 255, 255), font=font(True, 16), anchor="mm")
    draw.text((MARGIN + 56, y + 14), title, fill=(255, 255, 255), font=font(True, 18), anchor="lt")
    draw.text((MARGIN + 56, y + 38), sub, fill=(168, 176, 184), font=font(False, 14), anchor="lt")
    y += row_h

draw.text(
    (W // 2, y + 8),
    "Not fact-checking magic — a second perspective before you act.",
    fill=(136, 140, 168),
    font=font(False, 13),
    anchor="mm",
)

draw_cta(draw, 868)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
