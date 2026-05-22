#!/usr/bin/env bash
# Compose v1.0.0-beta.13 feature ads (1080×1080).
# Usage: ./scripts/compose-beta13-ads.sh [security|cli-agents|tiff-previewer|slack|slack-nondev|slack-models|all]
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
VERSION = "v1.0.0-beta.15"


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
    draw_brand(draw, "12 CLI AGENTS · AUTO-DETECTED")
    headline = "Copilot. Codex. Cursor.\nTwelve agents, auto-detected."
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
    draw.text((W // 2, 620), "Auto-detected when you start the hub", fill=MUTED, font=font(False, 19), anchor="mm")
    draw.text((W // 2, 655), "They join #general — ready to @mention", fill=MUTED, font=font(False, 17), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-cli-agents-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_tiff_previewer():
    img, draw = new_canvas()
    draw_brand(draw, "LIFE SCIENCES · DESKTOP")
    headline = "TIFF previewer\nbuilt into the app."
    hf = font(True, 42)
    y = 150
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 52
    box = (MARGIN, 280, W - MARGIN, 720)
    draw.rounded_rectangle(box, radius=18, fill=PANEL, outline=BLUE, width=2)
    draw.text((box[0] + 24, box[1] + 20), "PLATE VIEW", fill=BLUE, font=font(True, 16), anchor="lt")
    grid_x, grid_y = box[0] + 32, box[1] + 64
    for r in range(5):
        for c in range(8):
            col = GREEN if (r + c) % 4 == 0 else (40, 48, 72)
            draw.rectangle(
                (grid_x + c * 34, grid_y + r * 34, grid_x + c * 34 + 28, grid_y + r * 34 + 28),
                fill=col,
            )
    preview = (box[0] + 520, box[1] + 120, box[2] - 24, box[3] - 80)
    draw.rounded_rectangle(preview, radius=12, fill=(24, 32, 58), outline=GREEN, width=2)
    draw.text((preview[0] + 16, preview[1] + 12), "TIFF PREVIEW", fill=GREEN, font=font(True, 14), anchor="lt")
    draw.text((preview[0] + 16, preview[1] + 44), "Well A3 · zoom · pan", fill=MUTED, font=font(False, 16), anchor="lt")
    draw.text((preview[0] + 16, preview[1] + 76), "Open from summary JSON", fill=MUTED, font=font(False, 15), anchor="lt")
    bullets = ["Click any well in the grid", "Preview microscopy TIFFs in-app", "No extra viewer app needed"]
    ty = box[3] - 100
    for b in bullets:
        draw.text((box[0] + 28, ty), f"• {b}", fill=MUTED, font=font(False, 17), anchor="lt")
        ty += 28
    draw.text((W // 2, 748), "Life sciences pack · BiologyExpert workflows", fill=MUTED, font=font(False, 16), anchor="mm")
    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-tiff-previewer-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def draw_slack_logo(draw, cx: int, cy: int, size: int = 80):
    """Slack four-color mark (simplified)."""
    blob = size // 3
    gap = size // 5
    colors = [(224, 30, 90), (54, 197, 240), (46, 182, 125), (236, 178, 46)]
    offsets = [(-gap, -gap), (gap, -gap), (-gap, gap), (gap, gap)]
    for col, (ox, oy) in zip(colors, offsets):
        x0, y0 = cx + ox - blob, cy + oy - blob
        x1, y1 = cx + ox + blob, cy + oy + blob
        draw.rounded_rectangle((x0, y0, x1, y1), radius=blob, fill=col)


def draw_chat_bubble(draw, x0, y0, x1, y1, text: str, fill, text_fill=TEXT, fs=14):
    draw.rounded_rectangle((x0, y0, x1, y1), radius=12, fill=fill)
    f = font(False, fs)
    ty = y0 + 10
    for line in text.split("\n"):
        draw.text((x0 + 14, ty), line, fill=text_fill, font=f)
        ty += fs + 6


def ad_slack():
    img, draw = new_canvas()
    SLACK_CYAN = (54, 197, 240)
    SLACK_PANEL = (48, 22, 58)
    BUBBLE_SLACK = (56, 38, 72)
    BUBBLE_BOT = (32, 48, 88)

    draw_brand(draw, "SLACK · ANY AGENT AS YOUR BOT")

    draw_slack_logo(draw, W // 2, 168, size=72)

    headline = "Any agent.\nOne Slack channel.\nMinimal setup."
    hf = font(True, 40)
    y = 218
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 46

    # Hero card: Slack thread ←→ NJ binding
    box = (MARGIN + 12, 358, W - MARGIN - 12, 668)
    draw.rounded_rectangle(box, radius=20, fill=SLACK_PANEL, outline=SLACK_CYAN, width=2)

    slack_col = box[0] + 24
    nj_col = box[2] - 248
    mid_x = W // 2

    draw_slack_logo(draw, slack_col + 36, box[1] + 36, size=36)
    draw.text((slack_col + 88, box[1] + 36), "#neural-junkie", fill=TEXT, font=font(True, 18), anchor="lm")
    draw.text((slack_col + 88, box[1] + 58), "in Slack", fill=MUTED, font=font(False, 14), anchor="lm")

    draw_chat_bubble(
        draw, slack_col, box[1] + 78, slack_col + 218, box[1] + 118,
        "@Neural Junkie add two numbers?", BUBBLE_SLACK, fs=13,
    )
    draw_chat_bubble(
        draw, slack_col + 12, box[1] + 128, slack_col + 230, box[1] + 198,
        "def add(a, b):\n    return a + b", BUBBLE_BOT, fs=12,
    )
    draw.text((slack_col + 12, box[1] + 206), "↳ Assistant · in thread", fill=SLACK_CYAN, font=font(False, 12), anchor="lt")

    # Bridge
    bridge_y = box[1] + 150
    draw.line([(slack_col + 240, bridge_y), (nj_col - 24, bridge_y)], fill=(70, 80, 110), width=3)
    draw.rounded_rectangle(
        (mid_x - 72, bridge_y - 22, mid_x + 72, bridge_y + 22),
        radius=12,
        fill=(24, 36, 62),
        outline=SLACK_CYAN,
        width=2,
    )
    draw.text((mid_x, bridge_y), "Connect Slack", fill=SLACK_CYAN, font=font(True, 15), anchor="mm")
    draw.text((mid_x, bridge_y + 38), "local hub", fill=MUTED, font=font(False, 12), anchor="mm")

    # NJ side
    draw.ellipse((nj_col - 4, box[1] + 28, nj_col + 52, box[1] + 84), fill=ACCENT)
    draw.text((nj_col + 24, box[1] + 58), "NJ", fill=TEXT, font=font(True, 20), anchor="mm")
    draw.text((nj_col + 64, box[1] + 36), "slack:C… channel", fill=TEXT, font=font(True, 17), anchor="lm")
    draw.text((nj_col + 64, box[1] + 58), "bind primary agent", fill=MUTED, font=font(False, 14), anchor="lm")

    agents = [
        ("Assistant", True),
        ("Cursor", False),
        ("Gemini", False),
        ("Claude", False),
        ("Codex", False),
        ("Biology", False),
    ]
    chip_x = nj_col
    chip_y = box[1] + 88
    chip_w = 76
    for i, (name, highlight) in enumerate(agents):
        col, row = i % 3, i // 3
        cx = chip_x + col * (chip_w + 4)
        cy = chip_y + row * 34
        draw.rounded_rectangle(
            (cx, cy, cx + chip_w, cy + 26),
            radius=8,
            fill=(233, 69, 96) if highlight else (24, 32, 58),
            outline=SLACK_CYAN if highlight else PANEL_EDGE,
            width=2 if highlight else 1,
        )
        label = name if len(name) <= 9 else name[:8] + "…"
        draw.text(
            (cx + chip_w // 2, cy + 13),
            label,
            fill=TEXT if highlight else MUTED,
            font=font(True, 10),
            anchor="mm",
        )

    draw.text(
        (W // 2, box[3] - 24),
        "Any LLM or agent NJ runs → one Slack bot per channel · no token paste",
        fill=MUTED,
        font=font(False, 16),
        anchor="mm",
    )

    bullets = [
        "Settings → Connect Slack (loopback OAuth)",
        "Pick channel · assign agent · @mention or always-on",
        "Replies sync both ways · tools approve in the app",
    ]
    ty = 688
    for b in bullets:
        draw.text((MARGIN + 8, ty), f"•  {b}", fill=MUTED, font=font(False, 15), anchor="lt")
        ty += 26

    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-slack-ad-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_slack_nondev():
    """Team / power-user audience — no code, agents, or OAuth jargon."""
    img, draw = new_canvas()
    SLACK_CYAN = (54, 197, 240)
    SLACK_PANEL = (48, 22, 58)
    BUBBLE_USER = (56, 38, 72)
    BUBBLE_AI = (40, 52, 78)

    draw_brand(draw, "SLACK · AI FOR YOUR TEAM")

    draw_slack_logo(draw, W // 2, 168, size=72)

    headline = "Ask in Slack.\nGet answers in the thread.\nOne-click setup."
    hf = font(True, 40)
    y = 218
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 46

    box = (MARGIN + 12, 358, W - MARGIN - 12, 668)
    draw.rounded_rectangle(box, radius=20, fill=SLACK_PANEL, outline=SLACK_CYAN, width=2)

    slack_col = box[0] + 24
    nj_col = box[2] - 258
    mid_x = W // 2

    draw_slack_logo(draw, slack_col + 36, box[1] + 36, size=36)
    draw.text((slack_col + 88, box[1] + 36), "#team-help", fill=TEXT, font=font(True, 18), anchor="lm")
    draw.text((slack_col + 88, box[1] + 58), "where your team already works", fill=MUTED, font=font(False, 13), anchor="lm")

    draw_chat_bubble(
        draw, slack_col, box[1] + 78, slack_col + 228, box[1] + 132,
        "@Neural Junkie help me plan\na team offsite for 12 people", BUBBLE_USER, fs=12,
    )
    draw_chat_bubble(
        draw, slack_col + 8, box[1] + 142, slack_col + 238, box[1] + 228,
        "Here are 3 venue ideas with\nrough budgets and a simple\nweekend agenda you can share.", BUBBLE_AI, fs=12,
    )
    draw.text((slack_col + 8, box[1] + 236), "↳ Your AI helper · in thread", fill=SLACK_CYAN, font=font(False, 12), anchor="lt")

    bridge_y = box[1] + 158
    draw.line([(slack_col + 248, bridge_y), (nj_col - 20, bridge_y)], fill=(70, 80, 110), width=3)
    draw.rounded_rectangle(
        (mid_x - 78, bridge_y - 24, mid_x + 78, bridge_y + 24),
        radius=12,
        fill=(24, 36, 62),
        outline=SLACK_CYAN,
        width=2,
    )
    draw.text((mid_x, bridge_y - 6), "Connect", fill=SLACK_CYAN, font=font(True, 16), anchor="mm")
    draw.text((mid_x, bridge_y + 14), "in Settings", fill=TEXT, font=font(False, 13), anchor="mm")
    draw.text((mid_x, bridge_y + 44), "on your computer", fill=MUTED, font=font(False, 11), anchor="mm")

    draw.ellipse((nj_col - 4, box[1] + 28, nj_col + 52, box[1] + 84), fill=ACCENT)
    draw.text((nj_col + 24, box[1] + 58), "NJ", fill=TEXT, font=font(True, 20), anchor="mm")
    draw.text((nj_col + 64, box[1] + 34), "Choose your helper", fill=TEXT, font=font(True, 17), anchor="lm")
    draw.text((nj_col + 64, box[1] + 56), "for each Slack channel", fill=MUTED, font=font(False, 13), anchor="lm")

    helpers = [
        ("Assistant", True),
        ("Writing", False),
        ("Planning", False),
        ("Research", False),
        ("Budget", False),
        ("Notes", False),
    ]
    chip_x = nj_col
    chip_y = box[1] + 86
    chip_w = 76
    for i, (name, highlight) in enumerate(helpers):
        col, row = i % 3, i // 3
        cx = chip_x + col * (chip_w + 4)
        cy = chip_y + row * 34
        draw.rounded_rectangle(
            (cx, cy, cx + chip_w, cy + 26),
            radius=8,
            fill=(233, 69, 96) if highlight else (24, 32, 58),
            outline=SLACK_CYAN if highlight else PANEL_EDGE,
            width=2 if highlight else 1,
        )
        draw.text(
            (cx + chip_w // 2, cy + 13),
            name,
            fill=TEXT if highlight else MUTED,
            font=font(True, 10),
            anchor="mm",
        )

    draw.text(
        (W // 2, box[3] - 24),
        "Your team stays in Slack — you pick which AI helps in each channel",
        fill=MUTED,
        font=font(False, 16),
        anchor="mm",
    )

    bullets = [
        "Connect Slack once — no copying answers between apps",
        "Writing coach, trip planner, reminders — your choice per channel",
        "Replies land in Slack threads your colleagues can read",
    ]
    ty = 688
    for b in bullets:
        draw.text((MARGIN + 8, ty), f"•  {b}", fill=MUTED, font=font(False, 15), anchor="lt")
        ty += 26

    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-slack-ad-nondev-1080.png"
    img.save(out, optimize=True)
    print(out)


def ad_slack_models():
    """Slack ad — pick real LLM backends (Claude, GPT, Ollama, etc.) per channel."""
    img, draw = new_canvas()
    SLACK_CYAN = (54, 197, 240)
    SLACK_PANEL = (48, 22, 58)
    BUBBLE_USER = (56, 38, 72)
    BUBBLE_AI = (40, 52, 78)
    GOLD = (236, 178, 46)
    CLAUDE_TAN = (217, 153, 110)
    GPT_GREEN = (116, 195, 101)
    GEMINI_BLUE = (100, 180, 255)

    draw_brand(draw, "SLACK · CHOOSE YOUR MODEL")

    draw_slack_logo(draw, W // 2, 168, size=72)

    headline = "Claude. Gemini. Ollama.\nBack your Slack bot.\nYour keys, your choice."
    hf = font(True, 38)
    y = 218
    for line in headline.split("\n"):
        draw.text((W // 2, y), line, fill=TEXT, font=hf, anchor="mm")
        y += 44

    box = (MARGIN + 12, 352, W - MARGIN - 12, 672)
    draw.rounded_rectangle(box, radius=20, fill=SLACK_PANEL, outline=SLACK_CYAN, width=2)

    slack_col = box[0] + 24
    nj_col = box[2] - 268
    mid_x = W // 2

    draw_slack_logo(draw, slack_col + 36, box[1] + 32, size=34)
    draw.text((slack_col + 84, box[1] + 32), "#product", fill=TEXT, font=font(True, 17), anchor="lm")
    draw.text((slack_col + 84, box[1] + 54), "one bot, your model", fill=MUTED, font=font(False, 13), anchor="lm")

    draw_chat_bubble(
        draw, slack_col, box[1] + 74, slack_col + 224, box[1] + 118,
        "@Neural Junkie tighten this\nlaunch email?", BUBBLE_USER, fs=12,
    )
    draw_chat_bubble(
        draw, slack_col + 8, box[1] + 128, slack_col + 232, box[1] + 198,
        "Here's a shorter version with\na clearer CTA — powered by\nthe model you picked.", BUBBLE_AI, fs=12,
    )
    draw.text((slack_col + 8, box[1] + 206), "↳ Claude · in thread", fill=CLAUDE_TAN, font=font(False, 12), anchor="lt")

    bridge_y = box[1] + 152
    draw.line([(slack_col + 242, bridge_y), (nj_col - 18, bridge_y)], fill=(70, 80, 110), width=3)
    draw.rounded_rectangle(
        (mid_x - 80, bridge_y - 26, mid_x + 80, bridge_y + 26),
        radius=12,
        fill=(24, 36, 62),
        outline=SLACK_CYAN,
        width=2,
    )
    draw.text((mid_x, bridge_y - 8), "Connect Slack", fill=SLACK_CYAN, font=font(True, 15), anchor="mm")
    draw.text((mid_x, bridge_y + 12), "pick model", fill=TEXT, font=font(False, 13), anchor="mm")
    draw.text((mid_x, bridge_y + 32), "per channel", fill=MUTED, font=font(False, 11), anchor="mm")

    draw.text((nj_col + 4, box[1] + 28), "Models that can", fill=TEXT, font=font(True, 16), anchor="lt")
    draw.text((nj_col + 4, box[1] + 50), "power your bot", fill=MUTED, font=font(False, 13), anchor="lt")

    models = [
        ("Claude", True, CLAUDE_TAN),
        ("GPT", False, GPT_GREEN),
        ("Gemini", False, GEMINI_BLUE),
        ("Ollama", False, GREEN),
        ("LM Studio", False, MUTED),
        ("OpenAI API", False, MUTED),
        ("HuggingFace", False, PINK),
        ("DeepSeek", False, MUTED),
        ("Local", False, SLACK_CYAN),
    ]
    chip_x = nj_col
    chip_y = box[1] + 72
    chip_w = 82
    chip_h = 28
    for i, (name, highlight, accent) in enumerate(models):
        col, row = i % 3, i // 3
        cx = chip_x + col * (chip_w + 3)
        cy = chip_y + row * (chip_h + 6)
        draw.rounded_rectangle(
            (cx, cy, cx + chip_w, cy + chip_h),
            radius=8,
            fill=(233, 69, 96) if highlight else (24, 32, 58),
            outline=accent if highlight else PANEL_EDGE,
            width=2 if highlight else 1,
        )
        label = name if len(name) <= 11 else name[:10] + "…"
        draw.text(
            (cx + chip_w // 2, cy + chip_h // 2),
            label,
            fill=TEXT if highlight else MUTED,
            font=font(True, 10),
            anchor="mm",
        )

    draw.text(
        (W // 2, box[3] - 22),
        "Cloud APIs · local Ollama · OpenAI-compatible — route per Slack channel",
        fill=MUTED,
        font=font(False, 15),
        anchor="mm",
    )

    bullets = [
        "Anthropic · OpenAI · Google Gemini · Ollama · LM Studio · HF",
        "#sales on Claude · #eng on local Ollama — same Slack, different brains",
        "Your API keys stay in Settings — colleagues just @mention the bot",
    ]
    ty = 688
    for b in bullets:
        draw.text((MARGIN + 8, ty), f"•  {b}", fill=MUTED, font=font(False, 14), anchor="lt")
        ty += 24

    draw_cta(draw, 820)
    out = ASSETS / "neural-junkie-slack-ad-models-1080.png"
    img.save(out, optimize=True)
    print(out)


variants = {
    "security": ad_security,
    "cli-agents": ad_cli_agents,
    "tiff-previewer": ad_tiff_previewer,
    "slack": ad_slack,
    "slack-nondev": ad_slack_nondev,
    "slack-models": ad_slack_models,
}
if VARIANT == "all":
    for fn in variants.values():
        fn()
else:
    if VARIANT not in variants:
        raise SystemExit(f"unknown variant: {VARIANT}")
    variants[VARIANT]()
PY
