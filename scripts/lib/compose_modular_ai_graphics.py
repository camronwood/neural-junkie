#!/usr/bin/env python3
"""Render modular-AI marketing graphics (square ads + LinkedIn article cover)."""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

# Brand palette (matches other Neural Junkie ads)
BG_TOP = (14, 16, 32)
BG_BOTTOM = (8, 20, 28)
PANEL = (18, 24, 44)
PANEL_SOFT = (14, 20, 36)
TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
PINK = (199, 125, 255)
ACCENT = (233, 69, 96)
GREEN = (72, 199, 142)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
DIM = (136, 140, 168)
ARROW = (80, 88, 110)
RELEASE = "github.com/camronwood/neural-junkie/releases"


def font(bold: bool, size: int, mono: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    paths = ["/System/Library/Fonts/Menlo.ttc"] if mono else [
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold else "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/Library/Fonts/Arial Bold.ttf" if bold else "/Library/Fonts/Arial.ttf",
    ]
    for path in paths:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def wrap(draw: ImageDraw.ImageDraw, text: str, fnt, max_w: int) -> list[str]:
    words = text.split()
    lines: list[str] = []
    cur = ""
    for word in words:
        test = f"{cur} {word}".strip()
        if draw.textlength(test, font=fnt) <= max_w:
            cur = test
        else:
            if cur:
                lines.append(cur)
            cur = word
    if cur:
        lines.append(cur)
    return lines


def gradient_bg(canvas: Image.Image, top=BG_TOP, bottom=BG_BOTTOM) -> None:
    draw = ImageDraw.Draw(canvas)
    w, h = canvas.size
    for y in range(h):
        t = y / max(h - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (w, y)], fill=(r, g, b))


def apply_dot_grid(canvas: Image.Image, step: int = 36) -> Image.Image:
    w, h = canvas.size
    overlay = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    od = ImageDraw.Draw(overlay)
    for x in range(0, w, step):
        for y in range(0, h, step):
            od.ellipse((x - 1, y - 1, x + 1, y + 1), fill=(255, 255, 255, 18))
    return Image.alpha_composite(canvas.convert("RGBA"), overlay).convert("RGB")


