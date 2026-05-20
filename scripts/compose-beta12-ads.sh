#!/usr/bin/env bash
# Compose v1.0.0-beta.12 feature ads (1080×1080).
# Usage: ./scripts/compose-beta12-ads.sh [packs|session-memory|runbook-actions|biology|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
VARIANT="${1:-all}"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT/assets" "$VARIANT" <<'PY'
import math
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ASSETS = Path(sys.argv[1])
VARIANT = sys.argv[2]
W, H = 1080, 1080
MARGIN = 56
BG_TOP = (13, 13, 26)
BG_BOT = (22, 18, 42)
ACCENT = (233, 69, 96)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
PANEL = (18, 24, 44)
PANEL_EDGE = (48, 58, 88)
PINK = (199, 125, 255)
GREEN = (72, 199, 142)
BLUE = (100, 180, 255)


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


def gradient_bg(canvas: Image.Image):
    draw = ImageDraw.Draw(canvas)
    for y in range(H):
        t = y / max(H - 1, 1)
        r = int(BG_TOP[0] + (BG_BOT[0] - BG_TOP[0]) * t)
        g = int(BG_TOP[1] + (BG_BOT[1] - BG_TOP[1]) * t)
        b = int(BG_TOP[2] + (BG_BOT[2] - BG_TOP[2]) * t)
        draw.line([(0, y), (W, y)], fill=(r, g, b))


