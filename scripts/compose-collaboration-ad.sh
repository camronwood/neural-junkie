#!/usr/bin/env bash
# Compose neural-junkie-collaboration-ad-1080.png — agent orchestration /collaborate (no screenshots).
# Usage: ./scripts/compose-collaboration-ad.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/assets/neural-junkie-collaboration-ad-1080.png"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$OUT" <<'PY'
import random
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

OUT = Path(sys.argv[1])

W, H = 1080, 1080
BG = (13, 13, 26)
PANEL = (18, 24, 44)
PANEL_EDGE = (48, 58, 88)
BOX_FILL = (24, 34, 58)
BOX_EDGE = (83, 52, 131)
ACCENT = (233, 69, 96)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
PINK = (199, 125, 255)
GREEN = (72, 199, 142)
AMBER = (255, 193, 94)
NET_LINE = (42, 38, 72)
NET_NODE = (110, 70, 150)
MOCK_BG = (12, 16, 32)
MOCK_BAR = (28, 36, 58)
BUBBLE_A = (32, 44, 72)
BUBBLE_B = (38, 52, 88)
BUBBLE_U = (52, 38, 78)

# Layout grid (explicit y positions — no stacked offsets)
PANEL_BOX = (48, 186, 1032, 1012)
INSET = 28
INNER_L = PANEL_BOX[0] + INSET
INNER_R = PANEL_BOX[2] - INSET
INNER_W = INNER_R - INNER_L
CX = W // 2


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


def draw_centered_lines(draw, y_start, lines, fnt, fill, line_gap: int):
    y = y_start
    for line in lines:
        draw.text((CX, y), line, fill=fill, font=fnt, anchor="mm")
        y += line_gap
    return y


def draw_network(draw: ImageDraw.ImageDraw, rng: random.Random):
    nodes = [(rng.randint(30, W - 30), rng.randint(20, 168)) for _ in range(12)]
    for i, (x1, y1) in enumerate(nodes):
        for x2, y2 in nodes[i + 1 :]:
            if (x1 - x2) ** 2 + (y1 - y2) ** 2 < 130**2:
                draw.line([(x1, y1), (x2, y2)], fill=NET_LINE, width=1)
    for x, y in nodes:
        r = 3
        draw.ellipse((x - r, y - r, x + r, y + r), fill=NET_NODE)


def draw_brand(draw: ImageDraw.ImageDraw):
    draw.text((CX, 38), "FOR DEVELOPERS", fill=MUTED, font=font(False, 12), anchor="mm")
    title = "NEURAL JUNKIE"
    ft = font(True, 48)
    tw = draw.textlength(title, font=ft)
    tx = (W - tw) // 2
    draw.text((tx, 58), title, fill=TEXT, font=ft, anchor="lt")
    draw.rectangle((tx, 114, tx + tw, 119), fill=ACCENT)
    pill = "Agent orchestration"
    fp = font(True, 14)
    pw = int(draw.textlength(pill, font=fp)) + 26
    px = (W - pw) // 2
    draw.rounded_rectangle((px, 130, px + pw, 156), radius=10, fill=BOX_FILL, outline=BOX_EDGE)
    draw.text((CX, 143), pill, fill=PINK, font=fp, anchor="mm")


