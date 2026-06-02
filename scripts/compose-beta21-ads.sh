#!/usr/bin/env bash
# Compose three beta.21 LinkedIn ads (1200×627) — one audience per image.
# Usage: ./scripts/compose-beta21-ads.sh [slack|chat|test|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VARIANT="${1:-all}"
PY="$ROOT/.venv-icon/bin/python"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT" "$VARIANT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(sys.argv[1])
VARIANT = sys.argv[2]
W, H = 1200, 627
MARGIN = 48
CONTENT_W = W - 2 * MARGIN
FOOTER_H = 46
FOOTER_Y = H - MARGIN - FOOTER_H
RELEASE = "github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21"

TEAL = (72, 180, 200)
AMBER = (255, 193, 94)
PINK = (199, 125, 255)
ACCENT = (233, 69, 96)
GREEN = (72, 199, 142)
SLACK = (224, 168, 255)
MUTED = (168, 176, 184)


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


def gradient_bg(canvas, top, bottom):
    draw = ImageDraw.Draw(canvas)
    for y in range(H):
        t = y / max(H - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (W, y)], fill=(r, g, b))


def badge(draw, text: str, accent):
    bf = font(True, 11)
    bw = int(draw.textlength(text, font=bf)) + 24
    bx0, by0 = MARGIN, 32
    draw.rounded_rectangle((bx0, by0, bx0 + bw, by0 + 28), radius=8, fill=(28, 32, 20), outline=accent, width=2)
    draw.text((bx0 + bw // 2, by0 + 14), text, fill=accent, font=bf, anchor="mm")


def footer(draw, text: str, fill=ACCENT):
    draw.rounded_rectangle((MARGIN, FOOTER_Y, MARGIN + CONTENT_W, FOOTER_Y + FOOTER_H), radius=12, fill=fill)
    draw.text(
        (MARGIN + CONTENT_W // 2, FOOTER_Y + FOOTER_H // 2),
        text,
        fill=(255, 255, 255),
        font=font(True, 13),
        anchor="mm",
    )


def bullet_list(draw, y: int, items, accent, max_w=CONTENT_W - 40):
    fs = font(False, 14)
    for item in items:
        draw.text((MARGIN + 8, y), "•", fill=accent, font=font(True, 16), anchor="lt")
        for line in wrap(draw, item, fs, max_w - 24):
            draw.text((MARGIN + 28, y), line, fill=(210, 215, 225), font=fs, anchor="lt")
            y += 20
        y += 6
    return y


def draw_layer_box(draw, x0, y0, x1, y1, num, title, sub, accent, mono):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=14, fill=(18, 24, 44), outline=accent, width=2)
    r = 17
    cx, cy = x0 + 28, y0 + (y1 - y0) // 2
    draw.ellipse((cx - r, cy - r, cx + r, cy + r), fill=accent)
    draw.text((cx, cy), num, fill=(255, 255, 255), font=font(True, 15), anchor="mm")
    tx = x0 + 56
    tw = x1 - tx - 16
    draw.text((tx, y0 + 16), title, fill=(255, 255, 255), font=font(True, 17), anchor="lt")
    fs = font(False, 11)
    for i, line in enumerate(wrap(draw, sub, fs, tw)[:2]):
        draw.text((tx, y0 + 38 + i * 15), line, fill=MUTED, font=fs, anchor="lt")
    draw.text((tx, y1 - 22), mono, fill=accent, font=font(True, 10, mono=True), anchor="lt")


def render_slack(out: Path):
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (18, 12, 36), (8, 20, 32))
    draw = ImageDraw.Draw(canvas)
    badge(draw, "beta.21 · SLACK · NEURAL JUNKIE", SLACK)

    hy = 76
    draw.text((MARGIN, hy), "SLACK ON YOUR PHONE.", fill=(255, 255, 255), font=font(True, 44), anchor="lt")
    draw.text((MARGIN, hy + 50), "AGENTS ON YOUR DESKTOP.", fill=SLACK, font=font(True, 40), anchor="lt")
    sy = hy + 104
    fsub = font(False, 15)
    for line in wrap(draw, "DM the NJ bot from Slack mobile — your specialist answers in the same thread.", fsub, CONTENT_W):
        draw.text((MARGIN, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 20

    sy = bullet_list(
        draw,
        sy + 16,
        [
            "Personal inbox → private hub channel + agent you pick",
            "Forward channels: @you, nj: prefix, or reaction emoji",
            "Away mode (opt-in): labeled replies in your 1:1 DMs",
        ],
        SLACK,
    )

    # Mock thread panel
    px0 = MARGIN + CONTENT_W // 2 + 12
    py0 = hy
    draw.rounded_rectangle((px0, py0, MARGIN + CONTENT_W, py0 + 200), radius=12, fill=(14, 18, 32), outline=TEAL, width=2)
    draw.text((px0 + 14, py0 + 12), "#eng  ·  forwarded", fill=TEAL, font=font(True, 10, mono=True), anchor="lt")
    thread = [
        "you: nj: summarize the API change",
        "BackendEngineer: Here's the diff…",
        "↳ replies in this Slack thread",
    ]
    ty = py0 + 36
    tf = font(False, 12, mono=True)
    for line in thread:
        draw.text((px0 + 14, ty), line, fill=(175, 185, 200), font=tf, anchor="lt")
        ty += 22

    footer(draw, RELEASE, (120, 80, 200))
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_chat(out: Path):
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (12, 20, 28), (8, 28, 24))
    draw = ImageDraw.Draw(canvas)
    badge(draw, "beta.21 · FOR BUILDERS", GREEN)

    hy = 76
    draw.text((MARGIN, hy), "CAN YOU SEE", fill=(255, 255, 255), font=font(True, 46), anchor="lt")
    draw.text((MARGIN, hy + 52), "MY WORKSPACE?", fill=GREEN, font=font(True, 48), anchor="lt")
    sy = hy + 108
    fsub = font(False, 15)
    for line in wrap(
        draw,
        "Stop getting fake package advice. Chat/Code mode, honest visibility answers, and closure that doesn't re-hash the whole thread.",
        fsub,
        CONTENT_W // 2 + 20,
    ):
        draw.text((MARGIN, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 20

    sy = bullet_list(
        draw,
        sy + 8,
        [
            "Chat vs Code composer — right tools for greetings vs reviews",
            "Workspace visibility — file tree & project name, not hallucinated deps",
            "Echo & closure fixes — “What?” and “thanks” behave like chat",
        ],
        GREEN,
        max_w=CONTENT_W // 2 + 10,
    )

    px0 = MARGIN + CONTENT_W // 2 + 8
    py0 = hy + 8
    draw.rounded_rectangle((px0, py0, MARGIN + CONTENT_W, py0 + 220), radius=12, fill=(12, 18, 28), outline=GREEN, width=2)
    draw.text((px0 + 14, py0 + 12), "Agent reply (excerpt)", fill=GREEN, font=font(True, 11), anchor="lt")
    reply = [
        "Yes — I have workspace context.",
        "Project: neural-junkie",
        "File tree: desktop/ internal/ …",
        "Context scope: outline",
    ]
    ty = py0 + 40
    tf = font(False, 12, mono=True)
    for line in reply:
        draw.text((px0 + 14, ty), line, fill=(180, 200, 190), font=tf, anchor="lt")
        ty += 20

    draw.text(
        (MARGIN, FOOTER_Y - 36),
        "Deep dive: docs/CONTEXT_MODEL.md",
        fill=MUTED,
        font=font(False, 12),
        anchor="lt",
    )
    footer(draw, RELEASE)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


def render_test(out: Path):
    canvas = Image.new("RGB", (W, H), (10, 14, 28))
    gradient_bg(canvas, (14, 16, 32), (8, 22, 30))
    draw = ImageDraw.Draw(canvas)
    badge(draw, "beta.21 · CLONE & CONTRIBUTE", PINK)

    hy = 76
    draw.text((MARGIN, hy), "DON'T CLICK-TEST", fill=(255, 255, 255), font=font(True, 42), anchor="lt")
    draw.text((MARGIN, hy + 46), "THE HUB.", fill=PINK, font=font(True, 46), anchor="lt")
    sy = hy + 98
    fsub = font(False, 15)
    for line in wrap(
        draw,
        "For maintainers and contributors: three layers so orchestrator logic and live Ollama conversations stay regression-safe.",
        fsub,
        CONTENT_W,
    ):
        draw.text((MARGIN, sy), line, fill=(200, 205, 220), font=fsub, anchor="lt")
        sy += 20

    box_h = 86
    gap = 10
    layers_y = FOOTER_Y - 16 - 3 * box_h - 2 * gap
    layers = [
        ("1", "Deterministic Go", "Router, collab lifecycle, plan parser — make test-go", TEAL, "CI on every push"),
        ("2", "API smoke", "Collab phases via HTTP — make collab-smoke", AMBER, "No Ollama"),
        ("3", "Live JSON scenarios", "Real agents — chat-scenarios · collab-scenario-regression", PINK, "Local pre-release"),
    ]
    for i, layer in enumerate(layers):
        y0 = layers_y + i * (box_h + gap)
        draw_layer_box(draw, MARGIN, y0, MARGIN + CONTENT_W, y0 + box_h, *layer)

    footer(draw, "docs/TESTING.md  ·  " + RELEASE)
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


VARIANTS = {
    "slack": ("neural-junkie-beta21-slack-ad-1200.png", render_slack),
    "chat": ("neural-junkie-beta21-chat-ad-1200.png", render_chat),
    "test": ("neural-junkie-beta21-test-ad-1200.png", render_test),
}

if VARIANT == "all":
    targets = VARIANTS.values()
else:
    if VARIANT not in VARIANTS:
        raise SystemExit(f"unknown variant {VARIANT!r}; use slack|chat|test|all")
    targets = [VARIANTS[VARIANT]]

assets = ROOT / "assets"
assets.mkdir(parents=True, exist_ok=True)
for name, fn in targets:
    fn(assets / name)
PY

# Legacy single combined image (deprecated — use audience-specific ads)
if [[ "$VARIANT" == "all" ]]; then
  echo "Tip: post slack, chat, and test ads separately — see docs/marketing/BETA21-LINKEDIN.md"
fi
