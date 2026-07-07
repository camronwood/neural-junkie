#!/usr/bin/env python3
"""Render solo-vs-collab parity marketing graphics (LinkedIn article cover)."""

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


def draw_agent_node(
    draw: ImageDraw.ImageDraw,
    cx: int,
    cy: int,
    label: str,
    accent: tuple[int, int, int],
    *,
    r: int = 28,
    initials: str | None = None,
) -> None:
    badge = initials or label[:2].upper()
    draw.ellipse((cx - r, cy - r, cx + r, cy + r), fill=(22, 30, 52), outline=accent, width=3)
    draw.text((cx, cy - 6), badge, fill=accent, font=font(True, 16), anchor="mm")
    draw.text((cx, cy + r + 14), label, fill=MUTED, font=font(False, 9), anchor="mm")


def draw_arrow_down(draw: ImageDraw.ImageDraw, cx: int, y0: int, y1: int, color=ARROW) -> None:
    draw.line([(cx, y0), (cx, y1 - 8)], fill=color, width=2)
    draw.polygon([(cx, y1), (cx - 6, y1 - 10), (cx + 6, y1 - 10)], fill=color)


def draw_file_card(
    draw: ImageDraw.ImageDraw,
    x0: int,
    y0: int,
    x1: int,
    y1: int,
    title: str,
    lines: list[str],
    accent: tuple[int, int, int],
) -> None:
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=(12, 16, 28), outline=accent, width=2)
    draw.rounded_rectangle((x0 + 8, y0 + 8, x0 + 28, y0 + 22), radius=4, fill=accent)
    draw.text((x0 + 36, y0 + 14), title, fill=accent, font=font(True, 10, mono=True), anchor="lt")
    ty = y0 + 32
    for line in lines:
        draw.text((x0 + 12, ty), line, fill=(190, 198, 210), font=font(False, 9, mono=True), anchor="lt")
        ty += 14


