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


def draw_lane_panel(
    draw: ImageDraw.ImageDraw,
    x0: int,
    y0: int,
    x1: int,
    y1: int,
    title: str,
    accent: tuple[int, int, int],
) -> int:
    """Draw lane border; return inner content top."""
    draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=(14, 18, 34), outline=accent, width=2)
    tw = int(draw.textlength(title, font=font(True, 12)))
    badge_w = tw + 20
    bx0 = x0 + (x1 - x0 - badge_w) // 2
    draw.rounded_rectangle((bx0, y0 - 12, bx0 + badge_w, y0 + 12), radius=10, fill=(14, 18, 34), outline=accent, width=2)
    draw.text((bx0 + badge_w // 2, y0), title, fill=accent, font=font(True, 12), anchor="mm")
    return y0 + 24


def draw_agent_badge(draw: ImageDraw.ImageDraw, cx: int, cy: int, initials: str, name: str, accent: tuple[int, int, int]) -> None:
    r = 24
    draw.ellipse((cx - r, cy - r, cx + r, cy + r), fill=(22, 30, 52), outline=accent, width=2)
    draw.text((cx, cy), initials, fill=accent, font=font(True, 14), anchor="mm")
    nw = int(draw.textlength(name, font=font(False, 9)))
    draw.text((cx, cy + r + 12), name, fill=MUTED, font=font(False, 9), anchor="mm")


def draw_step_box(draw: ImageDraw.ImageDraw, cx: int, y0: int, text: str, accent: tuple[int, int, int], *, mono: bool = False) -> int:
    f = font(True, 9, mono=mono)
    tw = int(draw.textlength(text, font=f)) + 20
    bh = 24
    x0 = cx - tw // 2
    draw.rounded_rectangle((x0, y0, x0 + tw, y0 + bh), radius=8, fill=PANEL, outline=accent, width=1)
    draw.text((cx, y0 + bh // 2), text, fill=accent if mono else MUTED, font=f, anchor="mm")
    return y0 + bh


def draw_arrow_down(draw: ImageDraw.ImageDraw, cx: int, y0: int, y1: int, color=ARROW) -> None:
    if y1 <= y0 + 8:
        return
    draw.line([(cx, y0), (cx, y1 - 8)], fill=color, width=2)
    draw.polygon([(cx, y1), (cx - 5, y1 - 8), (cx + 5, y1 - 8)], fill=color)


def draw_deliverable_card(
    draw: ImageDraw.ImageDraw,
    cx: int,
    y0: int,
    bw: int,
    accent: tuple[int, int, int],
    bullets: list[str],
) -> int:
    bh = 78
    x0, x1 = cx - bw // 2, cx + bw // 2
    y1 = y0 + bh
    draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=(10, 14, 26), outline=accent, width=2)
    draw.rounded_rectangle((x0 + 10, y0 + 10, x0 + 26, y0 + 24), radius=3, fill=accent)
    draw.text((x0 + 32, y0 + 16), "findings.md", fill=accent, font=font(True, 10, mono=True), anchor="lt")
    ty = y0 + 34
    for line in bullets:
        draw.text((x0 + 12, ty), line, fill=(190, 198, 210), font=font(False, 9, mono=True), anchor="lt")
        ty += 14
    return y1


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    # ── Left column (match loop-stack proportions) ──
    lx, lw = margin, 500
    draw.rounded_rectangle((lx, 32, lx + 220, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 110, 45), "OPEN SOURCE · NEURAL JUNKIE", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "WHEN DOES", fill=TEXT, font=font(True, 38), anchor="lt")
    draw.text((lx, hy + 42), "MULTI-AGENT", fill=ACCENT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 90), "ACTUALLY HELP?", fill=GREEN, font=font(True, 34), anchor="lt")

    sy = hy + 138
    for line in wrap(
        draw,
        "Same fixture. Same deliverable. Solo DM vs structured collab — measured with a parity scenario.",
        font(False, 14),
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=font(False, 14), anchor="lt")
        sy += 20

    for glyph, text, accent in [
        ("▸", "Grounded in README + main.go", TEAL),
        ("▸", "No index.js hallucinations", ACCENT),
        ("▸", "Parity scenario, not vibes", PINK),
    ]:
        draw.text((lx, sy), glyph, fill=accent, font=font(True, 13), anchor="lt")
        draw.text((lx + 18, sy), text, fill=MUTED, font=font(False, 12), anchor="lt")
        sy += 22

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=PINK)
    draw.text((lx + lw // 2, 546), "github.com/camronwood/neural-junkie", fill=TEXT, font=font(True, 14), anchor="mm")
    draw.text((lx + lw // 2, h - 16), "Measure collaboration overhead — don't just demo it", fill=DIM, font=font(False, 10), anchor="ms")

    # ── Right column — structured parity diagram ──
    rx = 560
    rw = w - margin - rx
    panel_top, panel_bottom = 36, 580

    # Shared fixture bar
    fix_y0, fix_y1 = panel_top, panel_top + 72
    draw.rounded_rectangle((rx, fix_y0, rx + rw, fix_y1), radius=12, fill=PANEL, outline=TEAL, width=2)
    draw.text((rx + 16, fix_y0 + 14), "FIXTURE", fill=TEAL, font=font(True, 10), anchor="lt")
    draw.text((rx + 16, fix_y0 + 32), "minimal-repo", fill=TEXT, font=font(True, 12, mono=True), anchor="lt")
    tree_x = rx + rw // 2 + 20
    for i, (line, color) in enumerate([
        ("README.md", MUTED),
        ("core/sample/main.go", GREEN),
    ]):
        draw.text((tree_x, fix_y0 + 18 + i * 18), line, fill=color, font=font(False, 10, mono=True), anchor="lt")

    # Two equal lanes
    lane_gap = 16
    lane_w = (rw - lane_gap) // 2
    lane_y0 = fix_y1 + 20
    lane_y1 = panel_bottom - 56
    solo_x0, solo_x1 = rx, rx + lane_w
    collab_x0, collab_x1 = rx + lane_w + lane_gap, rx + rw
    solo_cx = (solo_x0 + solo_x1) // 2
    collab_cx = (collab_x0 + collab_x1) // 2

    solo_inner = draw_lane_panel(draw, solo_x0, lane_y0, solo_x1, lane_y1, "SOLO", TEAL)
    collab_inner = draw_lane_panel(draw, collab_x0, lane_y0, collab_x1, lane_y1, "COLLAB", AMBER)

    card_w = lane_w - 36

    # Solo lane content
    y = solo_inner + 16
    draw_agent_badge(draw, solo_cx, y + 24, "BE", "Backend", TEAL)
    y += 68
    draw_arrow_down(draw, solo_cx, y, y + 16, TEAL)
    y += 20
    y = draw_step_box(draw, solo_cx, y, "DM @Backend", TEAL, mono=True)
    draw_arrow_down(draw, solo_cx, y + 4, y + 24, TEAL)
    y += 28
    draw_deliverable_card(draw, solo_cx, y, card_w, TEAL, ["README scope", "main.go path", "minimal fixture"])

    # Collab lane content
    y = collab_inner + 12
    draw_agent_badge(draw, collab_cx - 38, y + 22, "SA", "Architect", AMBER)
    draw_agent_badge(draw, collab_cx + 38, y + 22, "BE", "Backend", TEAL)
    y += 62
    draw_arrow_down(draw, collab_cx, y, y + 12, AMBER)
    y += 16
    chip_y = y + 11
    chips = [("plan", PINK), ("approve", AMBER), ("execute", GREEN)]
    chip_gap = 8
    total_chip_w = sum(int(draw.textlength(c, font=font(True, 8))) + 14 for c, _ in chips) + chip_gap * (len(chips) - 1)
    chip_x = collab_cx - total_chip_w // 2
    for label, accent in chips:
        tw = int(draw.textlength(label, font=font(True, 8))) + 14
        draw.rounded_rectangle((chip_x, chip_y - 11, chip_x + tw, chip_y + 11), radius=10, fill=PANEL, outline=accent, width=1)
        draw.text((chip_x + tw // 2, chip_y), label, fill=accent, font=font(True, 8), anchor="mm")
        chip_x += tw + chip_gap
    y = chip_y + 20
    draw_arrow_down(draw, collab_cx, y, y + 16, AMBER)
    y += 20
    draw_deliverable_card(draw, collab_cx, y, card_w, AMBER, ["same task", "same assertions", "LLM judge"])

    # Shared assertions footer
    bar_y0 = panel_bottom - 44
    draw.rounded_rectangle((rx, bar_y0, rx + rw, panel_bottom), radius=12, fill=(18, 26, 38), outline=GREEN, width=2)
    draw.text((rx + rw // 2, bar_y0 + 14), "SAME ASSERTIONS", fill=GREEN, font=font(True, 10), anchor="mm")
    checks = "main.go ✓   ·   no index.js ✓   ·   ≥40 bytes ✓"
    draw.text((rx + rw // 2, bar_y0 + 30), checks, fill=MUTED, font=font(False, 10, mono=True), anchor="mm")

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
    assets = root / "campaigns" / "solo-vs-collab-parity" / "creatives"
    assets.mkdir(parents=True, exist_ok=True)
    covers = root / "docs" / "media" / "articles" / "covers"

    if args.variant == "article":
        out = assets / "neural-junkie-solo-vs-collab-parity-1200.png"
        gallery = covers / "neural-junkie-solo-vs-collab-parity-1200.png"
        render_article_cover(out, gallery)


if __name__ == "__main__":
    main()
