#!/usr/bin/env python3
"""Render inference-layer marketing graphics (square ads + LinkedIn article cover)."""

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


def render_skip_ad(out: Path) -> None:
    w, h = 1080, 1080
    margin = 56
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    top_badge(draw, w, "INFERENCE LAYER · AVOID", TEAL)
    y = 100
    draw.text((w // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
    y += 40
    y = headline_block(draw, w, y, [('"THANKS!"', TEXT), ("= ZERO TOKENS", TEAL)], size=48)
    y = subhead(draw, w, y + 8, "Classify intent before the GPU wakes up. Closure → canned reply. No 14B. No scan.")

    # Intent table
    table_y = y + 36
    rows = [
        ("closure", '"Thanks!"', "canned reply", "0 tokens", TEAL),
        ("casual", '"Hey"', "minimal prompt", "small model", GREEN),
        ("substantive", "refactor task", "full specialist", "14B / LoRA", AMBER),
        ("task", "fix + paths", "scan + tools", "full stack", PINK),
    ]
    row_h = 72
    gap = 10
    headers = ["INTENT", "EXAMPLE", "ACTION", "COST"]
    col_w = (w - 2 * margin - 3 * 12) // 4
    hx = margin
    for i, hdr in enumerate(headers):
        cw = col_w if i > 0 else 120
        draw.text((hx + cw // 2, table_y), hdr, fill=DIM, font=font(True, 10), anchor="mm")
        hx += cw + 12

    for row_i, (intent, example, action, cost, accent) in enumerate(rows):
        ry = table_y + 24 + row_i * (row_h + gap)
        draw.rounded_rectangle((margin, ry, w - margin, ry + row_h), radius=12, fill=PANEL, outline=accent, width=2)
        cols = [intent.upper(), example, action, cost]
        hx = margin + 12
        widths = [120, col_w, col_w, col_w]
        for col_i, (cell, cw) in enumerate(zip(cols, widths)):
            color = accent if col_i == 0 else TEXT if col_i < 3 else GREEN if "0" in cell else MUTED
            fnt = font(True, 12 if col_i == 0 else 11, mono=col_i > 0)
            draw.text((hx + 12, ry + row_h // 2), cell, fill=color, font=fnt, anchor="lm")
            hx += cw + 12

    strip_y = table_y + 24 + len(rows) * (row_h + gap) + 28
    draw.rounded_rectangle((margin, strip_y, w - margin, strip_y + 100), radius=14, fill=PANEL_SOFT, outline=TEAL, width=2)
    draw.text((margin + 20, strip_y + 16), "Context stack before every LLM call", fill=TEAL, font=font(True, 14), anchor="lt")
    bullets = [
        "Mode · intent · memory · grounding · persona · budget",
        "Delegation skipped for closure and casual intents",
        "Session summaries on 9B — not 14B every turn",
    ]
    by = strip_y + 42
    for item in bullets:
        draw.text((margin + 28, by), "✓", fill=GREEN, font=font(True, 13), anchor="lt")
        draw.text((margin + 48, by), item, fill=(210, 214, 224), font=font(False, 13), anchor="lt")
        by += 22

    cta_footer(draw, w, h, margin, "the fastest inference is the one you don't run")
    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_gate_ad(out: Path) -> None:
    w, h = 1080, 1080
    margin = 56
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    top_badge(draw, w, "INFERENCE LAYER · GATE", AMBER)
    y = 100
    draw.text((w // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
    y += 40
    y = headline_block(draw, w, y, [("DECIDE BEFORE", TEXT), ("YOU GENERATE", AMBER)], size=48)
    y = subhead(draw, w, y + 4, "Three questions before tokens get spent: Should we infer? Which model? Which provider?")

    # Vertical pipeline
    cx = w // 2
    stages = [
        ("USER MESSAGE", "", MUTED, 220, 56),
        ("CONTEXT STACK", "mode · intent · budget", TEAL, 240, 68),
        ("SKIP?", "closure → zero tokens", GREEN, 200, 60),
        ("9B ROUTER", "domain · cost · tools", AMBER, 220, 68),
        ("MODEL + PROVIDER", "Ollama · HF · cloud", PINK, 260, 68),
        ("REPLY + BADGE", "see which model ran", ACCENT, 240, 60),
    ]
    cy = y + 48
    gap = 28
    prev_bottom = None
    for title, sub, accent, bw, bh in stages:
        x0, y0, x1, y1 = flow_box(draw, cx, cy + bh // 2, bw, bh, title, sub, accent)
        if prev_bottom is not None:
            draw.line([(cx, prev_bottom + 4), (cx, y0 - 4)], fill=accent, width=2)
            draw.polygon([(cx, y0 - 4), (cx - 5, y0 - 12), (cx + 5, y0 - 12)], fill=accent)
        prev_bottom = y1
        cy += bh + gap

    # Branch note for SKIP
    branch_x = cx + 140
    skip_y = y + 48 + 56 + 28 + 68 + 28 + 30
    draw.rounded_rectangle((branch_x, skip_y - 24, w - margin, skip_y + 24), radius=10, fill=(14, 32, 24), outline=GREEN, width=2)
    draw.text((branch_x + 16, skip_y), "YES → canned reply", fill=GREEN, font=font(True, 12), anchor="lm")

    cta_footer(draw, w, h, margin, "open source · macOS · Windows · Linux")
    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_trust_ad(out: Path) -> None:
    w, h = 1080, 1080
    margin = 56
    canvas = Image.new("RGB", (w, h), BG_TOP)
    gradient_bg(canvas)
    canvas = apply_dot_grid(canvas)
    draw = ImageDraw.Draw(canvas)

    top_badge(draw, w, "INFERENCE LAYER · TRUST", PINK)
    y = 100
    draw.text((w // 2, y), "NEURAL JUNKIE", fill=PINK, font=font(True, 24), anchor="mm")
    y += 40
    y = headline_block(draw, w, y, [("WHICH MODEL", TEXT), ("ACTUALLY RAN?", PINK)], size=48)
    y = subhead(draw, w, y + 4, "Routing badge on every reply — chat, tool, reason, classifier source.")

    panel_x0, panel_y0 = margin, y + 24
    panel_x1, panel_y1 = w - margin, panel_y0 + 400
    draw.rounded_rectangle((panel_x0, panel_y0, panel_x1, panel_y1), radius=18, fill=(12, 18, 34), outline=(60, 68, 96), width=2)
    draw.text((panel_x0 + 24, panel_y0 + 20), "BiologyExpert", fill=TEAL, font=font(True, 14), anchor="lt")
    draw.text((panel_x0 + 24, panel_y0 + 42), "Agent", fill=DIM, font=font(False, 11), anchor="lt")

    bubble_x0 = panel_x0 + 24
    bubble_y0 = panel_y0 + 72
    bubble_x1 = panel_x1 - 80
    bubble_y1 = bubble_y0 + 100
    draw.rounded_rectangle((bubble_x0, bubble_y0, bubble_x1, bubble_y1), radius=14, fill=PANEL, outline=(70, 78, 108), width=1)
    msg = "Peptide analyzed and folded. PDB saved under ~/.neural-junkie/bio/"
    mf = font(False, 15)
    my = bubble_y0 + 20
    for line in wrap(draw, msg, mf, bubble_x1 - bubble_x0 - 32):
        draw.text((bubble_x0 + 18, my), line, fill=(220, 224, 232), font=mf, anchor="lt")
        my += 22

    badge_y = bubble_y1 + 20
    line1 = "routing · chat: koesn/openbiollm-8b · tool: qwen3.5:9b"
    line2 = "source: llm · reason: biology_tools"
    bf = font(False, 11, mono=True)
    chip_w = panel_x1 - panel_x0 - 48
    chip_h = 48
    draw.rounded_rectangle((bubble_x0, badge_y, bubble_x0 + chip_w, badge_y + chip_h), radius=8, fill=(28, 22, 48), outline=PINK, width=2)
    draw.text((bubble_x0 + 14, badge_y + 12), line1, fill=PINK, font=bf, anchor="lm")
    draw.text((bubble_x0 + 14, badge_y + 30), line2, fill=PINK, font=bf, anchor="lm")

    preview_y = badge_y + 58
    draw.text((panel_x0 + 24, preview_y), "Split inference — one answer, two models:", fill=MUTED, font=font(True, 12), anchor="lt")
    meta_lines = [
        "OpenBio 8B → domain reasoning",
        "qwen3.5:9b → MCP tool loop (analyze + fold)",
        "User sees one BiologyExpert reply",
    ]
    mf2 = font(False, 12)
    ly = preview_y + 26
    for line in meta_lines:
        draw.text((panel_x0 + 32, ly), "▸", fill=GREEN, font=font(True, 12), anchor="lt")
        draw.text((panel_x0 + 50, ly), line, fill=(210, 214, 224), font=mf2, anchor="lt")
        ly += 22

    draw.text((panel_x0 + 24, panel_y1 - 36), "Toggle: Settings → Layout → Routing badges on messages", fill=DIM, font=font(False, 12), anchor="lt")

    cta_footer(draw, w, h, margin, "modular AI only feels trustworthy when routing is legible")
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

    lx = margin
    lw = 520
    draw.rounded_rectangle((lx, 32, lx + 200, 58), radius=6, fill=(20, 28, 40), outline=TEAL, width=1)
    draw.text((lx + 100, 45), "OPEN SOURCE · NEURAL JUNKIE", fill=TEAL, font=font(True, 10), anchor="mm")

    hy = 78
    draw.text((lx, hy), "DECIDE BEFORE", fill=TEXT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 46), "YOU GENERATE", fill=ACCENT, font=font(True, 44), anchor="lt")
    draw.text((lx, hy + 100), "The Inference Layer", fill=AMBER, font=font(True, 18), anchor="lt")

    sy = hy + 138
    fsub = font(False, 15)
    for line in wrap(
        draw,
        "Should we infer? Which model? Which provider? Context stack → router → badge on every reply.",
        fsub,
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 22

    bullets = [
        '"Thanks!" → zero tokens (closure intent)',
        "9B router + rules fallback per job",
        "OpenBio chat + Qwen tool loop split",
    ]
    for item in bullets:
        draw.text((lx, sy), "▸", fill=TEAL, font=font(True, 14), anchor="lt")
        draw.text((lx + 18, sy), item, fill=MUTED, font=font(False, 13), anchor="lt")
        sy += 22

    draw.rounded_rectangle((lx, 520, lx + lw, 572), radius=12, fill=ACCENT)
    draw.text((lx + lw // 2, 546), RELEASE, fill=TEXT, font=font(True, 15), anchor="mm")

    # Right column — pipeline diagram
    rx = 600
    rw = w - rx - margin
    flow_y = 72
    steps = [
        ("MESSAGE", MUTED, 90),
        ("CONTEXT", TEAL, 100),
        ("SKIP?", GREEN, 80),
        ("ROUTER", AMBER, 100),
        ("MODEL", PINK, 90),
        ("BADGE", ACCENT, 90),
    ]
    cx = rx + 10
    prev_right = None
    for title, accent, bw in steps:
        x0, y0 = cx, flow_y - 24
        x1, y1 = cx + bw, flow_y + 24
        draw.rounded_rectangle((x0, y0, x1, y1), radius=10, fill=PANEL, outline=accent, width=2)
        draw.text((cx + bw // 2, flow_y), title, fill=accent, font=font(True, 9), anchor="mm")
        if prev_right is not None:
            draw_arrow(draw, prev_right + 4, flow_y, x0 - 4)
        prev_right = x1
        cx += bw + 16

    # Three paths panel
    py = flow_y + 72
    paths = [
        ("ZERO", '"Thanks!" → 0 tokens', TEAL),
        ("CHAT", "OpenBio reasoning only", GREEN),
        ("SPLIT", "OpenBio + Qwen tools", PINK),
    ]
    pw = (rw - 24) // 3
    for i, (title, sub, color) in enumerate(paths):
        px0 = rx + i * (pw + 12)
        draw.rounded_rectangle((px0, py, px0 + pw, py + 96), radius=12, fill=PANEL_SOFT, outline=color, width=2)
        draw.text((px0 + pw // 2, py + 26), title, fill=color, font=font(True, 12), anchor="mm")
        for j, line in enumerate(wrap(draw, sub, font(False, 10), pw - 16)):
            draw.text((px0 + pw // 2, py + 50 + j * 16), line, fill=MUTED, font=font(False, 10), anchor="mm")

    # Questions strip
    stack_y = py + 120
    draw.rounded_rectangle((rx, stack_y, w - margin, stack_y + 120), radius=14, fill=(10, 16, 28), outline=AMBER, width=2)
    draw.text((rx + 16, stack_y + 14), "Three questions before tokens get spent", fill=AMBER, font=font(True, 12), anchor="lt")
    qs = [
        "1. Should we infer at all?",
        "2. Which model tag when we do?",
        "3. Which provider + tool loop?",
    ]
    qy = stack_y + 40
    for q in qs:
        draw.text((rx + 24, qy), q, fill=(210, 214, 224), font=font(False, 13), anchor="lt")
        qy += 22

    out.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")
    if gallery:
        gallery.parent.mkdir(parents=True, exist_ok=True)
        canvas.save(gallery, "PNG")
        print(f"Wrote {gallery}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Render inference-layer marketing graphics")
    parser.add_argument(
        "variant",
        choices=["skip", "gate", "trust", "article", "all"],
        help="which graphic to render",
    )
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    root = args.root
    assets = root / "assets"
    gallery = root / "docs" / "media" / "gallery" / "ads"

    renders = {
        "skip": (render_skip_ad, assets / "neural-junkie-inference-skip-ad-1080.png"),
        "gate": (render_gate_ad, assets / "neural-junkie-inference-gate-ad-1080.png"),
        "trust": (render_trust_ad, assets / "neural-junkie-inference-trust-ad-1080.png"),
    }

    if args.variant == "article":
        render_article_cover(
            assets / "neural-junkie-inference-layer-1200.png",
            gallery / "neural-junkie-inference-layer-1200.png",
        )
        return

    if args.variant == "all":
        for fn, path in renders.values():
            fn(path)
        render_article_cover(
            assets / "neural-junkie-inference-layer-1200.png",
            gallery / "neural-junkie-inference-layer-1200.png",
        )
        return

    fn, path = renders[args.variant]
    fn(path)


if __name__ == "__main__":
    main()