def draw_brand(draw, kicker: str):
    draw.text((W // 2, 40), kicker, fill=MUTED, font=font(False, 13), anchor="mm")
    title = "NEURAL JUNKIE"
    ft = font(True, 48)
    tw = draw.textlength(title, font=ft)
    tx = (W - tw) // 2
    draw.text((tx, 64), title, fill=TEXT, font=ft)
    draw.rectangle((tx, 118, tx + tw, 122), fill=ACCENT)


def draw_cta(draw, y_btn: int):
    btn_h = 56
    draw.rounded_rectangle(
        (MARGIN, y_btn, W - MARGIN, y_btn + btn_h),
        radius=14,
        fill=ACCENT,
    )
    draw.text(
        (W // 2, y_btn + btn_h // 2),
        "Download — v1.0.0-beta.12",
        fill=TEXT,
        font=font(True, 20),
        anchor="mm",
    )
    draw.text(
        (W // 2, y_btn + btn_h + 26),
        "github.com/camronwood/neural-junkie/releases",
        fill=(120, 128, 150),
        font=font(False, 14),
        anchor="mm",
    )


def new_canvas():
    img = Image.new("RGB", (W, H), BG_TOP)
    gradient_bg(img)
    return img, ImageDraw.Draw(img)


def ad_packs():
    img, draw = new_canvas()
    draw_brand(draw, "OPEN BETA · DOMAIN PACKS")
    headline = "Dev team or lab team.\nOne toggle."
    hf = font(True, 44)
    y = 150
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 52
    box = (MARGIN, 280, W - MARGIN, 720)
    draw.rounded_rectangle(box, radius=18, fill=PANEL, outline=PANEL_EDGE, width=2)
    col_w = (box[2] - box[0] - 48) // 2
    x1, x2 = box[0] + 20, box[0] + 24 + col_w
    y0 = box[1] + 24
    for x, title, color, bullets in [
        (x1, "Software dev", GREEN, ["Go · React · Rust", "DevOps · SQL · Security", "MCP analysis tools"]),
        (x2, "Life sciences", BLUE, ["BiologyExpert", "Sequence analysis", "ESMFold structures"]),
    ]:
        draw.rounded_rectangle((x, y0, x + col_w, box[3] - 24), radius=12, fill=(24, 30, 52), outline=color, width=2)
        draw.text((x + col_w // 2, y0 + 28), title, fill=color, font=font(True, 22), anchor="mm")
        ty = y0 + 56
        for b in bullets:
            draw.text((x + 16, ty), f"• {b}", fill=MUTED, font=font(False, 17), anchor="lt")
            ty += 30
    sub = "Settings → Domain packs · Fresh install stays lean"
    draw.text((W // 2, 748), sub, fill=MUTED, font=font(False, 18), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-domain-packs-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_session_memory():
    img, draw = new_canvas()
    draw_brand(draw, "FOR DEVELOPERS · CONTEXT MODEL")
    headline = "DMs that remember\nwithout re-pasting"
    hf = font(True, 42)
    y = 150
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 50
    # intent ladder
    labels = ["closure", "low", "meta", "full"]
    colors = [(80, 80, 100), (100, 120, 150), (130, 100, 170), PINK]
    bar_w = 760
    x0 = (W - bar_w) // 2
    y0 = 300
    draw.text((W // 2, y0 - 28), "Turn intent router", fill=MUTED, font=font(False, 14), anchor="mm")
    step = bar_w // (len(labels) - 1)
    for i, (lab, col) in enumerate(zip(labels, colors)):
        x = x0 + i * step
        h = 40 + i * 18
        draw.rounded_rectangle((x - 36, y0, x + 36, y0 + h), radius=8, fill=col)
        draw.text((x, y0 + h + 18), lab, fill=MUTED, font=font(False, 12), anchor="mm")
    # session summary panel
    box = (MARGIN, 480, W - MARGIN, 700)
    draw.rounded_rectangle(box, radius=14, fill=PANEL, outline=PINK, width=2)
    draw.text((box[0] + 20, box[1] + 16), "SESSION SUMMARY (rolling)", fill=PINK, font=font(True, 14), anchor="lt")
    body = (
        "User is refactoring auth middleware.\n"
        "Prefers Rust + tests in collab channel.\n"
        "Last decision: use JWT + refresh rotation."
    )
    ty = box[1] + 48
    for line in body.split("\n"):
        draw.text((box[0] + 20, ty), line, fill=MUTED, font=font(False, 17), anchor="lt")
        ty += 28
    draw.text((W // 2, 730), "qwen2.5:7b · every 3 turns · Clear history resets", fill=MUTED, font=font(False, 16), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-session-memory-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_runbook_actions():
    img, draw = new_canvas()
    draw_brand(draw, "RUNBOOKS · DETERMINISTIC STEPS")
    headline = "HTTP in the DAG.\nNo LLM turn."
    hf = font(True, 44)
    y = 150
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 52
    # mini DAG
    nodes = [
        (W // 2 - 200, 340, "Plan", MUTED),
        (W // 2, 420, "http_get", GREEN),
        (W // 2 + 200, 500, "branch?", PINK),
        (W // 2 - 120, 580, "@GoExpert", ACCENT),
        (W // 2 + 120, 580, "webhook", BLUE),
    ]
    edges = [(0, 1), (1, 2), (2, 3), (2, 4)]
    for i, j in edges:
        x1, y1 = nodes[i][0] + 50, nodes[i][1] + 22
        x2, y2 = nodes[j][0], nodes[j][1]
        draw.line([(x1, y1), (x2, y2)], fill=PANEL_EDGE, width=3)
    for x, y, label, col in nodes:
        draw.rounded_rectangle((x, y, x + 100, y + 44), radius=10, fill=(24, 32, 58), outline=col, width=2)
        draw.text((x + 50, y + 22), label, fill=TEXT, font=font(True, 13), anchor="mm")
    chips = ["http_get · http_post", "webhook · templates", "conditional edges"]
    ty = 660
    for c in chips:
        draw.text((W // 2, ty), c, fill=MUTED, font=font(False, 17), anchor="mm")
        ty += 28
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-runbook-actions-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_biology():
    img, draw = new_canvas()
    draw_brand(draw, "LIFE SCIENCES PACK")
    headline = "Sequences in.\nStructures out."
    hf = font(True, 46)
    y = 150
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 54
    box = (MARGIN, 280, W - MARGIN, 720)
    draw.rounded_rectangle(box, radius=18, fill=PANEL, outline=BLUE, width=2)
    draw.text((box[0] + 24, box[1] + 20), "ATCG… → analyze_sequence", fill=GREEN, font=font(True, 16), anchor="lt")
    seq = "MKTAYIAKQRQISFVKSHFSRQLEERLGLIEVQAPILSRVGDGTQDNLSGAEKAVQVKVKALPDAQFEVVHSLAKWKRQQIAAS"
    mono = font(False, 14, mono=True)
    ty = box[1] + 56
    chunk = 44
    for i in range(0, min(len(seq), 132), chunk):
        draw.text((box[0] + 24, ty), seq[i : i + chunk], fill=(140, 200, 160), font=mono, anchor="lt")
        ty += 22
    draw.text((box[0] + 24, ty + 16), "↓ ESMFold (HF) → PDB artifact", fill=BLUE, font=font(True, 16), anchor="lt")
    draw.text((box[0] + 24, ty + 48), "BiologyExpert · OpenBioLLM 8B · runbook templates", fill=MUTED, font=font(False, 16), anchor="lt")
    draw.text((W // 2, 748), "Research use · toggle in Settings → Domain packs", fill=MUTED, font=font(False, 17), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-biology-pack-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


variants = {
    "packs": ad_packs,
    "session-memory": ad_session_memory,
    "runbook-actions": ad_runbook_actions,
    "biology": ad_biology,
}
if VARIANT == "all":
    for fn in variants.values():
        fn()
else:
    if VARIANT not in variants:
        raise SystemExit(f"unknown variant: {VARIANT}")
    variants[VARIANT]()
PY
