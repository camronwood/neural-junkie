#!/usr/bin/env python3
"""Render LoRA v2 marketing graphics (LinkedIn article cover + square ad)."""

from __future__ import annotations

import argparse
import math
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

BG_TOP = (14, 16, 32)
BG_BOTTOM = (8, 20, 28)
PANEL = (18, 24, 44)
TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
PINK = (199, 125, 255)
ACCENT = (233, 69, 96)
GREEN = (72, 199, 142)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
DIM = (136, 140, 168)
ARROW = (80, 88, 110)


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


def apply_dot_grid(canvas: Image.Image, step: int = 32) -> Image.Image:
    w, h = canvas.size
    overlay = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    od = ImageDraw.Draw(overlay)
    for x in range(0, w, step):
        for y in range(0, h, step):
            od.ellipse((x - 1, y - 1, x + 1, y + 1), fill=(255, 255, 255, 18))
    return Image.alpha_composite(canvas.convert("RGBA"), overlay).convert("RGB")


def draw_stack_row(
    draw: ImageDraw.ImageDraw,
    x0: int,
    x1: int,
    y0: int,
    height: int,
    label: str,
    sub: str,
    accent: tuple[int, int, int],
    *,
    indent: int = 0,
) -> None:
    ix0 = x0 + indent
    y1 = y0 + height
    draw.rounded_rectangle((ix0, y0, x1, y1), radius=8, fill=PANEL, outline=accent, width=2)
    draw.rounded_rectangle((ix0 + 2, y0 + 4, ix0 + 6, y1 - 4), radius=2, fill=accent)
    draw.text((ix0 + 14, y0 + 8), label, fill=accent, font=font(True, 10), anchor="lt")
    if sub:
        draw.text((ix0 + 14, y0 + 24), sub, fill=MUTED, font=font(False, 8, mono=True), anchor="lt")


def draw_loop_node(draw: ImageDraw.ImageDraw, cx: int, cy: int, r: int, label: str, accent: tuple[int, int, int]) -> None:
    draw.ellipse((cx - r, cy - r, cx + r, cy + r), fill=(20, 28, 44), outline=accent, width=2)
    for i, line in enumerate(wrap(draw, label, font(True, 8), r * 2 - 8)):
        draw.text((cx, cy - 6 + i * 10), line, fill=accent, font=font(True, 8), anchor="mm")


