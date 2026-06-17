#!/usr/bin/env bash
# Compose IDE v4 "open source / build with us" ads from product screenshots.
# Usage: ./scripts/compose-ide-v4-ads.sh [hero|square|square-elon|story|carousel|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VARIANT="${1:-all}"
PY="$ROOT/.venv-icon/bin/python"
MARKETING="$ROOT/assets/marketing"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT" "$MARKETING" "$VARIANT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont, ImageFilter

ROOT = Path(sys.argv[1])
OUT = Path(sys.argv[2])
VARIANT = sys.argv[3]
SHOTS = ROOT / "assets" / "screenshots"


def pick(*patterns: str):
    for pat in patterns:
        hits = sorted(SHOTS.glob(pat))
        if hits:
            return hits[0]
    return None


WORKSPACE = pick("Screenshot 2026-05-29 at 2.31.27*", "Screenshot*.png", "Screenshot*.jpg")
AGENTS = pick("Screenshot 2026-05-13 at 12.36.21*", "Screenshot*.png")
EDITOR = pick("Screenshot 2026-05-13 at 12.36.44*", "Screenshot*.png")
CHAT = pick("Screenshot 2026-05-13 at 12.35.40*", "Screenshot 2026-05-13 at 12.36.02*")

BG = (13, 13, 26)
BG2 = (10, 14, 28)
PANEL_EDGE = (83, 52, 131)
ACCENT = (233, 69, 96)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
TEAL = (72, 199, 142)
PINK = (199, 125, 255)
REPO = "github.com/camronwood/neural-junkie"
RELEASE = f"{REPO}/releases/latest"


def font(bold=False, size=32, mono=False):
    paths = ["/System/Library/Fonts/Menlo.ttc"] if mono else [
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold else "/System/Library/Fonts/Supplemental/Arial.ttf",
    ]
    for path in paths:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            pass
    return ImageFont.load_default()


def wrap(draw, text, fnt, max_w):
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


def paste_screenshot(canvas, shot_path, box, border=3, radius=10):
    if shot_path is None:
        return
    x0, y0, x1, y1 = box
    bw, bh = x1 - x0, y1 - y0
    img = Image.open(shot_path).convert("RGBA")
    scale = min((bw - 2 * border) / img.width, (bh - 2 * border) / img.height)
    nw, nh = int(img.width * scale), int(img.height * scale)
    img = img.resize((nw, nh), Image.Resampling.LANCZOS)
    px = x0 + (bw - nw - 2 * border) // 2
    py = y0 + (bh - nh - 2 * border) // 2

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


