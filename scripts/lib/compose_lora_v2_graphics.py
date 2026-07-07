#!/usr/bin/env python3
"""Render LoRA v2 marketing graphics (LinkedIn article cover + square ad)."""

from __future__ import annotations

import argparse
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


def draw_compound_layer(
    draw: ImageDraw.ImageDraw,
    cx: int,
    y0: int,
    bw: int,
    bh: int,
    label: str,
    sub: str,
    accent: tuple[int, int, int],
    *,
    offset: int = 0,
) -> None:
    x0 = cx - bw // 2 + offset
    x1 = x0 + bw
    y1 = y0 + bh
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=PANEL, outline=accent, width=2)
    draw.rounded_rectangle((x0 + 3, y0 + 4, x0 + 7, y1 - 4), radius=2, fill=accent)
    draw.text((x0 + 16, y0 + 10), label, fill=accent, font=font(True, 11), anchor="lt")
    draw.text((x0 + 16, y0 + 28), sub, fill=MUTED, font=font(False, 9, mono=True), anchor="lt")


def draw_refresh_arc(draw: ImageDraw.ImageDraw, cx: int, cy: int, r: int, accent: tuple[int, int, int]) -> None:
    bbox = (cx - r, cy - r, cx + r, cy + r)
    draw.arc(bbox, start=200, end=340, fill=accent, width=3)
    # arrow head
    ax, ay = cx + int(r * 0.64), cy - int(r * 0.77)
    draw.polygon([(ax, ay), (ax - 8, ay + 4), (ax - 2, ay + 10)], fill=accent)


def draw_growth_dots(draw: ImageDraw.ImageDraw, x0: int, y0: int, accent: tuple[int, int, int]) -> None:
    sizes = [4, 6, 8, 10, 12]
    for i, s in enumerate(sizes):
        x = x0 + i * 18
        y = y0 - i * 6
        draw.ellipse((x - s, y - s, x + s, y + s), fill=accent if i >= 3 else (accent[0] // 2, accent[1] // 2, accent[2] // 2))


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    lx = margin
    lw = 480
    draw.rounded_rectangle((lx, 32, lx + 200, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 100, 45), "OPEN SOURCE · LORA V2", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "TRAIN ONCE.", fill=TEXT, font=font(True, 42), anchor="lt")
    draw.text((lx, hy + 50), "COMPOUND", fill=GREEN, font=font(True, 44), anchor="lt")
    draw.text((lx, hy + 102), "FOREVER.", fill=AMBER, font=font(True, 40), anchor="lt")

    sy = hy + 158
    fsub = font(False, 14)
    for line in wrap(
        draw,
        "Incremental refresh, dual-tag profiles, unified routing, MLX training, and team sharing — weights that keep learning.",
        fsub,
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 20

    pills = [
        ("refresh", GREEN),
        ("rollback", TEAL),
        ("eval gate", AMBER),
        ("MCP + HF", PINK),
    ]
    px = lx
    for label, accent in pills:
        tw = int(draw.textlength(label, font=font(True, 9))) + 18
        draw.rounded_rectangle((px, sy, px + tw, sy + 22), radius=11, fill=(20, 28, 44), outline=accent, width=1)
        draw.text((px + tw // 2, sy + 11), label, fill=accent, font=font(True, 9), anchor="mm")
        px += tw + 8

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=GREEN)
    draw.text((lx + lw // 2, 546), "github.com/camronwood/neural-junkie · LORA_V2.md", fill=TEXT, font=font(True, 13), anchor="mm")

    # Right — compound stack + loop
    rx = 560
    stack_cx = rx + (w - margin - rx) // 2
    stack_top = 56

    draw.text((rx, 40), "COMPOUND STACK", fill=AMBER, font=font(True, 12), anchor="lt")

    layers = [
        ("BASE WEIGHTS", "llama3.2:3b · compose tier", MUTED, 0, 52),
        ("LORA Δ v1", "repo sessions · 10+ turns", TEAL, 8, 44),
        ("LORA Δ v2", "incremental refresh", GREEN, 16, 44),
        ("LORA Δ v3", "curated delta rows", GREEN, 24, 44),
        ("COMPOSED TAG", "nj-repo-* · one UI tag", PINK, 32, 48),
    ]
    y = stack_top
    for label, sub, accent, offset, bh in layers:
        draw_compound_layer(draw, stack_cx, y, 280, bh, label, sub, accent, offset=offset)
        if y + bh < h - 120:
            draw.text((stack_cx + 150, y + bh + 2), "+", fill=ARROW, font=font(True, 14), anchor="mm")
        y += bh + 14

    # Refresh loop arc beside stack
    arc_cx = stack_cx + 168
    arc_cy = stack_top + 120
    draw_refresh_arc(draw, arc_cx, arc_cy, 52, GREEN)
    draw.text((arc_cx + 58, arc_cy - 8), "refresh", fill=GREEN, font=font(True, 9), anchor="lt")

    # Dual-tag profile row
    prof_y = 430
    draw.rounded_rectangle((rx, prof_y, w - margin, prof_y + 72), radius=12, fill=(16, 22, 38), outline=TEAL, width=2)
    draw.text((rx + 16, prof_y + 12), "model_profile", fill=TEAL, font=font(True, 10, mono=True), anchor="lt")

    tag_specs = [
        ("inference", "qwen3.5:9b", TEAL, rx + 20),
        ("compose", "llama3.2:3b", AMBER, rx + 200),
        ("tools", "qwen fallback", PINK, rx + 360),
    ]
    for title, tag, accent, tx in tag_specs:
        draw.rounded_rectangle((tx, prof_y + 34, tx + 150, prof_y + 62), radius=8, fill=PANEL, outline=accent, width=1)
        draw.text((tx + 10, prof_y + 40), title, fill=accent, font=font(True, 8), anchor="lt")
        draw.text((tx + 10, prof_y + 52), tag, fill=MUTED, font=font(False, 8, mono=True), anchor="lt")

    # Growth curve
    draw_growth_dots(draw, rx + 20, prof_y - 24, GREEN)

    draw.text((w // 2, h - 14), "Prompt-time memory + weight-time compounding", fill=DIM, font=font(False, 10), anchor="ms")

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
    for i, (label, accent) in enumerate([
        ("base weights", MUTED),
        ("+ adapter v1", TEAL),
        ("+ delta v2", GREEN),
        ("→ composed tag", PINK),
    ]):
        bh = 44
        draw.rounded_rectangle((stack_x, stack_y, stack_x + 420, stack_y + bh), radius=10, fill=PANEL, outline=accent, width=2)
        draw.text((stack_x + 16, stack_y + 14), label, fill=accent, font=font(True, 14), anchor="lt")
        stack_y += bh + 10

    draw_refresh_arc(draw, stack_x + 460, margin + 280, 70, GREEN)

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
    assets = root / "assets"
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
