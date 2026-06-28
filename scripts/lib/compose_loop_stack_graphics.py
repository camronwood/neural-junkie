#!/usr/bin/env python3
"""Render loop-stack marketing graphics (LinkedIn article cover)."""

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


def stack_layer_metrics(
    draw: ImageDraw.ImageDraw,
    x0: int,
    x1: int,
    label: str,
    sub: str,
    indent: int = 0,
    *,
    compact: bool = False,
) -> tuple[int, list[str], list[str]]:
    """Return pixel height and wrapped label/sub lines for a stack row."""
    pad_x = 14
    pad_y = 6 if compact else 8
    label_gap = 3 if compact else 4
    sub_line_h = 11 if compact else 12
    label_line_h = 12 if compact else 13
    ix0 = x0 + indent
    inner_w = max(x1 - ix0 - 2 * pad_x, 72)
    label_f = font(True, 10 if not compact else 9)
    sub_f = font(False, 8 if not compact else 7)
    label_lines = wrap(draw, label, label_f, inner_w)
    sub_lines = wrap(draw, sub, sub_f, inner_w) if sub else []
    label_block_h = max(len(label_lines), 1) * label_line_h
    sub_block_h = len(sub_lines) * sub_line_h
    gap = label_gap if sub_lines else 0
    height = pad_y * 2 + label_block_h + gap + sub_block_h + 2
    return max(height, 32 if compact else 36), label_lines, sub_lines


def fit_stack_layout(
    draw: ImageDraw.ImageDraw,
    x0: int,
    x1: int,
    layers: list[tuple[str, str, tuple[int, int, int], int]],
    top: int,
    bottom: int,
) -> tuple[int, bool, list[tuple[int, list[str], list[str]]]]:
    """Choose gap/compact settings so the stack ends above bottom."""
    specs: list[tuple[int, list[str], list[str]]] = []
    for gap in (3, 2, 1):
        for compact in (False, True):
            total = top
            specs = []
            for label, sub, _accent, indent in layers:
                height, label_lines, sub_lines = stack_layer_metrics(
                    draw, x0, x1, label, sub, indent, compact=compact
                )
                specs.append((height, label_lines, sub_lines))
                total += height
            total += gap * (len(layers) - 1)
            if total <= bottom:
                return gap, compact, specs
    return 1, True, specs


