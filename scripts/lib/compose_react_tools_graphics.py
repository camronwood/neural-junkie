#!/usr/bin/env python3
"""Render ReAct tools marketing graphics (LinkedIn article cover)."""

from __future__ import annotations

import argparse
import shutil
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

BG_TOP = (14, 16, 32)
BG_BOTTOM = (8, 20, 28)
TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
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


def draw_tier(
    draw: ImageDraw.ImageDraw,
    x0: int,
    y0: int,
    x1: int,
    y1: int,
    title: str,
    subtitle: str,
    accent: tuple[int, int, int],
    highlight: bool = False,
) -> None:
    fill = (22, 30, 52) if highlight else (18, 24, 44)
    outline = accent if highlight else (40, 48, 72)
    width = 2 if highlight else 1
    draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=fill, outline=outline, width=width)
    draw.text((x0 + 16, y0 + 14), title, fill=accent if highlight else TEXT, font=font(True, 13), anchor="lt")
    draw.text((x0 + 16, y0 + 38), subtitle, fill=MUTED, font=font(False, 12), anchor="lt")


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas, step=32)
    draw = ImageDraw.Draw(canvas)

    lx = margin
    draw.rounded_rectangle((lx, 32, lx + 220, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 110, 45), "OPEN SOURCE · NEURAL JUNKIE", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "TOOLS WITHOUT", fill=TEXT, font=font(True, 36), anchor="lt")
    draw.text((lx, hy + 42), "NATIVE TOOL CALLING", fill=ACCENT, font=font(True, 34), anchor="lt")
    draw.text((lx, hy + 92), "ReAct wrapper · gemma3:12b · MCP loop", fill=AMBER, font=font(True, 18), anchor="lt")

    sy = hy + 132
    fsub = font(False, 15)
    for line in [
        "Strong chat models that lack Ollama tools capability",
        "can still run MCP loops on the same model.",
        "Qwen swap remains the reliability fallback.",
    ]:
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 24

    draw.rounded_rectangle((lx, 520, lx + 500, 572), radius=12, fill=TEAL)
    draw.text((lx + 250, 546), "github.com/camronwood/neural-junkie", fill=TEXT, font=font(True, 14), anchor="mm")

    rx = 560
    tier_w = w - margin - rx
    ty = 72
    th = 118
    gap = 16
    tiers = [
        ("NATIVE TOOLS", "qwen3.5:9b · tool_calls API", GREEN, False),
        ("REACT (same model)", "gemma3:12b · <tool_call> JSON", TEAL, True),
        ("QWEN FALLBACK", "swap on parse / cap failure", AMBER, False),
    ]
    for title, sub, accent, highlight in tiers:
        draw_tier(draw, rx, ty, rx + tier_w, ty + th, title, sub, accent, highlight=highlight)
        if ty + th + gap < h - 80:
            ax = rx + tier_w // 2
            draw.text((ax, ty + th + 2), "↓", fill=ARROW, font=font(True, 18), anchor="mt")
        ty += th + gap

    draw.text((w // 2, h - 16), "One model for chat and tools when native calling is unavailable", fill=DIM, font=font(False, 10), anchor="ms")

    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")
    if gallery:
        gallery.parent.mkdir(parents=True, exist_ok=True)
        canvas.save(gallery, "PNG")
        print(f"Wrote {gallery}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Render ReAct tools marketing graphics")
    parser.add_argument("variant", choices=["article"], help="which graphic to render")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    root = args.root
    assets = root / "campaigns" / "react-tools" / "creatives"
    assets.mkdir(parents=True, exist_ok=True)
    covers = root / "docs" / "media" / "articles" / "covers"
    gallery = root / "docs" / "media" / "gallery" / "ads"

    if args.variant == "article":
        asset_path = assets / "neural-junkie-react-tools-1200.png"
        render_article_cover(asset_path, gallery / "neural-junkie-react-tools-1200.png")
        covers.mkdir(parents=True, exist_ok=True)
        shutil.copy(asset_path, covers / "neural-junkie-react-tools-1200.png")
        print(f"Wrote {covers / 'neural-junkie-react-tools-1200.png'}")


if __name__ == "__main__":
    main()
