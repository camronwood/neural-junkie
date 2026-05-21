#!/usr/bin/env bash
# Compose v1.0.0-beta.13 feature ads (1080×1080).
# Usage: ./scripts/compose-beta13-ads.sh [security|cli-agents|scan-slack|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
VARIANT="${1:-all}"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT/assets" "$VARIANT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ASSETS = Path(sys.argv[1])
VARIANT = sys.argv[2]
W, H = 1080, 1080
MARGIN = 56
BG_TOP = (13, 13, 26)
BG_BOT = (22, 18, 42)
ACCENT = (233, 69, 96)
TEXT = (255, 255, 255)
MUTED = (168, 176, 184)
PANEL = (18, 24, 44)
PANEL_EDGE = (48, 58, 88)
PINK = (199, 125, 255)
GREEN = (72, 199, 142)
BLUE = (100, 180, 255)
VERSION = "v1.0.0-beta.13"


def font(bold: bool, size: int, mono: bool = False):
    if mono:
        paths = ["/System/Library/Fonts/Menlo.ttc"]
    else:
        paths = [
            "/System/Library/Fonts/Supplemental/Arial Bold.ttf"
            if bold
            else "/System/Library/Fonts/Supplemental/Arial.ttf",
        ]
    for path in paths:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def gradient_bg(canvas: Image.Image):
    draw = ImageDraw.Draw(canvas)
    for y in range(H):
        t = y / max(H - 1, 1)
        r = int(BG_TOP[0] + (BG_BOT[0] - BG_TOP[0]) * t)
        g = int(BG_TOP[1] + (BG_BOT[1] - BG_TOP[1]) * t)
        b = int(BG_TOP[2] + (BG_BOT[2] - BG_TOP[2]) * t)
        draw.line([(0, y), (W, y)], fill=(r, g, b))


def draw_brand(draw, kicker: str):
    draw.text((W // 2, 40), kicker, fill=MUTED, font=font(False, 13), anchor="mm")
    title = "NEURAL JUNKIE"
    ft = font(True, 48)
    tw = draw.textlength(title, font=ft)
    tx = (W - tw) // 2
    draw.text((tx, 64), title, fill=TEXT, font=ft)
    draw.rectangle((tx, 118, tx + tw, 122), fill=ACCENT)


def draw_cta(draw, y_btn: int):
    btn_h = 56
    draw.rounded_rectangle(
        (MARGIN, y_btn, W - MARGIN, y_btn + btn_h),
        radius=14,
        fill=ACCENT,
    )
    draw.text(
        (W // 2, y_btn + btn_h // 2),
        f"Download — {VERSION}",
        fill=TEXT,
        font=font(True, 20),
        anchor="mm",
    )
    draw.text(
        (W // 2, y_btn + btn_h + 26),
        "github.com/camronwood/neural-junkie/releases",
        fill=(120, 128, 150),
        font=font(False, 14),
        anchor="mm",
    )


def new_canvas():
    img = Image.new("RGB", (W, H), BG_TOP)
    gradient_bg(img)
    return img, ImageDraw.Draw(img)


def ad_security():
    img, draw = new_canvas()
    draw_brand(draw, "LOCAL-FIRST · SECURITY")
    headline = "Hub on loopback.\nSessions on requests."
    hf = font(True, 42)
    y = 150
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 50
    box = (MARGIN, 280, W - MARGIN, 720)
    draw.rounded_rectangle(box, radius=18, fill=PANEL, outline=GREEN, width=2)
    bullets = [
        "127.0.0.1:18765 by default",
        "X-NJ-Session + channel ACL",
        "Rate limits · encrypted secrets",
        "Tauri credential encryption",
    ]
    ty = box[1] + 36
    for b in bullets:
        draw.text((box[0] + 28, ty), f"✓  {b}", fill=TEXT, font=font(False, 22), anchor="lt")
        ty += 44
    draw.text((W // 2, 748), "docs/SECURITY.md · opt-in LAN bind", fill=MUTED, font=font(False, 17), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-security-local-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_cli_agents():
    img, draw = new_canvas()
    draw_brand(draw, "12 CLI AGENTS · AUTO-DETECT")
    headline = "Copilot. Codex. Cursor.\nPlus nine more on PATH."
    hf = font(True, 38)
    y = 140
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 46
    agents = [
        "cursor", "claude", "gemini", "copilot", "codex",
        "aider", "opencode", "amazonq", "crush", "amp", "droid", "kiro",
    ]
    cols, rows = 4, 3
    gw = (W - 2 * MARGIN) // cols
    gh = 88
    y0 = 300
    for i, name in enumerate(agents):
        col, row = i % cols, i // cols
        x = MARGIN + col * gw + gw // 2
        yc = y0 + row * gh + gh // 2
        draw.rounded_rectangle(
            (x - gw // 2 + 8, yc - 28, x + gw // 2 - 8, yc + 28),
            radius=10,
            fill=(24, 32, 58),
            outline=PANEL_EDGE,
            width=1,
        )
        draw.text((x, yc), name, fill=PINK if name in ("copilot", "codex") else MUTED, font=font(True, 15), anchor="mm")
    draw.text((W // 2, 620), "Join #general on hub start · /list-cli-agents", fill=MUTED, font=font(False, 17), anchor="mm")
    draw.text((W // 2, 655), "Modern copilot + legacy github-copilot-cli", fill=MUTED, font=font(False, 15), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-cli-agents-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_scan_slack():
    img, draw = new_canvas()
    draw_brand(draw, "SCAN SUMMARIES · SLACK")
    headline = "Plates in the UI.\nThreads in Slack."
    hf = font(True, 42)
    y = 150
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 50
    left = (MARGIN, 290, W // 2 - 16, 700)
    right = (W // 2 + 16, 290, W - MARGIN, 700)
    draw.rounded_rectangle(left, radius=16, fill=PANEL, outline=BLUE, width=2)
    draw.rounded_rectangle(right, radius=16, fill=PANEL, outline=GREEN, width=2)
    draw.text(((left[0] + left[2]) // 2, left[1] + 24), "SCAN SUMMARY", fill=BLUE, font=font(True, 16), anchor="mm")
    grid_x, grid_y = left[0] + 24, left[1] + 56
    for r in range(4):
        for c in range(6):
            col = GREEN if (r + c) % 3 == 0 else (40, 48, 72)
            draw.rectangle(
                (grid_x + c * 36, grid_y + r * 36, grid_x + c * 36 + 30, grid_y + r * 36 + 30),
                fill=col,
            )
    draw.text((left[0] + 20, left[3] - 44), "Well nav · TIFF preview", fill=MUTED, font=font(False, 15), anchor="lt")
    draw.text(((right[0] + right[2]) // 2, right[1] + 24), "SLACK BRIDGE", fill=GREEN, font=font(True, 16), anchor="mm")
    slack_lines = ["#general ↔ channel", "OAuth · bindings API", "test-post from Settings"]
    ty = right[1] + 80
    for line in slack_lines:
        draw.text((right[0] + 24, ty), f"• {line}", fill=MUTED, font=font(False, 18), anchor="lt")
        ty += 36
    draw.text((W // 2, 748), "Biology pack · delegation between agents", fill=MUTED, font=font(False, 16), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-scan-slack-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


variants = {
    "security": ad_security,
    "cli-agents": ad_cli_agents,
    "scan-slack": ad_scan_slack,
}
if VARIANT == "all":
    for fn in variants.values():
        fn()
else:
    if VARIANT not in variants:
        raise SystemExit(f"unknown variant: {VARIANT}")
    variants[VARIANT]()
PY