def draw_phase_chip(draw: ImageDraw.ImageDraw, cx: int, cy: int, text: str, accent: tuple[int, int, int]) -> int:
    f = font(True, 9)
    tw = int(draw.textlength(text, font=f)) + 16
    x0, y0 = cx - tw // 2, cy - 11
    draw.rounded_rectangle((x0, y0, x0 + tw, y0 + 22), radius=11, fill=(20, 28, 44), outline=accent, width=1)
    draw.text((cx, cy), text, fill=accent, font=f, anchor="mm")
    return tw


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    # Left column — headline
    lx = margin
    lw = 500
    draw.rounded_rectangle((lx, 32, lx + 230, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 115, 45), "OPEN SOURCE · NEURAL JUNKIE", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "WHEN DOES", fill=TEXT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 46), "MULTI-AGENT", fill=ACCENT, font=font(True, 42), anchor="lt")
    draw.text((lx, hy + 96), "ACTUALLY HELP?", fill=GREEN, font=font(True, 36), anchor="lt")

    sy = hy + 148
    fsub = font(False, 14)
    for line in wrap(
        draw,
        "Same minimal-repo fixture. Same findings.md deliverable. Solo DM vs structured collab — measured, not demoed.",
        fsub,
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 20

    bullets = [
        ("▸", "Grounded in README + main.go", TEAL),
        ("▸", "No index.js hallucinations", ACCENT),
        ("▸", "Parity scenario, not vibes", PINK),
    ]
    for glyph, text, accent in bullets:
        draw.text((lx, sy), glyph, fill=accent, font=font(True, 13), anchor="lt")
        draw.text((lx + 18, sy), text, fill=MUTED, font=font(False, 12), anchor="lt")
        sy += 22

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=PINK)
    draw.text((lx + lw // 2, 546), "solo-vs-collab-parity.json", fill=TEXT, font=font(True, 13, mono=True), anchor="mm")

    # Right column — split parity diagram
    rx = 580
    rw = w - margin - rx
    mid_x = rx + rw // 2

    # Repo fixture panel (top)
    repo_x0, repo_y0 = rx, 44
    repo_x1, repo_y1 = rx + rw, 130
    draw.rounded_rectangle((repo_x0, repo_y0, repo_x1, repo_y1), radius=12, fill=PANEL, outline=TEAL, width=2)
    draw.text((repo_x0 + 14, repo_y0 + 12), "minimal-repo", fill=TEAL, font=font(True, 11, mono=True), anchor="lt")
    tree_lines = [
        "README.md",
        "core/",
        "  sample/main.go",
    ]
    ty = repo_y0 + 34
    for i, line in enumerate(tree_lines):
        color = GREEN if "main.go" in line else MUTED
        draw.text((repo_x0 + 20 + (12 if i > 0 else 0), ty), line, fill=color, font=font(False, 10, mono=True), anchor="lt")
        ty += 16

    # Vertical divider with PARITY badge
    div_y0, div_y1 = 148, 500
    draw.line([(mid_x, div_y0), (mid_x, div_y1)], fill=(50, 58, 82), width=2)
    draw.rounded_rectangle((mid_x - 42, 250, mid_x + 42, 286), radius=14, fill=(24, 20, 40), outline=PINK, width=2)
    draw.text((mid_x, 268), "PARITY", fill=PINK, font=font(True, 12), anchor="mm")

    # Solo column (left half)
    solo_cx = rx + rw // 4
    draw.text((solo_cx, 152), "SOLO", fill=TEAL, font=font(True, 13), anchor="mm")
    draw_agent_node(draw, solo_cx, 210, "BackendEngineer", TEAL, initials="BE")
    draw_arrow_down(draw, solo_cx, 248, 290, TEAL)
    draw.rounded_rectangle((solo_cx - 70, 292, solo_cx + 70, 318), radius=8, fill=(16, 22, 38), outline=TEAL, width=1)
    draw.text((solo_cx, 305), "DM @BackendEngineer", fill=TEAL, font=font(False, 8, mono=True), anchor="mm")
    draw_arrow_down(draw, solo_cx, 322, 360, TEAL)
    draw_file_card(
        draw,
        solo_cx - 88,
        364,
        solo_cx + 88,
        448,
        "findings.md",
        ["• README scope", "• main.go path", "• minimal fixture"],
        TEAL,
    )

    # Collab column (right half)
    collab_cx = rx + 3 * rw // 4
    draw.text((collab_cx, 152), "COLLAB", fill=AMBER, font=font(True, 13), anchor="mm")
    draw_agent_node(draw, collab_cx - 36, 200, "SoftwareArchitect", AMBER, r=22, initials="SA")
    draw_agent_node(draw, collab_cx + 36, 200, "BackendEngineer", TEAL, r=22, initials="BE")
    draw.line([(collab_cx - 36, 200), (collab_cx + 36, 200)], fill=ARROW, width=1)

    chip_y = 248
    chips = [("plan", PINK), ("approve", AMBER), ("execute", GREEN)]
    chip_x = collab_cx - 80
    for i, (label, accent) in enumerate(chips):
        tw = draw_phase_chip(draw, chip_x + i * 56, chip_y, label, accent)
        if i < len(chips) - 1:
            draw.line([(chip_x + i * 56 + tw // 2 + 4, chip_y), (chip_x + (i + 1) * 56 - tw // 2 - 4, chip_y)], fill=ARROW, width=1)

    draw_arrow_down(draw, collab_cx, 268, 360, AMBER)
    draw_file_card(
        draw,
        collab_cx - 88,
        364,
        collab_cx + 88,
        448,
        "findings.md",
        ["• same task", "• same assertions", "• LLM judge"],
        AMBER,
    )

    # Shared assertions bar
    bar_y0, bar_y1 = 462, 508
    draw.rounded_rectangle((rx, bar_y0, rx + rw, bar_y1), radius=12, fill=(20, 28, 36), outline=GREEN, width=2)
    draw.text((mid_x, bar_y0 + 16), "SAME ASSERTIONS", fill=GREEN, font=font(True, 11), anchor="mm")
    checks = ["main.go ✓", "no index.js ✓", "≥40 bytes ✓"]
    cx = rx + 40
    for check in checks:
        draw.text((cx, bar_y0 + 34), check, fill=MUTED, font=font(False, 10, mono=True), anchor="lt")
        cx += int(draw.textlength(check, font=font(False, 10, mono=True))) + 28

    draw.text((w // 2, h - 14), "Measure collaboration overhead — don't just demo it", fill=DIM, font=font(False, 10), anchor="ms")

    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")
    if gallery:
        gallery.parent.mkdir(parents=True, exist_ok=True)
        canvas.save(gallery, "PNG")
        print(f"Wrote {gallery}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Render solo-vs-collab parity marketing graphics")
    parser.add_argument("variant", choices=["article"], help="which graphic to render")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    root = args.root
    assets = root / "assets"
    covers = root / "docs" / "media" / "articles" / "covers"

    if args.variant == "article":
        out = assets / "neural-junkie-solo-vs-collab-parity-1200.png"
        gallery = covers / "neural-junkie-solo-vs-collab-parity-1200.png"
        render_article_cover(out, gallery)


if __name__ == "__main__":
    main()
