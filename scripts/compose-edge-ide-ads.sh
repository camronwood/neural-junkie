#!/usr/bin/env bash
# Compose "IDE for AI at the Edge" ads from REAL product screenshots (no fake UI mockups).
# Usage: ./scripts/compose-edge-ide-ads.sh [hero|square|story|agents|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VARIANT="${1:-all}"
PY="$ROOT/.venv-icon/bin/python"
MARKETING="$ROOT/campaigns/edge-ide/creatives"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT" "$MARKETING" "$VARIANT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont, ImageFilter

ROOT = Path(sys.argv[1])
OUT_DIR = Path(sys.argv[2])
VARIANT = sys.argv[3]

SHOTS = ROOT / "assets" / "screenshots"
WORKSPACE = next(SHOTS.glob("Screenshot 2026-05-29 at 2.31.27*"))
AGENTS = next(SHOTS.glob("Screenshot 2026-05-13 at 12.36.21*"))
EDITOR = next(SHOTS.glob("Screenshot 2026-05-13 at 12.36.44*"))

BG = (13, 13, 26)
BG2 = (10, 14, 28)
PANEL_EDGE = (83, 52, 131)
ACCENT = (233, 69, 96)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
TEAL = (72, 199, 142)
PINK = (199, 125, 255)
RELEASE = "github.com/camronwood/neural-junkie/releases"


def font(bold: bool, size: int, mono: bool = False):
    paths = ["/System/Library/Fonts/Menlo.ttc"] if mono else [
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold else "/System/Library/Fonts/Supplemental/Arial.ttf",
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


def gradient_bg(canvas, top=BG2, bottom=(8, 20, 32)):
    draw = ImageDraw.Draw(canvas)
    w, h = canvas.size
    for y in range(h):
        t = y / max(h - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (w, y)], fill=(r, g, b))


def paste_screenshot(canvas, shot_path: Path, box, border=3, radius=10):
    """Fit screenshot into box with purple border and subtle shadow (in-place)."""
    x0, y0, x1, y1 = box
    bw, bh = x1 - x0, y1 - y0
    img = Image.open(shot_path).convert("RGBA")

    scale = min((bw - 2 * border) / img.width, (bh - 2 * border) / img.height)
    nw, nh = int(img.width * scale), int(img.height * scale)
    img = img.resize((nw, nh), Image.Resampling.LANCZOS)

    px = x0 + (bw - nw - 2 * border) // 2
    py = y0 + (bh - nh - 2 * border) // 2

    # Drop shadow behind framed shot
    shadow = Image.new("RGBA", (nw + 16, nh + 16), (0, 0, 0, 0))
    sh_draw = ImageDraw.Draw(shadow)
    sh_draw.rounded_rectangle((8, 8, nw + 8, nh + 8), radius=radius, fill=(0, 0, 0, 140))
    shadow = shadow.filter(ImageFilter.GaussianBlur(6))
    base = canvas.convert("RGBA")
    base.paste(shadow, (px - 4, py - 2), shadow)

    framed = Image.new("RGBA", (nw + 2 * border, nh + 2 * border), (*PANEL_EDGE, 255))
    framed.paste(img, (border, border), img)
    base.paste(framed, (px, py), framed)
    canvas.paste(base.convert("RGB"))