def badge(draw, text, x, y, accent=TEAL):
    bf = font(True, 11)
    bw = int(draw.textlength(text, font=bf)) + 24
    draw.rounded_rectangle((x, y, x + bw, y + 28), radius=8, fill=(18, 24, 44), outline=accent, width=2)
    draw.text((x + bw // 2, y + 14), text, fill=accent, font=bf, anchor="mm")


def badge_centered(draw, text, cx, y, accent=TEAL):
    bf = font(True, 11)
    bw = int(draw.textlength(text, font=bf)) + 24
    x = cx - bw // 2
    badge(draw, text, x, y, accent)


def cta_bar(draw, canvas_w, y, label, sub=""):
    cx = canvas_w // 2
    draw.rounded_rectangle((cx - 340, y, cx + 340, y + 48), radius=12, fill=ACCENT)
    draw.text((cx, y + 24), label, fill=TEXT, font=font(True, 18), anchor="mm")
    if sub:
        draw.text((cx, y + 68), sub, fill=MUTED, font=font(False, 14), anchor="mm")


def bullets(draw, x, y, items, max_w, size=15):
    sy = y
    for item in items:
        for line in wrap(draw, item, font(False, size), max_w - 22):
            draw.text((x + 4, sy), "•", fill=TEAL, font=font(True, size), anchor="lt")
            draw.text((x + 22, sy), line, fill=MUTED, font=font(False, size), anchor="lt")
            sy += size + 8
        sy += 4
    return sy


def render_hero():
    w, h = 1200, 627
    canvas = Image.new("RGB", (w, h), BG)
    gradient_bg(canvas)
    draw = ImageDraw.Draw(canvas)

    badge(draw, "NEURAL JUNKIE · IDE v4 · OPEN SOURCE", 48, 36, TEAL)
    lx, lw = 48, 520
    hy = 88
    draw.text((lx, hy), "IF THEY BOUGHT", fill=ACCENT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 46), "YOUR IDE", fill=ACCENT, font=font(True, 40), anchor="lt")
    draw.text((lx, hy + 100), "BUILD THE NEXT ONE", fill=TEXT, font=font(True, 34), anchor="lt")
    draw.text((lx, hy + 142), "WITH US", fill=TEXT, font=font(True, 34), anchor="lt")

    sy = hy + 196
    for line in wrap(
        draw,
        "Full LSP · remote SSH · dev containers · multi-agent workspace — local-first, you approve every edit.",
        font(False, 15),
        lw,
    ):
        draw.text((lx, sy), line, fill=(200, 205, 220), font=font(False, 15), anchor="lt")
        sy += 22

    bullets(draw, lx, sy + 4, [
        "gopls · rust-analyzer · pyright via Monaco LSP",
        "nj-remote sidecar on EC2 or your dev box",
        "Bring your own model — Ollama, Claude, GPT",
    ], lw, size=14)

    paste_screenshot(canvas, WORKSPACE or AGENTS, (580, 56, 1160, 540), border=4)
    draw = ImageDraw.Draw(canvas)
    draw.text((w // 2, h - 28), RELEASE, fill=MUTED, font=font(False, 13), anchor="mm")
    out = OUT / "ide-v4-hero-banner.png"
    canvas.save(out)
    print(f"Wrote {out}")


def render_square(out_name="ide-v4-social-square.png", hook_lines=None, subline="BUILD THE NEXT ONE WITH US"):
    w, h = 1080, 1080
    canvas = Image.new("RGB", (w, h), BG)
    gradient_bg(canvas)
    draw = ImageDraw.Draw(canvas)
    cx = w // 2

    if hook_lines is None:
        hook_lines = ["IF THEY BOUGHT YOUR IDE"]

    badge_centered(draw, "OPEN SOURCE", cx, 44, TEAL)

    hy = 44 + 28 + 40  # badge bottom + breathing room before headline
    hf = font(True, 32)
    for line in hook_lines:
        draw.text((cx, hy), line, fill=ACCENT, font=hf, anchor="mm")
        hy += 42

    draw.text((cx, hy + 8), subline, fill=TEXT, font=font(True, 26), anchor="mm")
    draw.text(
        (cx, hy + 52),
        "Full LSP · Remote SSH · Dev containers · Multi-agent",
        fill=PINK,
        font=font(False, 15),
        anchor="mm",
    )

    shot_top = hy + 88
    paste_screenshot(canvas, WORKSPACE or EDITOR, (50, shot_top, 1030, 900), border=4)
    draw = ImageDraw.Draw(canvas)
    cta_bar(draw, w, 930, "JOIN THE BUILD", REPO)
    out = OUT / out_name
    canvas.save(out)
    print(f"Wrote {out}")


def render_square_elon():
    render_square(
        "ide-v4-social-square-elon.png",
        hook_lines=["If Elon Musk bought your", "favorite IDE"],
    )


def render_story():
    w, h = 1080, 1920
    canvas = Image.new("RGB", (w, h), BG)
    gradient_bg(canvas, (14, 12, 32), (8, 18, 28))
    draw = ImageDraw.Draw(canvas)
    cx = w // 2

    badge(draw, "IDE v4", cx - 36, 72, ACCENT)
    draw.text((cx, 140), "YOUR IDE.", fill=TEXT, font=font(True, 44), anchor="mm")
    draw.text((cx, 192), "YOUR REPO.", fill=TEAL, font=font(True, 44), anchor="mm")
    draw.text((cx, 244), "YOUR ROADMAP.", fill=ACCENT, font=font(True, 44), anchor="mm")

    sy = 310
    for line in wrap(
        draw,
        "Open-source agent IDE with full language-server depth, remote SSH workspaces, and human approval on every file change.",
        font(False, 18),
        w - 80,
    ):
        draw.text((cx, sy), line, fill=MUTED, font=font(False, 18), anchor="mm")
        sy += 26

    paste_screenshot(canvas, EDITOR or WORKSPACE, (40, 360, 1040, 1580), border=4)
    draw = ImageDraw.Draw(canvas)
    cta_bar(draw, w, 1620, "DOWNLOAD BETA", "macOS · Windows · Linux")
    draw.text((cx, 1760), RELEASE, fill=MUTED, font=font(False, 14), anchor="mm")
    out = OUT / "ide-v4-story-vertical.png"
    canvas.save(out)
    print(f"Wrote {out}")


def render_carousel_slide(num, headline, sublines, shot, bullets_list=None, cta=None):
    w, h = 1080, 1080
    canvas = Image.new("RGB", (w, h), BG)
    gradient_bg(canvas)
    draw = ImageDraw.Draw(canvas)
    cx = w // 2

    badge(draw, f"SLIDE {num} / 5", 48, 44, MUTED)
    draw.text((cx, 110), headline, fill=ACCENT if num in (1, 5) else TEXT, font=font(True, 36), anchor="mm")

    sy = 168
    for sub in sublines:
        for line in wrap(draw, sub, font(False, 17), w - 100):
            draw.text((cx, sy), line, fill=MUTED, font=font(False, 17), anchor="mm")
            sy += 24
        sy += 6

    if bullets_list:
        sy = bullets(draw, 80, sy + 8, bullets_list, w - 120, size=16) + 8

    shot_top = max(sy + 16, 280)
    paste_screenshot(canvas, shot, (60, shot_top, 1020, 900 if not cta else 860), border=4)

    if cta:
        draw = ImageDraw.Draw(canvas)
        cta_bar(draw, w, 920, cta, RELEASE)

    out = OUT / f"ide-v4-carousel-{num:02d}.png"
    canvas.save(out)
    print(f"Wrote {out}")


def render_carousel():
    render_carousel_slide(
        1,
        "Your IDE. Their acquisition.",
        ["Reports suggest major AI coding tools are consolidating.", "Who controls the roadmap now?"],
        WORKSPACE,
    )
    render_carousel_slide(
        2,
        "The uncertainty developers feel",
        [],
        AGENTS,
        bullets_list=[
            "Closed roadmaps after buyouts",
            "Cloud defaults — your code leaves the machine",
            "Workflows built around tools you don't own",
        ],
    )
    render_carousel_slide(
        3,
        "IDE v4 depth",
        ["Serious editor features, not a thin chat wrapper."],
        EDITOR,
        bullets_list=[
            "Full Monaco LSP — gopls, rust-analyzer, pyright",
            "Remote SSH via nj-remote sidecar",
            "Dev container attach for .devcontainer repos",
        ],
    )
    render_carousel_slide(
        4,
        "Open source. Local-first.",
        [],
        CHAT or AGENTS,
        bullets_list=[
            "Fork it · run it locally · send a PR",
            "BYOM — Ollama on your machine, cloud when you choose",
            "Human approval on every file change",
        ],
    )
    render_carousel_slide(
        5,
        "Join the build",
        ["Download the beta. Open an issue. Contribute a feature.", "Build the IDE you actually own."],
        WORKSPACE,
        cta="GET STARTED",
    )


VARIANTS = {
    "hero": [render_hero],
    "square": [lambda: render_square()],
    "square-elon": [render_square_elon],
    "story": [render_story],
    "carousel": [render_carousel],
}

OUT.mkdir(parents=True, exist_ok=True)

if VARIANT == "all":
    for fn in (render_hero, lambda: render_square(), render_square_elon, render_story, render_carousel):
        fn()
elif VARIANT in VARIANTS:
    for fn in VARIANTS[VARIANT]:
        fn()
else:
    raise SystemExit(f"unknown variant {VARIANT!r}; use hero|square|square-elon|story|carousel|all")
PY