def draw_stack_layer(
    draw: ImageDraw.ImageDraw,
    x0: int,
    y0: int,
    x1: int,
    height: int,
    label: str,
    sub: str,
    accent: tuple[int, int, int],
    indent: int = 0,
    *,
    compact: bool = False,
) -> None:
    pad_x = 14
    pad_y = 6 if compact else 8
    label_gap = 3 if compact else 4
    sub_line_h = 11 if compact else 12
    label_line_h = 12 if compact else 13
    ix0 = x0 + indent
    ix1 = x1
    y1 = y0 + height
    inner_w = max(ix1 - ix0 - 2 * pad_x, 72)

    draw.rounded_rectangle((ix0, y0, ix1, y1), radius=8, fill=PANEL, outline=accent, width=2)
    draw.rounded_rectangle((ix0 + 2, y0 + 4, ix0 + 6, y1 - 4), radius=2, fill=accent)

    label_f = font(True, 10 if not compact else 9)
    sub_f = font(False, 8 if not compact else 7)
    label_lines = wrap(draw, label, label_f, inner_w)
    sub_lines = wrap(draw, sub, sub_f, inner_w) if sub else []

    label_block_h = max(len(label_lines), 1) * label_line_h
    sub_block_h = len(sub_lines) * sub_line_h
    content_h = label_block_h + (label_gap if sub_lines else 0) + sub_block_h
    ty = y0 + max(pad_y, (height - content_h) // 2)

    for line in label_lines:
        draw.text((ix0 + pad_x, ty), line, fill=accent, font=label_f, anchor="lt")
        ty += label_line_h

    if sub_lines:
        ty += label_gap
        for line in sub_lines:
            draw.text((ix0 + pad_x, ty), line, fill=MUTED, font=sub_f, anchor="lt")
            ty += sub_line_h


def render_article_cover(out: Path, gallery: Path | None = None) -> None:
    w, h = 1200, 627
    margin = 48
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas, step=32)
    draw = ImageDraw.Draw(canvas)

    # Left column — headline
    lx = margin
    lw = 520
    draw.rounded_rectangle((lx, 32, lx + 200, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 100, 45), "OPEN SOURCE · NEURAL JUNKIE", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "WE DIDN'T BUILD", fill=TEXT, font=font(True, 38), anchor="lt")
    draw.text((lx, hy + 44), "ONE AGENT LOOP.", fill=ACCENT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 94), "We Built a Stack.", fill=AMBER, font=font(True, 20), anchor="lt")

    sy = hy + 132
    fsub = font(False, 15)
    for line in wrap(
        draw,
        "Implementation sessions, Fix Loop policy, repro verify, tool loops, collab phases — each loop closes a specific failure mode.",
        fsub,
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 22

    bullets = [
        "Command spam → circuit breaker + playbooks",
        "Silent exits → guaranteed session finale",
        "Wrong specialist → boot-fix routing",
    ]
    for item in bullets:
        draw.text((lx, sy), "▸", fill=TEAL, font=font(True, 14), anchor="lt")
        draw.text((lx + 18, sy), item, fill=MUTED, font=font(False, 13), anchor="lt")
        sy += 22

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=ACCENT)
    draw.text((lx + lw // 2, 546), "github.com/camronwood/neural-junkie", fill=TEXT, font=font(True, 14), anchor="mm")

    # Right column — stacked loop layers (dynamic height + wrapped subtext)
    rx = 560
    stack_x0 = rx
    stack_x1 = w - margin - 24
    stack_top = 52
    footer_top = h - 40  # reserved strip for bottom tagline

    layers = [
        ("COLLAB", "planning · execution · discussion", PINK, 0),
        ("IMPLEMENT", "discover · edit · verify · repair", TEAL, 6),
        ("RUNTIME V2", "open loop · Cursor parity", GREEN, 12),
        ("FIX LOOP", "telemetry · breaker · playbooks", ACCENT, 18),
        ("FIX-LIKE", "repro bootstrap · repro verify", AMBER, 24),
        ("BOOT-FIX", "grounding · routing · diagnostics", ACCENT, 30),
        ("VERIFY / REPAIR", "stack-aware · typed feedback", GREEN, 36),
        ("MULTI-FILE", "same turn · manifest targets", TEAL, 42),
        ("MCP TOOL", "native tools · model fallback", PINK, 48),
        ("DELEGATION", "cross-specialist consult", AMBER, 48),
        ("ANTI-LOOP", "no agent-to-agent spam", DIM, 54),
    ]

    draw.text((stack_x0, 36), "THE LOOP STACK", fill=AMBER, font=font(True, 12), anchor="lt")

    gap, compact, specs = fit_stack_layout(draw, stack_x0, stack_x1, layers, stack_top, footer_top)

    y = stack_top
    stack_mid = stack_top
    for (label, sub, accent, indent), (height, _, _) in zip(layers, specs):
        draw_stack_layer(draw, stack_x0, y, stack_x1, height, label, sub, accent, indent, compact=compact)
        y += height + gap
        stack_mid = (stack_top + y - gap) // 2

    # Flow arrow beside stack
    ax = stack_x1 + 10
    draw.text((ax, stack_mid - 36), "↓", fill=TEAL, font=font(True, 24), anchor="lt")
    draw.text((ax, stack_mid + 4), "compose", fill=DIM, font=font(False, 8), anchor="lt")

    # Bottom tagline — left column only so it never overlaps the stack
    draw.text(
        (lx + lw // 2, h - 16),
        "Platform loops beat bigger models for reliability",
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
    parser = argparse.ArgumentParser(description="Render loop-stack marketing graphics")
    parser.add_argument(
        "variant",
        choices=["article"],
        help="which graphic to render",
    )
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    root = args.root
    assets = root / "assets"
    gallery = root / "docs" / "media" / "gallery" / "ads"

    if args.variant == "article":
        render_article_cover(
            assets / "neural-junkie-loop-stack-1200.png",
            gallery / "neural-junkie-loop-stack-1200.png",
        )


if __name__ == "__main__":
    main()