def draw_workflow(draw: ImageDraw.ImageDraw, y_top: int, inner_l: int, inner_r: int):
    steps = ["Agents discuss", "Shared plan", "You approve", "They execute"]
    n = len(steps)
    pill_h = 38
    arrow_w = 24
    gap = 10
    usable = inner_r - inner_l
    pill_w = (usable - (n - 1) * (arrow_w + gap)) // n
    cy = y_top + pill_h // 2
    x = inner_l
    for i, label in enumerate(steps):
        x1, x2 = x, x + pill_w
        draw.rounded_rectangle(
            (x1, y_top, x2, y_top + pill_h), radius=9, fill=BOX_FILL, outline=BOX_EDGE, width=1
        )
        fs = 13 if len(label) > 11 else 14
        draw.text(((x1 + x2) // 2, cy), label, fill=TEXT, font=font(True, fs), anchor="mm")
        x = x2
        if i < n - 1:
            ax0, ax1 = x + 4, x + arrow_w - 4
            draw.line([(ax0, cy), (ax1, cy)], fill=ACCENT, width=2)
            draw.polygon([(ax1, cy), (ax1 - 6, cy - 4), (ax1 - 6, cy + 4)], fill=ACCENT)
            x += arrow_w + gap
    return y_top + pill_h


def draw_collab_mock(draw: ImageDraw.ImageDraw, box):
    x0, y0, x1, y1 = box
    draw.rounded_rectangle(box, radius=12, fill=MOCK_BG, outline=PANEL_EDGE, width=1)

    bar_h = 34
    draw.rectangle((x0, y0, x1, y0 + bar_h), fill=MOCK_BAR)
    dot_y = y0 + bar_h // 2
    for i, col in enumerate((ACCENT, AMBER, GREEN)):
        dx = x0 + 16 + i * 18
        draw.ellipse((dx - 5, dot_y - 5, dx + 5, dot_y + 5), fill=col)

    draw.text(
        (x0 + 70, dot_y),
        "collab-7f3a…  ·  planning",
        fill=TEXT,
        font=font(False, 13),
        anchor="lm",
    )
    badge = "agents ↔ agents · bounded turns"
    bf = font(True, 10)
    bw = int(draw.textlength(badge, font=bf)) + 16
    bx1 = x1 - 14
    bx0 = bx1 - bw
    draw.rounded_rectangle((bx0, y0 + 7, bx1, y0 + bar_h - 7), radius=6, fill=(20, 28, 48), outline=BOX_EDGE)
    draw.text(((bx0 + bx1) // 2, dot_y), badge, fill=PINK, font=bf, anchor="mm")

    pad = 12
    side_w = 168
    content_top = y0 + bar_h + pad
    content_bot = y1 - pad
    chat_r = x1 - pad - side_w - 10
    chat_l = x0 + pad

    # Side panel — full height of content area
    px0 = x1 - pad - side_w
    draw.rounded_rectangle(
        (px0, content_top, x1 - pad, content_bot), radius=8, fill=BOX_FILL, outline=BOX_EDGE
    )
    pw = x1 - pad - px0
    pcx = px0 + pw // 2
    draw.text((px0 + 12, content_top + 12), "Plan v3", fill=TEXT, font=font(True, 13), anchor="lt")
    draw.text((px0 + 12, content_top + 30), "phase: reviewing", fill=AMBER, font=font(False, 11), anchor="lt")

    tasks = ["1  @RustExpert  impl", "2  @SecurityExpert  review", "3  @DevOpsPro  package"]
    ty = content_top + 52
    row_h = 24
    row_gap = 6
    for t in tasks:
        draw.rounded_rectangle(
            (px0 + 10, ty, x1 - pad - 10, ty + row_h), radius=4, fill=MOCK_BG, outline=BOX_EDGE
        )
        draw.text((px0 + 16, ty + row_h // 2), t, fill=MUTED, font=font(False, 10), anchor="lm")
        ty += row_h + row_gap

    btn_h = 28
    btn_y1 = content_bot - 8
    btn_y0 = btn_y1 - btn_h
    draw.rounded_rectangle((px0 + 10, btn_y0, x1 - pad - 10, btn_y1), radius=6, fill=ACCENT)
    draw.text((pcx, (btn_y0 + btn_y1) // 2), "/approve-plan", fill=TEXT, font=font(True, 11), anchor="mm")

    # Chat bubbles — evenly spaced in remaining height
    messages = [
        (BUBBLE_U, "You", "/collaborate @RustExpert @SecurityExpert — AES file CLI", 0, 46),
        (BUBBLE_A, "@RustExpert", "Propose argon2id + chunked encrypt; CLI with clap", 0, 50),
        (BUBBLE_B, "@SecurityExpert", "Add KDF limits + threat model section", 12, 46),
        (BUBBLE_A, "@RustExpert", "Agree — I'll own implementation tasks", 0, 42),
    ]
    n = len(messages)
    avail_h = content_bot - content_top
    total_bubble_h = sum(m[4] for m in messages)
    gap = max(8, (avail_h - total_bubble_h) // (n + 1))
    chat_y = content_top + gap
    chat_w = chat_r - chat_l

    for fill, who, line, indent, bh in messages:
        x = chat_l + indent
        w = chat_w - indent
        draw.rounded_rectangle((x, chat_y, x + w, chat_y + bh), radius=8, fill=fill, outline=BOX_EDGE)
        draw.text((x + 10, chat_y + 7), who, fill=PINK if who.startswith("@") else MUTED, font=font(True, 11))
        draw.text((x + 10, chat_y + 24), line, fill=TEXT, font=font(False, 12))
        chat_y += bh + gap


def draw_callouts(draw: ImageDraw.ImageDraw, y_start: int):
    lines = [
        "Agents talk to each other (turn limits — not endless loops)",
        "One shared plan · /approve-plan before execution",
        "Tasks fan out · file edits still need your OK",
    ]
    f = font(False, 14)
    y = y_start
    row_h = 26
    for text in lines:
        cy = y + row_h // 2
        tw = draw.textlength(text, font=f)
        bullet_x = CX - tw // 2 - 14
        draw.ellipse((bullet_x - 4, cy - 4, bullet_x + 4, cy + 4), fill=ACCENT)
        draw.text((CX, cy), text, fill=MUTED, font=f, anchor="mm")
        y += row_h
    return y


canvas = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(canvas)
draw_network(draw, random.Random(11))
draw_brand(draw)

draw.rounded_rectangle(PANEL_BOX, radius=18, fill=PANEL, outline=PANEL_EDGE, width=1)

# --- Headline block ---
y = PANEL_BOX[1] + 26
draw.text((CX, y), "One chat isn't a team.", fill=TEXT, font=font(True, 32), anchor="mm")
y += 40
fsub = font(False, 17)
sub_lines = wrap(
    draw,
    "Specialists orchestrate each other — you approve, they execute.",
    fsub,
    INNER_W - 40,
)
y = draw_centered_lines(draw, y, sub_lines[:2], fsub, MUTED, 22)

# --- Mock UI ---
y += 14
mock_h = 236
mock_box = (INNER_L, y, INNER_R, y + mock_h)
draw_collab_mock(draw, mock_box)
y = mock_box[3] + 22

# --- Workflow ---
y = draw_workflow(draw, y, INNER_L, INNER_R) + 20

# --- Callouts ---
y = draw_callouts(draw, y) + 14

# --- Command strip ---
cmds = "/collaborate   /approve-plan   /revise-plan   /collab-status"
draw.text((CX, y), cmds, fill=MUTED, font=font(False, 13, mono=True), anchor="mm")
y += 32

# --- CTA (centered, fixed width) ---
cta_w = 800
cta_x0 = (W - cta_w) // 2
cta_h = 50
draw.rounded_rectangle((cta_x0, y, cta_x0 + cta_w, y + cta_h), radius=12, fill=ACCENT)
draw.text((CX, y + cta_h // 2), "Open source beta — macOS · Windows · Linux", fill=TEXT, font=font(True, 20), anchor="mm")
y += cta_h + 14

draw.text(
    (CX, y),
    "github.com/camronwood/neural-junkie/releases",
    fill=MUTED,
    font=font(False, 15),
    anchor="mm",
)

canvas.save(OUT, "PNG")
print(f"Wrote {OUT}")
PY