def draw_loop_arrow(draw: ImageDraw.ImageDraw, x0: int, y0: int, x1: int, y1: int, color=ARROW) -> None:
    draw.line([(x0, y0), (x1, y1)], fill=color, width=2)
    # simple arrowhead toward (x1, y1)
    if abs(x1 - x0) > abs(y1 - y0):
        tip = (x1 - 6 if x1 > x0 else x1 + 6, y1)
        draw.polygon([(x1, y1), (tip[0], y1 - 4), (tip[0], y1 + 4)], fill=color)
    else:
        tip = (x1, y1 - 6 if y1 > y0 else y1 + 6)
        draw.polygon([(x1, y1), (x1 - 4, tip[1]), (x1 + 4, tip[1])], fill=color)


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    # ── Left column ──
    lx, lw = margin, 500
    draw.rounded_rectangle((lx, 32, lx + 200, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 100, 45), "OPEN SOURCE · LORA V2", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "TRAIN ONCE.", fill=TEXT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 46), "COMPOUND", fill=GREEN, font=font(True, 42), anchor="lt")
    draw.text((lx, hy + 96), "FOREVER.", fill=AMBER, font=font(True, 38), anchor="lt")

    sy = hy + 144
    for line in wrap(
        draw,
        "Incremental refresh, dual-tag profiles, unified routing, MLX training, and team sharing.",
        font(False, 14),
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=font(False, 14), anchor="lt")
        sy += 20

    pillars = [
        ("Compound refresh loop", GREEN),
        ("Dual-tag model profiles", TEAL),
        ("Post-train eval gate", AMBER),
        ("MCP + Hugging Face sharing", PINK),
    ]
    for text, accent in pillars:
        draw.text((lx, sy), "▸", fill=accent, font=font(True, 13), anchor="lt")
        draw.text((lx + 18, sy), text, fill=MUTED, font=font(False, 12), anchor="lt")
        sy += 22

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=GREEN)
    draw.text((lx + lw // 2, 546), "github.com/camronwood/neural-junkie", fill=TEXT, font=font(True, 14), anchor="mm")
    draw.text((lx + lw // 2, h - 16), "Prompt-time memory + weight-time compounding", fill=DIM, font=font(False, 10), anchor="ms")

    # ── Right column — full panel with stack + loop + profile ──
    rx = 560
    rw = w - margin - rx
    panel_x0, panel_x1 = rx, rx + rw
    panel_y0, panel_y1 = 36, 580

    draw.rounded_rectangle((panel_x0, panel_y0, panel_x1, panel_y1), radius=14, fill=(12, 16, 30), outline=TEAL, width=2)
    draw.text((panel_x0 + 16, panel_y0 + 14), "COMPOUND LEARNING LOOP", fill=AMBER, font=font(True, 11), anchor="lt")

    # Adapter stack (left side of panel)
    stack_x0 = panel_x0 + 16
    stack_x1 = panel_x0 + rw // 2 + 20
    stack_y = panel_y0 + 40
    row_h = 38
    row_gap = 6
    layers = [
        ("BASE WEIGHTS", "llama3.2:3b", MUTED, 0),
        ("LORA Δ v1", "initial train", TEAL, 6),
        ("LORA Δ v2", "incremental refresh", GREEN, 12),
        ("LORA Δ v3", "curated delta rows", GREEN, 18),
        ("COMPOSED TAG", "nj-repo-* in Ollama", PINK, 24),
    ]
    for label, sub, accent, indent in layers:
        draw_stack_row(draw, stack_x0, stack_x1, stack_y, row_h, label, sub, accent, indent=indent)
        if stack_y + row_h < panel_y1 - 160:
            draw.text((stack_x1 - 8, stack_y + row_h + 1), "+", fill=ARROW, font=font(True, 10), anchor="lt")
        stack_y += row_h + row_gap

    # Cyclic loop diagram (right side of panel)
    loop_cx = panel_x0 + rw * 3 // 4
    loop_cy = panel_y0 + 200
    loop_r = 88
    draw.ellipse(
        (loop_cx - loop_r - 20, loop_cy - loop_r - 20, loop_cx + loop_r + 20, loop_cy + loop_r + 20),
        outline=(40, 48, 72),
        width=1,
    )
    nodes = [
        (loop_cx, loop_cy - loop_r + 10, "learn", PINK),
        (loop_cx + loop_r - 10, loop_cy, "refresh", GREEN),
        (loop_cx, loop_cy + loop_r - 10, "eval", AMBER),
        (loop_cx - loop_r + 10, loop_cy, "assign", TEAL),
    ]
    for i, (nx, ny, label, accent) in enumerate(nodes):
        draw_loop_node(draw, nx, ny, 28, label, accent)
        nxt = nodes[(i + 1) % len(nodes)]
        draw_loop_arrow(draw, nx + 20, ny, nxt[0] - 20, nxt[1], ARROW)

    draw.text((loop_cx, loop_cy), "↻", fill=GREEN, font=font(True, 28), anchor="mm")

    # v1 → v2 callout
    callout_x0 = stack_x1 + 8
    callout_y0 = panel_y0 + 130
    callout_x1 = loop_cx - loop_r - 28
    callout_y1 = callout_y0 + 56
    if callout_x1 > callout_x0 + 60:
        draw.rounded_rectangle((callout_x0, callout_y0, callout_x1, callout_y1), radius=8, fill=(16, 22, 38), outline=AMBER, width=1)
        draw.text((callout_x0 + 10, callout_y0 + 10), "v1 → v2", fill=AMBER, font=font(True, 9), anchor="lt")
        draw.text((callout_x0 + 10, callout_y0 + 26), "one-shot → compound", fill=MUTED, font=font(False, 8), anchor="lt")
        draw.text((callout_x0 + 10, callout_y0 + 40), "rollback anytime", fill=MUTED, font=font(False, 8), anchor="lt")

    # model_profile footer inside panel
    prof_y0 = panel_y1 - 88
    prof_y1 = panel_y1 - 16
    draw.rounded_rectangle((panel_x0 + 16, prof_y0, panel_x1 - 16, prof_y1), radius=10, fill=(16, 22, 38), outline=TEAL, width=2)
    draw.text((panel_x0 + 28, prof_y0 + 12), "model_profile", fill=TEAL, font=font(True, 10, mono=True), anchor="lt")

    tag_w = (rw - 64) // 3
    for i, (title, tag, accent) in enumerate([
        ("inference", "qwen3.5:9b", TEAL),
        ("compose", "llama3.2:3b", AMBER),
        ("tools", "qwen fallback", PINK),
    ]):
        tx0 = panel_x0 + 24 + i * (tag_w + 8)
        tx1 = tx0 + tag_w - 8
        draw.rounded_rectangle((tx0, prof_y0 + 30, tx1, prof_y1 - 12), radius=8, fill=PANEL, outline=accent, width=1)
        draw.text((tx0 + 10, prof_y0 + 38), title, fill=accent, font=font(True, 8), anchor="lt")
        draw.text((tx0 + 10, prof_y0 + 50), tag, fill=MUTED, font=font(False, 8, mono=True), anchor="lt")

    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")
    if gallery:
        gallery.parent.mkdir(parents=True, exist_ok=True)
        canvas.save(gallery, "PNG")
        print(f"Wrote {gallery}")


def render_square_ad(out: Path) -> None:
    s = 1080
    margin = 56
    canvas = Image.new("RGB", (s, s), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas, step=36)
    draw = ImageDraw.Draw(canvas)

    draw.text((margin, margin), "LORA V2", fill=TEAL, font=font(True, 14), anchor="lt")
    draw.text((margin, margin + 40), "Compound", fill=TEXT, font=font(True, 52), anchor="lt")
    draw.text((margin, margin + 100), "specialists", fill=GREEN, font=font(True, 52), anchor="lt")

    stack_x = margin
    stack_y = margin + 180
    for label, accent in [
        ("base weights", MUTED),
        ("+ adapter v1", TEAL),
        ("+ delta v2", GREEN),
        ("→ composed tag", PINK),
    ]:
        bh = 44
        draw.rounded_rectangle((stack_x, stack_y, stack_x + 420, stack_y + bh), radius=10, fill=PANEL, outline=accent, width=2)
        draw.text((stack_x + 16, stack_y + 14), label, fill=accent, font=font(True, 14), anchor="lt")
        stack_y += bh + 10

    loop_cx, loop_cy, loop_r = stack_x + 520, margin + 300, 70
    for i, (angle, label, accent) in enumerate([
        (270, "learn", PINK),
        (0, "refresh", GREEN),
        (90, "eval", AMBER),
        (180, "assign", TEAL),
    ]):
        rad = math.radians(angle)
        nx = loop_cx + int(loop_r * math.sin(rad))
        ny = loop_cy - int(loop_r * math.cos(rad))
        draw_loop_node(draw, nx, ny, 26, label, accent)

    pillars = ["Incremental refresh", "Dual-tag profiles", "MLX on Mac", "Post-train eval", "Team adapters"]
    py = margin + 400
    for i, p in enumerate(pillars):
        accent = [GREEN, TEAL, AMBER, PINK, ACCENT][i]
        draw.text((margin, py), "▸ " + p, fill=accent, font=font(False, 26), anchor="lt")
        py += 48

    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Render LoRA v2 marketing graphics")
    parser.add_argument(
        "variant",
        choices=["article", "all"],
        help="which graphic to render",
    )
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    root = args.root
    assets = root / "campaigns" / "lora-v2" / "creatives"
    assets.mkdir(parents=True, exist_ok=True)
    covers = root / "docs" / "media" / "articles" / "covers"
    gallery_ads = root / "docs" / "media" / "gallery" / "ads"

    header = assets / "neural-junkie-lora-v2-1200.png"
    ad = assets / "neural-junkie-lora-v2-ad-1080.png"

    if args.variant in ("article", "all"):
        render_article_cover(header, covers / "neural-junkie-lora-v2-1200.png")
    if args.variant in ("all",):
        render_square_ad(ad)
        render_square_ad(gallery_ads / "neural-junkie-lora-v2-ad-1080.png")


if __name__ == "__main__":
    main()
