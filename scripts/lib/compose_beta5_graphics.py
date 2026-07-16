#!/usr/bin/env python3
"""Render v1.2.0-beta.5 release article cover (1200×627)."""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

# Brand palette (matches other Neural Junkie ads)
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


def draw_feature_tile(
    draw: ImageDraw.ImageDraw,
    x0: int,
    y0: int,
    x1: int,
    y1: int,
    title: str,
    sub: str,
    accent: tuple[int, int, int],
) -> None:
    pad = 12
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=PANEL, outline=accent, width=2)
    draw.rounded_rectangle((x0 + 3, y0 + 8, x0 + 7, y1 - 8), radius=2, fill=accent)
    inner_w = max(x1 - x0 - 2 * pad - 8, 40)
    title_f = font(True, 11)
    sub_f = font(False, 9)
    ty = y0 + pad
    for line in wrap(draw, title, title_f, inner_w):
        draw.text((x0 + pad + 4, ty), line, fill=accent, font=title_f, anchor="lt")
        ty += 14
    ty += 2
    for line in wrap(draw, sub, sub_f, inner_w):
        draw.text((x0 + pad + 4, ty), line, fill=MUTED, font=sub_f, anchor="lt")
        ty += 12


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas, step=32)
    draw = ImageDraw.Draw(canvas)

    lx = margin
    lw = 500

    # Version badge
    draw.rounded_rectangle((lx, 36, lx + 248, 64), radius=6, fill=(20, 28, 40), outline=AMBER, width=2)
    draw.text((lx + 124, 50), "v1.2.0-beta.5 · OPEN BETA", fill=AMBER, font=font(True, 11), anchor="mm")

    hy = 88
    draw.text((lx, hy), "THE RELEASE", fill=TEXT, font=font(True, 36), anchor="lt")
    draw.text((lx, hy + 42), "WHERE THE", fill=TEXT, font=font(True, 36), anchor="lt")
    draw.text((lx, hy + 84), "LOOPS CLOSE", fill=ACCENT, font=font(True, 40), anchor="lt")

    sy = hy + 140
    fsub = font(False, 15)
    for line in wrap(
        draw,
        "Runbooks you can replay. Routing you can audit. Collab hardened by live gates. Release engineering that keeps betas honest.",
        fsub,
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 22

    bullets = [
        ("ReAct tools on Gemma", TEAL),
        ("Multi-repo workspace scope", GREEN),
        ("LoRA v2 + personal learning", PINK),
    ]
    for text, color in bullets:
        draw.text((lx, sy), "▸", fill=color, font=font(True, 14), anchor="lt")
        draw.text((lx + 18, sy), text, fill=MUTED, font=font(False, 13), anchor="lt")
        sy += 22

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=ACCENT)
    draw.text((lx + lw // 2, 546), "github.com/camronwood/neural-junkie", fill=TEXT, font=font(True, 14), anchor="mm")

    # Right — feature grid (2×4)
    rx = 580
    grid_w = w - margin - rx
    col_w = (grid_w - 12) // 2
    row_h = 108
    features = [
        ("REACT TOOLS", "Gemma MCP · same brain", TEAL),
        ("ROUTING TRACE", "badges · telemetry drawer", GREEN),
        ("RUNBOOKS V2", "library · replay · connectors", AMBER),
        ("WORKSPACE SCOPE", "project sets · cross-repo", PINK),
        ("COLLAB HARDENING", "plan parser · watchdog", ACCENT),
        ("LORA V2", "compound specialists", PINK),
        ("SLACK DIAGNOSE", "setup checklist · smoke", TEAL),
        ("LAYER GATES", "fix-loop · test-growth", GREEN),
    ]

    draw.text((rx, 40), "SHIPPING THIS WEEK", fill=AMBER, font=font(True, 12), anchor="lt")

    y = 68
    for i, (title, sub, accent) in enumerate(features):
        col = i % 2
        row = i // 2
        x0 = rx + col * (col_w + 12)
        y0 = y + row * (row_h + 10)
        draw_feature_tile(draw, x0, y0, x0 + col_w, y0 + row_h, title, sub, accent)

    draw.text(
        (rx + grid_w // 2, h - 16),
        "Local-first multi-agent · MIT licensed",
        fill=DIM,
        font=font(False, 10),
        anchor="ms",
    )

    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")
    if gallery:
        gallery.parent.mkdir(parents=True, exist_ok=True)
        canvas.save(gallery, "PNG")
        print(f"Wrote {gallery}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Render beta.5 release article cover")
    parser.add_argument(
        "variant",
        choices=["article"],
        help="which graphic to render",
    )
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    root = args.root
    assets = root / "campaigns" / "beta25" / "creatives"
    assets.mkdir(parents=True, exist_ok=True)
    gallery = root / "docs" / "media" / "gallery" / "ads"

    if args.variant == "article":
        render_article_cover(
            assets / "neural-junkie-beta5-1200.png",
            gallery / "neural-junkie-beta5-1200.png",
        )


if __name__ == "__main__":
    main()