def top_badge(draw: ImageDraw.ImageDraw, w: int, label: str, accent: tuple[int, int, int]) -> None:
    bf = font(True, 11)
    bw = int(draw.textlength(label, font=bf)) + 28
    bx0 = (w - bw) // 2
    by0 = 44
    draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 30), radius=8, fill=(20, 28, 40), outline=accent, width=2)
    draw.text((w // 2, by0 + 15), label, fill=accent, font=bf, anchor="mm")


def headline_block(
    draw: ImageDraw.ImageDraw,
    w: int,
    y: int,
    lines: list[tuple[str, tuple[int, int, int] | None]],
    size: int = 46,
) -> int:
    for text, color in lines:
        draw.text((w // 2, y), text, fill=color or TEXT, font=font(True, size), anchor="mm")
        y += size + 6
    return y


def subhead(draw: ImageDraw.ImageDraw, w: int, y: int, text: str, margin: int = 72) -> int:
    fsub = font(False, 17)
    for line in wrap(draw, text, fsub, w - 2 * margin):
        draw.text((w // 2, y), line, fill=(200, 205, 220), font=fsub, anchor="mm")
        y += 24
    return y + 8


def draw_arrow(draw: ImageDraw.ImageDraw, x0: int, y: int, x1: int, color=ARROW) -> None:
    draw.line([(x0, y), (x1 - 8, y)], fill=color, width=3)
    draw.polygon([(x1, y), (x1 - 10, y - 5), (x1 - 10, y + 5)], fill=color)


def flow_box(
    draw: ImageDraw.ImageDraw,
    cx: int,
    cy: int,
    bw: int,
    bh: int,
    title: str,
    sub: str,
    accent: tuple[int, int, int],
    mono_sub: bool = False,
) -> tuple[int, int, int, int]:
    x0, y0 = cx - bw // 2, cy - bh // 2
    x1, y1 = x0 + bw, y0 + bh
    draw.rounded_rectangle((x0, y0, x1, y1), radius=12, fill=PANEL, outline=accent, width=2)
    draw.text((cx, y0 + 22), title, fill=TEXT, font=font(True, 14), anchor="mm")
    if sub:
        sf = font(False, 10 if not mono_sub else 9, mono=mono_sub)
        for i, line in enumerate(wrap(draw, sub, sf, bw - 16)[:2]):
            draw.text((cx, y0 + 44 + i * 14), line, fill=MUTED, font=sf, anchor="mm")
    return x0, y0, x1, y1


def cta_footer(draw: ImageDraw.ImageDraw, w: int, h: int, margin: int, tagline: str = "") -> None:
    draw.rounded_rectangle((margin, h - 148, w - margin, h - 88), radius=14, fill=ACCENT)
    draw.text((w // 2, h - 118), RELEASE, fill=TEXT, font=font(True, 20), anchor="mm")
    if tagline:
        draw.text((w // 2, h - 52), tagline, fill=DIM, font=font(False, 14), anchor="mm")


def render_router_ad(out: Path) -> None:
    w, h = 1080, 1080
    margin = 56
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    top_badge(draw, w, "MODULAR AI · ROUTER", TEAL)
    y = 100
    draw.text((w // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
    y += 40
    y = headline_block(draw, w, y, [("A 9B ROUTER", TEXT), ("NOT A 70B GUESS", TEAL)], size=48)
    y = subhead(draw, w, y + 8, "Small classifier picks domain + cost tier. Rules fallback when confidence is low.")

    # Flow: prompt → classifier → branches
    cy = y + 100
    stages = [
        ("USER PROMPT", "collab · delegation · impl", MUTED),
        ("9B CLASSIFIER", "qwen3.5:9b · JSON", TEAL),
    ]
    box_w, box_h = 200, 76
    gap = 56
    total = len(stages) * box_w + gap
    sx = (w - total) // 2 + box_w // 2
    prev_right = None
    for i, (title, sub, accent) in enumerate(stages):
        cx = sx + i * (box_w + gap)
        x0, y0, x1, y1 = flow_box(draw, cx, cy, box_w, box_h, title, sub, accent)
        if prev_right is not None:
            draw_arrow(draw, prev_right + 6, cy, x0 - 6)
        prev_right = x1

    # Branch targets
    branch_y = cy + box_h // 2 + 72
    branches = [
        ("SECURITY", "nj-security:14b", PINK),
        ("BIOLOGY", "koesn / OpenBio", GREEN),
        ("CHEAP TIER", "qwen3.5:9b", AMBER),
    ]
    bw, bh = 248, 68
    bgap = 20
    btotal = len(branches) * bw + (len(branches) - 1) * bgap
    bx = (w - btotal) // 2 + bw // 2
    classifier_cx = sx + box_w + gap
    for i, (title, sub, accent) in enumerate(branches):
        cx = bx + i * (bw + bgap)
        flow_box(draw, cx, branch_y, bw, bh, title, sub, accent)
        draw.line([(classifier_cx, cy + box_h // 2), (classifier_cx, branch_y - bh // 2 - 8)], fill=accent, width=2)
        draw.line([(classifier_cx, branch_y - bh // 2 - 8), (cx, branch_y - bh // 2 - 8)], fill=accent, width=2)
        draw.line([(cx, branch_y - bh // 2 - 8), (cx, branch_y - bh // 2)], fill=accent, width=2)

    strip_y = branch_y + bh // 2 + 36
    draw.rounded_rectangle((margin, strip_y, w - margin, strip_y + 88), radius=14, fill=PANEL_SOFT, outline=TEAL, width=2)
    draw.text((margin + 20, strip_y + 16), "One router · three code paths unified", fill=TEAL, font=font(True, 14), anchor="lt")
    bullets = [
        "Collab task routing + delegation consults",
        "LLM default · rules fallback · 80+ table tests",
        "Debug: GET /api/debug/routing-classify",
    ]
    by = strip_y + 40
    for item in bullets:
        draw.text((margin + 28, by), "✓", fill=GREEN, font=font(True, 13), anchor="lt")
        draw.text((margin + 48, by), item, fill=(210, 214, 224), font=font(False, 13), anchor="lt")
        by += 22

    cta_footer(draw, w, h, margin, "open source · macOS · Windows · Linux")
    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_observe_ad(out: Path) -> None:
    w, h = 1080, 1080
    margin = 56
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    top_badge(draw, w, "MODULAR AI · OBSERVE", PINK)
    y = 100
    draw.text((w // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
    y += 40
    y = headline_block(draw, w, y, [("WHICH MODEL", TEXT), ("RAN?", PINK)], size=52)
    y = subhead(draw, w, y + 4, "Routing badge on every agent reply — chat, tool, reason, and classifier source.")

    # Mock chat panel
    panel_x0, panel_y0 = margin, y + 24
    panel_x1, panel_y1 = w - margin, panel_y0 + 420
    draw.rounded_rectangle((panel_x0, panel_y0, panel_x1, panel_y1), radius=18, fill=(12, 18, 34), outline=(60, 68, 96), width=2)
    draw.text((panel_x0 + 24, panel_y0 + 20), "SecurityExpert", fill=TEAL, font=font(True, 14), anchor="lt")
    draw.text((panel_x0 + 24, panel_y0 + 42), "Agent", fill=DIM, font=font(False, 11), anchor="lt")

    bubble_x0 = panel_x0 + 24
    bubble_y0 = panel_y0 + 72
    bubble_x1 = panel_x1 - 80
    bubble_y1 = bubble_y0 + 120
    draw.rounded_rectangle((bubble_x0, bubble_y0, bubble_x1, bubble_y1), radius=14, fill=PANEL, outline=(70, 78, 108), width=1)
    msg = "I'll scan the repo for SQL injection patterns in the auth handlers."
    mf = font(False, 15)
    my = bubble_y0 + 20
    for line in wrap(draw, msg, mf, bubble_x1 - bubble_x0 - 32):
        draw.text((bubble_x0 + 18, my), line, fill=(220, 224, 232), font=mf, anchor="lt")
        my += 22

    # Routing badge chip (two-line mono)
    badge_y = bubble_y1 + 20
    line1 = "routing · chat: nj-security:14b · tool: qwen3.5:9b"
    line2 = "source: llm · reason: security_task"
    bf = font(False, 11, mono=True)
    chip_w = panel_x1 - panel_x0 - 48
    chip_x0 = bubble_x0
    chip_h = 48
    draw.rounded_rectangle((chip_x0, badge_y, chip_x0 + chip_w, badge_y + chip_h), radius=8, fill=(28, 22, 48), outline=PINK, width=2)
    draw.text((chip_x0 + 14, badge_y + 12), line1, fill=PINK, font=bf, anchor="lm")
    draw.text((chip_x0 + 14, badge_y + 30), line2, fill=PINK, font=bf, anchor="lm")

    # Full badge preview (wrapped)
    preview_y = badge_y + 58
    draw.text((panel_x0 + 24, preview_y), "Full metadata on the message:", fill=MUTED, font=font(True, 12), anchor="lt")
    meta_lines = [
        "routing_chat_model → nj-security:14b",
        "routing_tool_model → qwen3.5:9b",
        "routing_reason → security_task",
        "routing_source → llm",
    ]
    mf2 = font(False, 12, mono=True)
    ly = preview_y + 26
    for line in meta_lines:
        draw.text((panel_x0 + 32, ly), line, fill=GREEN, font=mf2, anchor="lt")
        ly += 20

    draw.text((panel_x0 + 24, panel_y1 - 36), "Toggle: Settings → Layout → Show routing on messages", fill=DIM, font=font(False, 12), anchor="lt")

    cta_footer(draw, w, h, margin, "modular AI only feels trustworthy when routing is legible")
    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_compose_ad(out: Path) -> None:
    w, h = 1080, 1080
    margin = 56
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    top_badge(draw, w, "MODULAR AI · COMPOSE", GREEN)
    y = 100
    draw.text((w // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
    y += 40
    y = headline_block(
        draw,
        w,
        y,
        [("CHAT BRAIN", TEAL), ("TOOL HANDS", AMBER), ("LORA WEIGHTS", GREEN)],
        size=42,
    )
    y = subhead(draw, w, y + 4, "Packs declare composed specialists — biology pattern for every domain.")

    cards = [
        ("chat_model", "koesn/llama3-openbiollm-8b", "Domain reasoning · OpenBio", TEAL),
        ("tool_model", "qwen3.5:9b", "Fast tool loop · MCP calls", AMBER),
        ("lora_tag", "nj-biology:8b", "Composed specialist weights", GREEN),
    ]
    card_w = w - 2 * margin
    card_h = 108
    gap = 16
    cy = y + 36 + card_h // 2
    for key, value, desc, accent in cards:
        x0, y0 = margin, cy - card_h // 2
        x1, y1 = x0 + card_w, y0 + card_h
        draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=PANEL, outline=accent, width=2)
        draw.rounded_rectangle((x0 + 16, y0 + 16, x0 + 140, y0 + 40), radius=6, fill=(22, 30, 52), outline=accent, width=1)
        draw.text((x0 + 78, y0 + 28), key, fill=accent, font=font(True, 11, mono=True), anchor="mm")
        draw.text((x0 + 24, y0 + 58), value, fill=TEXT, font=font(True, 18, mono=True), anchor="lt")
        draw.text((x0 + 24, y0 + 84), desc, fill=MUTED, font=font(False, 13), anchor="lt")
        cy += card_h + gap

    # YAML snippet
    yaml_y = cy + 20
    draw.rounded_rectangle((margin, yaml_y, w - margin, yaml_y + 130), radius=14, fill=(10, 16, 28), outline=(60, 68, 96), width=2)
    draw.text((margin + 20, yaml_y + 14), "pack.yaml compose block", fill=DIM, font=font(True, 11), anchor="lt")
    yaml_lines = [
        "compose:",
        "  chat_model: koesn/llama3-openbiollm-8b:latest",
        "  tool_model: qwen3.5:9b",
        "  lora_tag: nj-biology:8b",
    ]
    yf = font(False, 11, mono=True)
    ly = yaml_y + 36
    for line in yaml_lines:
        color = PINK if line.startswith("compose") else GREEN if "lora" in line else TEAL if "chat" in line else AMBER
        draw.text((margin + 24, ly), line, fill=color, font=yf, anchor="lt")
        ly += 18

    cta_footer(draw, w, h, margin, "one code path for DM · collab · implementation")
    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_stacks_ad(out: Path) -> None:
    w, h = 1080, 1080
    margin = 56
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    top_badge(draw, w, "MODULAR AI · STACKS", AMBER)
    y = 100
    draw.text((w // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
    y += 40
    y = headline_block(draw, w, y, [("PICK A STACK", TEXT), ("FOR YOUR RAM", AMBER)], size=48)
    y = subhead(draw, w, y + 4, "Inference + LoRA compositions per tier — not download everything.")

    tiers = [
        ("minimal", "8–15 GB", "qwen3.5:9b", "1 LoRA base", TEAL),
        ("light", "16–24 GB", "qwen3.5:9b", "2 bases + 2 tags", GREEN),
        ("recommended", "25–35 GB", "qwen3.5:27b + 9b", "3 bases + 4 tags", AMBER),
        ("heavy", "50+ GB", "qwen3.5:27b + 9b", "full nj-* compose", PINK),
    ]
    row_h = 88
    gap = 12
    table_y = y + 28
    col_w = (w - 2 * margin - 3 * 12) // 4

    headers = ["TIER", "RAM", "INFERENCE", "LORA"]
    hx = margin
    for i, hdr in enumerate(headers):
        cw = col_w if i > 0 else 148
        draw.text((hx + cw // 2, table_y), hdr, fill=DIM, font=font(True, 11), anchor="mm")
        hx += cw + 12

    for row_i, (tier, ram, infer, lora, accent) in enumerate(tiers):
        ry = table_y + 28 + row_i * (row_h + gap)
        draw.rounded_rectangle((margin, ry, w - margin, ry + row_h), radius=12, fill=PANEL, outline=accent, width=2)
        cols = [tier.upper(), ram, infer, lora]
        hx = margin + 12
        widths = [148, col_w, col_w, col_w]
        for col_i, (cell, cw) in enumerate(zip(cols, widths)):
            color = accent if col_i == 0 else TEXT if col_i < 3 else MUTED
            fnt = font(True, 13 if col_i == 0 else 12, mono=col_i > 0)
            draw.text((hx + 12, ry + row_h // 2), cell, fill=color, font=fnt, anchor="lm")
            hx += cw + 12

    strip_y = table_y + 28 + len(tiers) * (row_h + gap) + 24
    draw.rounded_rectangle((margin, strip_y, w - margin, strip_y + 96), radius=14, fill=PANEL_SOFT, outline=AMBER, width=2)
    draw.text((margin + 20, strip_y + 16), "GET /api/system/hardware → recommended_stacks[]", fill=AMBER, font=font(True, 13, mono=True), anchor="lt")
    draw.text(
        (margin + 20, strip_y + 44),
        "Two-tier strategy: Qwen for inference · Llama/Mistral bases for nj-* LoRA",
        fill=(210, 214, 224),
        font=font(False, 14),
        anchor="lt",
    )
    draw.text(
        (margin + 20, strip_y + 68),
        "Documented in HARDWARE.md — pick your tier, pull only what you need",
        fill=MUTED,
        font=font(False, 13),
        anchor="lt",
    )

    cta_footer(draw, w, h, margin, "open source · macOS · Windows · Linux")
    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas, step=32)
    draw = ImageDraw.Draw(canvas)

    # Left column — copy
    lx = margin
    lw = 520
    draw.rounded_rectangle((lx, 32, lx + 200, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 100, 45), "OPEN SOURCE · NEURAL JUNKIE", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "MODULAR AI,", fill=TEXT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 46), "ONE HUB", fill=ACCENT, font=font(True, 44), anchor="lt")
    draw.text((lx, hy + 100), "Router · Compose · Observe", fill=AMBER, font=font(True, 18), anchor="lt")

    sy = hy + 138
    fsub = font(False, 15)
    for line in wrap(
        draw,
        "Stop running one model for everything. Small classifier, pack-declared stacks, routing badges on every reply.",
        fsub,
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 22

    bullets = [
        "9B classifier + rules fallback",
        "compose: chat + tool + LoRA in pack.yaml",
        "routing badge on every agent reply",
    ]
    for item in bullets:
        draw.text((lx, sy), "▸", fill=TEAL, font=font(True, 14), anchor="lt")
        draw.text((lx + 18, sy), item, fill=MUTED, font=font(False, 13), anchor="lt")
        sy += 22

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=ACCENT)
    draw.text((lx + lw // 2, 546), RELEASE, fill=TEXT, font=font(True, 15), anchor="mm")

    # Right column — diagram
    rx = 600
    rw = w - rx - margin
    ry = 48

    # Mini flow
    flow_y = ry + 40
    steps = [
        ("PROMPT", MUTED, 100),
        ("9B ROUTER", TEAL, 110),
        ("nj-* · OpenBio", GREEN, 130),
        ("REPLY + BADGE", PINK, 120),
    ]
    cx = rx + 20
    prev_right = None
    for title, accent, bw in steps:
        x0, y0 = cx, flow_y - 28
        x1, y1 = cx + bw, flow_y + 28
        draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=PANEL, outline=accent, width=2)
        draw.text((cx + bw // 2, flow_y), title, fill=accent, font=font(True, 10), anchor="mm")
        if prev_right is not None:
            draw_arrow(draw, prev_right + 4, flow_y, x0 - 4)
        prev_right = x1
        cx += bw + 28

    # Three pillars
    py = flow_y + 72
    pillars = [
        ("ROUTER", "domain · cost · tools", TEAL),
        ("COMPOSE", "chat + tool + LoRA", GREEN),
        ("OBSERVE", "badge every reply", PINK),
    ]
    pw = (rw - 24) // 3
    for i, (title, sub, color) in enumerate(pillars):
        px0 = rx + i * (pw + 12)
        draw.rounded_rectangle((px0, py, px0 + pw, py + 100), radius=12, fill=PANEL_SOFT, outline=color, width=2)
        draw.text((px0 + pw // 2, py + 28), title, fill=color, font=font(True, 13), anchor="mm")
        for j, line in enumerate(wrap(draw, sub, font(False, 11), pw - 16)):
            draw.text((px0 + pw // 2, py + 52 + j * 16), line, fill=MUTED, font=font(False, 11), anchor="mm")

    # Stack preview
    stack_y = py + 128
    draw.rounded_rectangle((rx, stack_y, w - margin, stack_y + 140), radius=14, fill=(10, 16, 28), outline=AMBER, width=2)
    draw.text((rx + 16, stack_y + 14), "recommended_stacks[]", fill=AMBER, font=font(True, 12, mono=True), anchor="lt")
    mini_tiers = [
        ("minimal", "qwen3.5:9b", TEAL),
        ("light", "9b + 2 LoRA", GREEN),
        ("recommended", "27b + 9b + 4 tags", AMBER),
        ("heavy", "full compose", PINK),
    ]
    tw = (rw - 48) // 4
    for i, (tier, models, accent) in enumerate(mini_tiers):
        tx = rx + 16 + i * (tw + 8)
        draw.rounded_rectangle((tx, stack_y + 40, tx + tw, stack_y + 118), radius=8, fill=PANEL, outline=accent, width=1)
        draw.text((tx + tw // 2, stack_y + 58), tier.upper(), fill=accent, font=font(True, 9), anchor="mm")
        draw.text((tx + tw // 2, stack_y + 82), models, fill=MUTED, font=font(False, 9), anchor="mm")

    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")
    if gallery:
        gallery.parent.mkdir(parents=True, exist_ok=True)
        canvas.save(gallery, "PNG")
        print(f"Wrote {gallery}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Render modular-AI marketing graphics")
    parser.add_argument(
        "variant",
        choices=["router", "observe", "compose", "stacks", "article", "all"],
        help="which graphic to render",
    )
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    root = args.root
    assets = root / "campaigns" / "modular-ai" / "creatives"
    assets.mkdir(parents=True, exist_ok=True)
    gallery = root / "docs" / "media" / "gallery" / "ads"

    renders = {
        "router": (render_router_ad, assets / "neural-junkie-modular-router-ad-1080.png"),
        "observe": (render_observe_ad, assets / "neural-junkie-routing-badge-ad-1080.png"),
        "compose": (render_compose_ad, assets / "neural-junkie-compose-specialist-ad-1080.png"),
        "stacks": (render_stacks_ad, assets / "neural-junkie-hardware-stacks-ad-1080.png"),
    }

    if args.variant == "article":
        render_article_cover(
            assets / "neural-junkie-modular-ai-composition-1200.png",
            gallery / "neural-junkie-modular-ai-composition-1200.png",
        )
        return

    if args.variant == "all":
        for fn, path in renders.values():
            fn(path)
        render_article_cover(
            assets / "neural-junkie-modular-ai-composition-1200.png",
            gallery / "neural-junkie-modular-ai-composition-1200.png",
        )
        return

    fn, path = renders[args.variant]
    fn(path)


if __name__ == "__main__":
    main()