def badge(draw, text: str, x: int, y: int, accent=TEAL):
    bf = font(True, 11)
    bw = int(draw.textlength(text, font=bf)) + 24
    draw.rounded_rectangle((x, y, x + bw, y + 28), radius=8, fill=(18, 24, 44), outline=accent, width=2)
    draw.text((x + bw // 2, y + 14), text, fill=accent, font=bf, anchor="mm")


def cta_bar(draw, canvas_w: int, y: int, label: str, sub: str = ""):
    cx = canvas_w // 2
    draw.rounded_rectangle((cx - 340, y, cx + 340, y + 48), radius=12, fill=ACCENT)
    draw.text((cx, y + 24), label, fill=TEXT, font=font(True, 18), anchor="mm")
    if sub:
        draw.text((cx, y + 68), sub, fill=MUTED, font=font(False, 14), anchor="mm")


def render_hero(out: Path):
    W, H = 1200, 627
    canvas = Image.new("RGB", (W, H), BG)
    gradient_bg(canvas)
    draw = ImageDraw.Draw(canvas)

    badge(draw, "NEURAL JUNKIE · OPEN BETA", 48, 36, TEAL)

    lx, lw = 48, 520
    hy = 88
    draw.text((lx, hy), "THE IDE FOR", fill=TEXT, font=font(True, 38), anchor="lt")
    draw.text((lx, hy + 44), "AI AT THE EDGE", fill=ACCENT, font=font(True, 42), anchor="lt")

    sy = hy + 100
    fsub = font(False, 16)
    for line in wrap(
        draw,
        "Real desktop workspace. Specialist agents. Local models via Ollama — hybrid when you want.",
        fsub,
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 22

    bullets = [
        "Files · editor · terminal · multi-agent chat",
        "Human approval on every file change",
        "macOS · Windows · Linux",
    ]
    for item in bullets:
        draw.text((lx + 4, sy + 8), "•", fill=TEAL, font=font(True, 16), anchor="lt")
        draw.text((lx + 22, sy + 8), item, fill=MUTED, font=font(False, 14), anchor="lt")
        sy += 26

    paste_screenshot(canvas, WORKSPACE, (580, 56, 1160, 540), border=4)

    draw.text((W // 2, H - 28), RELEASE, fill=MUTED, font=font(False, 13), anchor="mm")
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_square(out: Path):
    W, H = 1080, 1080
    canvas = Image.new("RGB", (W, H), BG)
    gradient_bg(canvas)
    draw = ImageDraw.Draw(canvas)

    cx = W // 2
    draw.text((cx, 44), "FOR DEVELOPERS", fill=MUTED, font=font(False, 12), anchor="mm")
    title = "THE IDE FOR AI AT THE EDGE"
    ft = font(True, 36)
    draw.text((cx, 78), title, fill=TEXT, font=ft, anchor="mm")
    tw = draw.textlength(title, font=ft)
    draw.rectangle((cx - tw // 2, 122, cx + tw // 2, 126), fill=ACCENT)

    draw.text(
        (cx, 148),
        "Multi-agent workspace · local-first · human-in-the-loop",
        fill=PINK,
        font=font(False, 15),
        anchor="mm",
    )

    paste_screenshot(canvas, WORKSPACE, (50, 180, 1030, 900), border=4)

    draw = ImageDraw.Draw(canvas)
    cta_bar(draw, W, 930, "DOWNLOAD BETA", RELEASE)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_story(out: Path):
    W, H = 1080, 1920
    canvas = Image.new("RGB", (W, H), BG)
    gradient_bg(canvas, (14, 12, 32), (8, 18, 28))
    draw = ImageDraw.Draw(canvas)

    cx = W // 2
    badge(draw, "NEURAL JUNKIE", cx - 70, 72, ACCENT)

    draw.text((cx, 140), "YOUR CODE.", fill=TEXT, font=font(True, 44), anchor="mm")
    draw.text((cx, 192), "YOUR MODELS.", fill=TEAL, font=font(True, 44), anchor="mm")
    draw.text((cx, 244), "YOUR TEAM.", fill=ACCENT, font=font(True, 44), anchor="mm")

    fsub = font(False, 18)
    sy = 300
    for line in wrap(
        draw,
        "The IDE for AI at the Edge — specialist agents on your machine, not someone else's cloud by default.",
        fsub,
        W - 80,
    ):
        draw.text((cx, sy), line, fill=MUTED, font=fsub, anchor="mm")
        sy += 26

    paste_screenshot(canvas, EDITOR, (40, 360, 1040, 1580), border=4)

    draw = ImageDraw.Draw(canvas)
    cta_bar(draw, W, 1620, "DOWNLOAD BETA", "macOS · Windows · Linux")
    draw.text((cx, 1760), RELEASE, fill=MUTED, font=font(False, 14), anchor="mm")
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_agents(out: Path):
    W, H = 1080, 1080
    canvas = Image.new("RGB", (W, H), BG)
    gradient_bg(canvas)
    draw = ImageDraw.Draw(canvas)

    cx = W // 2
    draw.text((cx, 48), "NOT ONE CHATBOT.", fill=TEXT, font=font(True, 34), anchor="mm")
    draw.text((cx, 92), "A WHOLE TEAM AT THE EDGE.", fill=ACCENT, font=font(True, 32), anchor="mm")
    draw.text(
        (cx, 132),
        "Backend · Security · Frontend · DevOps · Code Review · more",
        fill=MUTED,
        font=font(False, 14),
        anchor="mm",
    )

    paste_screenshot(canvas, AGENTS, (80, 170, 1000, 920), border=4)

    draw = ImageDraw.Draw(canvas)
    cta_bar(draw, W, 950, "MEET YOUR AGENTS", RELEASE)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


VARIANTS = {
    "hero": ("edge-ide-hero-banner.png", render_hero),
    "square": ("edge-ide-social-square.png", render_square),
    "story": ("edge-ide-story-vertical.png", render_story),
    "agents": ("edge-ide-agents-panel.png", render_agents),
}

OUT_DIR.mkdir(parents=True, exist_ok=True)

if VARIANT == "all":
    targets = VARIANTS.values()
else:
    if VARIANT not in VARIANTS:
        raise SystemExit(f"unknown variant {VARIANT!r}; use hero|square|story|agents|all")
    targets = [VARIANTS[VARIANT]]

for name, fn in targets:
    fn(OUT_DIR / name)
PY
