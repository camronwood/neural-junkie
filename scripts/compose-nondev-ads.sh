#!/usr/bin/env bash
# Compose 1080×1080 non-developer audience ads — three distinct layouts.
# Usage: ./scripts/compose-nondev-ads.sh [providers|experts|collaborate|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
VARIANT="${1:-all}"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

exec "$PY" - "$ROOT/assets" "$VARIANT" <<'PY'
import math
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ASSETS = Path(sys.argv[1])
VARIANT = sys.argv[2]
W, H = 1080, 1080
MARGIN = 64  # global horizontal margin


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


def gradient_bg(canvas: Image.Image, top, bottom):
    draw = ImageDraw.Draw(canvas)
    for y in range(H):
        t = y / max(H - 1, 1)
        r = int(top[0] + (bottom[0] - top[0]) * t)
        g = int(top[1] + (bottom[1] - top[1]) * t)
        b = int(top[2] + (bottom[2] - top[2]) * t)
        draw.line([(0, y), (W, y)], fill=(r, g, b))


def row_start(count: int, item_w: int, gap: int, container_w: int = W) -> int:
    """Left x for a horizontally centered row of `count` items."""
    total = count * item_w + (count - 1) * gap
    return (container_w - total) // 2


def draw_cta_block(draw, y_btn: int, btn_label: str, url: str, btn_inset: int = MARGIN):
    btn_h = 56
    draw.rounded_rectangle(
        (btn_inset, y_btn, W - btn_inset, y_btn + btn_h),
        radius=14,
        fill=(233, 69, 96),
    )
    draw.text(
        (W // 2, y_btn + btn_h // 2),
        btn_label,
        fill=(255, 255, 255),
        font=font(True, 22),
        anchor="mm",
    )
    draw.text(
        (W // 2, y_btn + btn_h + 28),
        url,
        fill=(136, 140, 168),
        font=font(False, 15),
        anchor="mm",
    )


def draw_brand_underline(draw, y_title: int, title: str, fnt):
    draw.text((W // 2, y_title), title, fill=(255, 255, 255), font=fnt, anchor="mm")
    tw = draw.textlength(title, font=fnt)
    ux0 = (W - tw) // 2
    draw.rectangle((ux0, y_title + 26, ux0 + tw, y_title + 30), fill=(233, 69, 96))


# ─── Ad 1: Hero typography + centered tab row ─────────────────────────────────

def ad_providers():
    canvas = Image.new("RGB", (W, H), (10, 12, 28))
    gradient_bg(canvas, (8, 10, 26), (22, 18, 48))
    draw = ImageDraw.Draw(canvas)

    # Centered tab row (symmetric)
    n_tabs, tw, th, tab_gap = 5, 168, 34, 10
    x0 = row_start(n_tabs, tw, tab_gap)
    y_tab = 48
    tab_colors = [(90, 100, 130), (120, 90, 140), (80, 120, 150), (140, 110, 80), (100, 130, 110)]
    for i, col in enumerate(tab_colors):
        tx = x0 + i * (tw + tab_gap)
        draw.rounded_rectangle((tx, y_tab, tx + tw, y_tab + th), radius=6, fill=(28, 32, 52), outline=col, width=2)
        cx_close = tx + tw - 18
        cy_close = y_tab + th // 2
        draw.line(
            [(cx_close - 5, cy_close - 5), (cx_close + 5, cy_close + 5)],
            fill=(233, 69, 96),
            width=2,
        )
        draw.line(
            [(cx_close + 5, cy_close - 5), (cx_close - 5, cy_close + 5)],
            fill=(233, 69, 96),
            width=2,
        )
        draw.text(
            (tx + tw // 2, y_tab + th // 2),
            f"Chat {i + 1}",
            fill=(120, 128, 150),
            font=font(False, 11),
            anchor="mm",
        )

    y = y_tab + th + 36
    draw.text((W // 2, y), "FIVE AI TABS", fill=(120, 128, 150), font=font(True, 26), anchor="mm")
    y += 52
    draw.text((W // 2, y), "ONE DESKTOP", fill=(255, 255, 255), font=font(True, 68), anchor="mm")
    y += 72
    draw.text(
        (W // 2, y),
        "Claude · Ollama · your other API — routed per agent",
        fill=(255, 140, 110),
        font=font(True, 21),
        anchor="mm",
    )
    y += 44
    body = (
        "Neural Junkie keeps writing, research, and brainstorming in one "
        "Slack-style workspace — not five copy-pasted chats."
    )
    fbody = font(False, 19)
    body_lines = wrap(draw, body, fbody, W - 2 * MARGIN - 40)
    for line in body_lines:
        draw.text((W // 2, y), line, fill=(200, 205, 220), font=fbody, anchor="mm")
        y += 26

    # Icon boxes — fixed row
    boxes = [
        ("stack", "One workspace", "Channels, DMs, threads"),
        ("key", "Your keys", "API + local models"),
        ("chart", "Your budget", "Track usage in Settings"),
    ]
    bw, bh, box_gap = 296, 136, 22
    bx0 = row_start(3, bw, box_gap)
    by0 = y + 28
    icon_bottom = by0 + 52
    for i, (kind, title, sub) in enumerate(boxes):
        x = bx0 + i * (bw + box_gap)
        cx = x + bw // 2
        draw.rounded_rectangle((x, by0, x + bw, by0 + bh), radius=14, outline=(83, 52, 131), width=2)
        iy = by0 + 14
        if kind == "stack":
            for j in range(3):
                ox = cx - 26 + j * 7
                draw.rounded_rectangle(
                    (ox, iy + j * 5, ox + 36, iy + 24 + j * 5),
                    radius=4,
                    outline=(199, 125, 255),
                    width=2,
                )
        elif kind == "key":
            draw.ellipse((cx - 12, iy + 2, cx + 12, iy + 22), outline=(72, 199, 142), width=2)
            draw.rectangle((cx - 22, iy + 12, cx - 6, iy + 18), outline=(72, 199, 142), width=2)
        else:
            for j, h in enumerate([16, 26, 20]):
                draw.rectangle(
                    (cx - 20 + j * 13, icon_bottom - 4 - h, cx - 8 + j * 13, icon_bottom - 4),
                    fill=(255, 193, 94),
                )
        draw.text((cx, by0 + 78), title, fill=(255, 255, 255), font=font(True, 17), anchor="mm")
        draw.text((cx, by0 + 102), sub, fill=(168, 176, 184), font=font(False, 13), anchor="mm")

    draw.text((W // 2, by0 + bh + 32), "NEURAL JUNKIE", fill=(255, 255, 255), font=font(True, 24), anchor="mm")
    draw_cta_block(
        draw,
        756,
        "Open source beta — macOS · Windows · Linux",
        "github.com/camronwood/neural-junkie/releases",
    )

    out = ASSETS / "neural-junkie-nondev-providers-ad-1080.png"
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


# ─── Ad 2: 2×2 persona grid ───────────────────────────────────────────────────

def draw_persona_card(draw, box, badge: str, title: str, body: str, accent):
    x0, y0, x1, y1 = box
    pad = 20
    draw.rounded_rectangle(box, radius=16, fill=(22, 28, 50), outline=accent, width=2)
    badge_f = font(True, 22)
    bw = int(draw.textlength(badge, font=badge_f)) + 16
    draw.rounded_rectangle(
        (x0 + pad, y0 + pad, x0 + pad + bw, y0 + pad + 32),
        radius=8,
        fill=(24, 30, 52),
        outline=accent,
    )
    draw.text((x0 + pad + bw // 2, y0 + pad + 16), badge, fill=accent, font=badge_f, anchor="mm")
    draw.text((x0 + pad, y0 + pad + 44), title, fill=(255, 255, 255), font=font(True, 21), anchor="lt")
    f = font(False, 14)
    lines = wrap(draw, body, f, x1 - x0 - 2 * pad)[:3]
    ty = y0 + pad + 74
    line_h = 19
    for line in lines:
        draw.text((x0 + pad, ty), line, fill=(180, 188, 200), font=f, anchor="lt")
        ty += line_h


def ad_experts():
    canvas = Image.new("RGB", (W, H), (16, 14, 32))
    gradient_bg(canvas, (24, 20, 44), (12, 14, 28))
    draw = ImageDraw.Draw(canvas)

    y = 52
    ft_brand = font(True, 34)
    draw_brand_underline(draw, y, "NEURAL JUNKIE", ft_brand)
    y += 56
    draw.text((W // 2, y), "Who do you need today?", fill=(199, 125, 255), font=font(True, 38), anchor="mm")
    y += 48
    draw.text(
        (W // 2, y),
        "Custom experts in private DMs — no codebase required",
        fill=(200, 205, 220),
        font=font(False, 18),
        anchor="mm",
    )

    gap = 18
    cw = (W - 2 * MARGIN - gap) // 2
    ch = 188
    y_grid = y + 40
    cards = [
        ("W", "Writing coach", "Investor updates, emails, tone and clarity", (255, 140, 110)),
        ("T", "Trip planner", "Itineraries, budgets, packing lists", (72, 180, 200)),
        ("$", "Budget buddy", "Scenarios, tradeoffs, what-if math", (72, 199, 142)),
        ("R", "Research partner", "Summarize articles, compare options", (255, 193, 94)),
    ]
    coords = [
        (MARGIN, y_grid),
        (MARGIN + cw + gap, y_grid),
        (MARGIN, y_grid + ch + gap),
        (MARGIN + cw + gap, y_grid + ch + gap),
    ]
    for box_xy, card in zip(coords, cards):
        x, y0 = box_xy
        draw_persona_card(draw, (x, y0, x + cw, y0 + ch), *card)

    strip_h = 64
    strip_y = y_grid + 2 * ch + gap + 20
    draw.rounded_rectangle(
        (MARGIN, strip_y, W - MARGIN, strip_y + strip_h),
        radius=12,
        fill=(28, 34, 58),
        outline=(83, 52, 131),
    )
    draw.text(
        (W // 2, strip_y + 20),
        "Plus Assistant:  tasks · reminders · notes · /summarize",
        fill=(199, 125, 255),
        font=font(True, 16),
        anchor="mm",
    )
    draw.text(
        (W // 2, strip_y + 42),
        "/create-expert or New DM to add any persona",
        fill=(168, 176, 184),
        font=font(False, 14),
        anchor="mm",
    )

    draw_cta_block(
        draw,
        868,
        "Download — macOS · Windows · Linux",
        "github.com/camronwood/neural-junkie/releases",
        btn_inset=120,
    )

    out = ASSETS / "neural-junkie-nondev-experts-ad-1080.png"
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


# ─── Ad 3: Split panel ────────────────────────────────────────────────────────

def hex_points(cx, cy, r):
    return [
        (
            cx + r * math.cos(math.radians(60 * i - 30)),
            cy + r * math.sin(math.radians(60 * i - 30)),
        )
        for i in range(6)
    ]


def draw_hex_cluster(draw, cx, cy, r=44):
    hr = r * 0.52
    layout = [
        (0, 0, "You", (233, 69, 96)),
        (-r * 1.05, -r * 0.6, "R", (199, 125, 255)),
        (r * 1.05, -r * 0.6, "F", (255, 193, 94)),
        (-r * 1.05, r * 0.6, "W", (72, 180, 200)),
        (r * 1.05, r * 0.6, "✓", (72, 199, 142)),
    ]
    for dx, dy, lab, col in layout:
        hx, hy = cx + dx, cy + dy
        draw.polygon(hex_points(hx, hy, hr), outline=col, fill=(24, 30, 52))
        draw.text((hx, hy), lab, fill=(255, 255, 255), font=font(True, 13), anchor="mm")


def draw_flow_pills(draw, cx, cy, items):
    """Centered row of pills with arrows in gaps."""
    pad_x, pill_h = 12, 34
    f = font(True, 12)
    widths = [int(draw.textlength(lab, font=f)) + 2 * pad_x for lab, _ in items]
    arrow_w = 20
    gap = 10
    total = sum(widths) + (len(items) - 1) * (arrow_w + gap)
    x = cx - total // 2
    mid_y = cy + pill_h // 2
    for i, (lab, col) in enumerate(items):
        w = widths[i]
        draw.rounded_rectangle((x, cy, x + w, cy + pill_h), radius=8, fill=(24, 30, 52), outline=col, width=2)
        draw.text((x + w // 2, mid_y), lab, fill=(255, 255, 255), font=f, anchor="mm")
        x += w
        if i < len(items) - 1:
            ax0, ax1 = x + 4, x + arrow_w - 4
            draw.line([(ax0, mid_y), (ax1, mid_y)], fill=col, width=2)
            draw.polygon([(ax1, mid_y), (ax1 - 5, mid_y - 4), (ax1 - 5, mid_y + 4)], fill=col)
            x += arrow_w + gap


def ad_collaborate():
    canvas = Image.new("RGB", (W, H), (10, 12, 24))
    draw = ImageDraw.Draw(canvas)

    left_w = 488
    pad_l = 48
    draw.rectangle((0, 0, left_w, H), fill=(14, 18, 36))
    draw.line([(left_w, 0), (left_w, H)], fill=(83, 52, 131), width=2)

    right_x0 = left_w
    right_w = W - left_w
    rx = right_x0 + right_w // 2

    # Left column
    y = 56
    draw.text((pad_l, y), "BEYOND ONE CHATBOT", fill=(136, 140, 168), font=font(True, 13), anchor="lt")
    y += 36
    draw.text((pad_l, y), "Big decisions", fill=(255, 255, 255), font=font(True, 36), anchor="lt")
    y += 44
    draw.text((pad_l, y), "need a team.", fill=(255, 255, 255), font=font(True, 36), anchor="lt")

    steps = [
        ("01", "Specialists debate", "Research, finance, writing — bounded turns"),
        ("02", "One shared plan", "Not five copy-pasted threads"),
        ("03", "You approve", "Nothing runs until you say so"),
        ("04", "Parallel tasks", "Optional folder bind; no git required"),
    ]
    y = 200
    row_h = 92
    circle = 44
    text_x = pad_l + circle + 16
    for num, title, sub in steps:
        cy = y + row_h // 2
        draw.ellipse(
            (pad_l, y + (row_h - circle) // 2, pad_l + circle, y + (row_h + circle) // 2),
            fill=(233, 69, 96),
        )
        draw.text((pad_l + circle // 2, cy), num, fill=(255, 255, 255), font=font(True, 15), anchor="mm")
        draw.text((text_x, y + 14), title, fill=(255, 255, 255), font=font(True, 19), anchor="lt")
        sub_lines = wrap(draw, sub, font(False, 13), left_w - text_x - pad_l)
        sy = y + 38
        for line in sub_lines:
            draw.text((text_x, sy), line, fill=(168, 176, 184), font=font(False, 13), anchor="lt")
            sy += 17
        y += row_h

    draw.text((pad_l, 612), "NEURAL JUNKIE", fill=(199, 125, 255), font=font(True, 17), anchor="lt")
    draw.text((pad_l, 636), "/collaborate", fill=(120, 128, 150), font=font(False, 13), anchor="lt")

    # Right column — stacked, centered
    ry = 56
    draw.text(
        (rx, ry),
        "Launch · trip · budget · vendor review",
        fill=(200, 205, 220),
        font=font(False, 15),
        anchor="mm",
    )
    ry += 48
    draw_hex_cluster(draw, rx, ry + 70, r=46)
    ry += 168
    draw_flow_pills(
        draw,
        rx,
        ry,
        [
            ("Discuss", (199, 125, 255)),
            ("One plan", (255, 193, 94)),
            ("You OK", (233, 69, 96)),
            ("Execute", (72, 199, 142)),
        ],
    )
    ry += 52

    card_pad = 28
    card_h = 232
    card = (right_x0 + card_pad, ry, W - card_pad, ry + card_h)
    draw.rounded_rectangle(card, radius=14, fill=(22, 28, 50), outline=(48, 58, 88))
    quotes = [
        ("@Research", '"Competitor moved pricing — soft launch first."'),
        ("@Finance", '"Two budget tiers until week-4 metrics."'),
        ("@Writer", '"Headline A/B: outcome vs feature."'),
    ]
    qy = card[1] + 20
    row_quote = 58
    for who, q in quotes:
        draw.text((card[0] + 20, qy), who, fill=(199, 125, 255), font=font(True, 12), anchor="lt")
        draw.text((card[0] + 20, qy + 18), q, fill=(220, 224, 232), font=font(False, 14), anchor="lt")
        qy += row_quote

    btn_h = 36
    btn_y0 = card[3] - btn_h - 16
    draw.rounded_rectangle((card[0] + 16, btn_y0, card[2] - 16, card[3] - 16), radius=8, fill=(233, 69, 96))
    draw.text(
        ((card[0] + card[2]) // 2, (btn_y0 + card[3] - 16) // 2),
        "You approve → then execute",
        fill=(255, 255, 255),
        font=font(True, 13),
        anchor="mm",
    )

    draw_cta_block(
        draw,
        812,
        "Open source beta — macOS · Windows · Linux",
        "github.com/camronwood/neural-junkie/releases",
    )

    out = ASSETS / "neural-junkie-nondev-collaborate-ad-1080.png"
    canvas.save(out, "PNG")
    print(f"Wrote {out}")


variants = {
    "providers": [ad_providers],
    "experts": [ad_experts],
    "collaborate": [ad_collaborate],
    "all": [ad_providers, ad_experts, ad_collaborate],
}
funcs = variants.get(VARIANT)
if not funcs:
    raise SystemExit(f"Unknown variant: {VARIANT!r} (use providers|experts|collaborate|all)")
for fn in funcs:
    fn()
PY
