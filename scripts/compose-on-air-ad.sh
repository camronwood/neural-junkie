#!/usr/bin/env bash
# Compose On Air — 1080×1080 station-ID heroes (procedural, no plates).
# Usage: ./scripts/compose-on-air-ad.sh [baseline|team|features|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
OUT_DIR="$ROOT/campaigns/on-air/creatives"
VARIANT="${1:-all}"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

mkdir -p "$OUT_DIR"

exec "$PY" - "$OUT_DIR" "$VARIANT" "$ROOT" <<'PY'
import random
import sys
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

OUT_DIR = Path(sys.argv[1])
VARIANT = sys.argv[2]
ROOT = Path(sys.argv[3])
W, H = 1080, 1080

FIELD = (13, 15, 20)       # #0d0f14
SIGNAL = (233, 69, 96)     # #e94560
INK = (232, 234, 239)      # #e8eaef
QUIET = (139, 147, 167)    # #8b93a7
RELEASE = "github.com/camronwood/neural-junkie/releases/latest"

# Same order as docs/index.html hero-providers strip
PROVIDER_LOGOS = [
    ("claude.png", "Claude"),
    ("gpt.png", "GPT"),
    ("gemini.png", "Gemini"),
    ("ollama.png", "Ollama"),
    ("hf.png", "HF"),
    ("cursor.png", "Cursor"),
    ("lmstudio.png", "LM Studio"),
]
PROVIDER_DIR = ROOT / "docs" / "assets" / "providers"
INTEGRATION_DIR = ROOT / "docs" / "assets" / "integrations"

# First-class workspace integrations (Settings / STATUS) — framed tiles generated to match providers
INTEGRATION_SPECS = [
    ("slack", "Slack", (224, 30, 90)),
    ("meet", "G Meet", (0, 163, 113)),       # Google Meet notes
    ("github", "GitHub", (232, 234, 239)),
    ("confluence", "Confluence", (24, 100, 198)),
    ("jira", "Jira", (38, 132, 255)),
]

# baseline = brand callsign; team = differentiators; features = major pillars readout
VARIANTS = {
    "baseline": {
        "out": "on-air-hero-1080.png",
        "corner": "NJ  //  ON AIR",
        "support": "Any model. On yours.",
        "sub": None,
        "features": None,
        "logos": False,
        "footer": "OPEN SOURCE  ·  LOCAL FIRST",
    },
    "team": {
        "out": "on-air-team-1080.png",
        "corner": "NJ  //  ON AIR  ·  TEAM",
        "support": "Not another local chat.",
        "sub": "Specialists. Approvals. Your models.",
        "features": None,
        "logos": False,
        "footer": "MULTI-AGENT  ·  HUMAN-IN-THE-LOOP  ·  BYOM",
    },
    "features": {
        "out": "on-air-features-1080.png",
        "corner": "NJ  //  ON AIR  ·  FEATURES",
        "support": "One desk. Full stack.",
        "sub": None,
        # Station readout — major pillars from README "What you get" (no cards/grid)
        "features": [
            "BYOM",
            "MULTI-AGENT  ·  APPROVALS",
            "DOMAIN PACKS",
            "IDE V4  ·  FIX LOOP",
            "SLACK CONNECT  ·  COLLAB",
            "NEURAL CANVAS",
        ],
        "logos": True,
        "footer": "OPEN SOURCE  ·  LOCAL FIRST",
    },
}


def display_font(size: int):
    # Avenir Next Condensed: 8=Heavy, 0=Bold (upright — not italic).
    candidates = [
        ("/System/Library/Fonts/Avenir Next Condensed.ttc", 8),
        ("/System/Library/Fonts/Avenir Next Condensed.ttc", 0),
        ("/System/Library/Fonts/Supplemental/DIN Condensed Bold.ttf", 0),
        ("/System/Library/Fonts/Supplemental/Arial Narrow Bold.ttf", 0),
        ("/System/Library/Fonts/Supplemental/Arial Bold.ttf", 0),
    ]
    for path, index in candidates:
        try:
            return ImageFont.truetype(path, size, index=index)
        except OSError:
            continue
        except Exception:
            try:
                return ImageFont.truetype(path, size)
            except OSError:
                continue
    return ImageFont.load_default()


def mono_font(size: int):
    for path in (
        "/System/Library/Fonts/Menlo.ttc",
        "/System/Library/Fonts/Monaco.ttf",
    ):
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def fit_width(text: str, max_w: int, start: int, floor: int = 48) -> ImageFont.FreeTypeFont:
    size = start
    while size >= floor:
        fnt = display_font(size)
        bbox = fnt.getbbox(text)
        if bbox[2] - bbox[0] <= max_w:
            return fnt
        size -= 2
    return display_font(floor)


def add_grain(img: Image.Image, amount: int = 14) -> Image.Image:
    rng = random.Random(42)
    px = img.load()
    for y in range(0, H, 2):
        for x in range(0, W, 2):
            n = rng.randint(-amount, amount)
            r, g, b = px[x, y]
            px[x, y] = (
                max(0, min(255, r + n)),
                max(0, min(255, g + n)),
                max(0, min(255, b + n)),
            )
    return img


def load_provider_tile(path: Path, tile: int) -> Image.Image:
    """Fit provider mark into a square RGBA cell (site tiles already include frame)."""
    src = Image.open(path).convert("RGBA")
    src.thumbnail((tile, tile), Image.Resampling.LANCZOS)
    cell = Image.new("RGBA", (tile, tile), (0, 0, 0, 0))
    cell.paste(src, ((tile - src.width) // 2, (tile - src.height) // 2), src)
    return cell


def framed_tile(draw_mark, frame_rgb: tuple[int, int, int], out_w: int = 47, out_h: int = 72) -> Image.Image:
    """Match docs/assets/providers aesthetic: dark field, colored rail, white glyph plate."""
    img = Image.new("RGB", (out_w, out_h), (7, 7, 7))
    draw = ImageDraw.Draw(img)
    # Soft top/bottom brand rails (like provider tiles)
    draw.rectangle((0, 0, out_w - 1, 3), fill=frame_rgb)
    draw.rectangle((0, out_h - 4, out_w - 1, out_h - 1), fill=frame_rgb)
    # White glyph plate
    pad_x, pad_y = 8, 16
    plate = (pad_x, pad_y, out_w - 1 - pad_x, out_h - 1 - pad_y)
    draw.rounded_rectangle(plate, radius=4, fill=(245, 245, 245))
    draw_mark(draw, plate)
    return img.convert("RGBA")


def mark_slack(draw: ImageDraw.ImageDraw, plate: tuple[int, int, int, int]) -> None:
    x0, y0, x1, y1 = plate
    cx, cy = (x0 + x1) / 2, (y0 + y1) / 2
    size = min(x1 - x0, y1 - y0) * 0.72
    blob = size / 5.2
    gap = size / 3.4
    colors = [(224, 30, 90), (54, 197, 240), (46, 182, 125), (236, 178, 46)]
    # Classic four-lobed Slack hash
    arms = [
        (cx - gap, cy, True, colors[0]),   # left horizontal
        (cx, cy - gap, False, colors[1]),  # top vertical
        (cx + gap, cy, True, colors[2]),   # right horizontal
        (cx, cy + gap, False, colors[3]),  # bottom vertical
    ]
    for ax, ay, horiz, col in arms:
        if horiz:
            draw.rounded_rectangle(
                (ax - size * 0.28, ay - blob, ax + size * 0.28, ay + blob),
                radius=blob,
                fill=col,
            )
        else:
            draw.rounded_rectangle(
                (ax - blob, ay - size * 0.28, ax + blob, ay + size * 0.28),
                radius=blob,
                fill=col,
            )


def mark_meet(draw: ImageDraw.ImageDraw, plate: tuple[int, int, int, int]) -> None:
    """Simplified Google Meet camera glyph."""
    x0, y0, x1, y1 = plate
    cx, cy = (x0 + x1) / 2, (y0 + y1) / 2
    w, h = x1 - x0, y1 - y0
    # Camera body
    bw, bh = w * 0.42, h * 0.34
    body = (cx - bw * 0.55, cy - bh / 2, cx + bw * 0.25, cy + bh / 2)
    draw.rounded_rectangle(body, radius=3, fill=(0, 163, 113))
    # Lens triangle / wedge
    tip_x = cx + bw * 0.55
    draw.polygon(
        [
            (cx + bw * 0.15, cy - bh * 0.35),
            (tip_x, cy),
            (cx + bw * 0.15, cy + bh * 0.35),
        ],
        fill=(0, 163, 113),
    )


def mark_github(draw: ImageDraw.ImageDraw, plate: tuple[int, int, int, int]) -> None:
    x0, y0, x1, y1 = plate
    cx, cy = (x0 + x1) / 2, (y0 + y1) / 2
    r = min(x1 - x0, y1 - y0) * 0.28
    draw.ellipse((cx - r, cy - r * 1.05, cx + r, cy + r * 0.95), fill=(24, 24, 24))
    # ears
    ear = r * 0.45
    draw.ellipse((cx - r * 0.95, cy - r * 1.15, cx - r * 0.25, cy - r * 0.35), fill=(24, 24, 24))
    draw.ellipse((cx + r * 0.25, cy - r * 1.15, cx + r * 0.95, cy - r * 0.35), fill=(24, 24, 24))
    # tentacles hint
    draw.rounded_rectangle(
        (cx - r * 0.35, cy + r * 0.55, cx + r * 0.35, cy + r * 1.25),
        radius=2,
        fill=(24, 24, 24),
    )


def mark_confluence(draw: ImageDraw.ImageDraw, plate: tuple[int, int, int, int]) -> None:
    """Two interlocking arcs — Confluence-ish mark."""
    x0, y0, x1, y1 = plate
    cx, cy = (x0 + x1) / 2, (y0 + y1) / 2
    s = min(x1 - x0, y1 - y0) * 0.32
    draw.pieslice((cx - s * 1.4, cy - s, cx + s * 0.2, cy + s * 1.6), 200, 340, fill=(24, 100, 198))
    draw.pieslice((cx - s * 0.2, cy - s * 1.6, cx + s * 1.4, cy + s), 20, 160, fill=(38, 132, 255))


def mark_jira(draw: ImageDraw.ImageDraw, plate: tuple[int, int, int, int]) -> None:
    """Stacked chevrons — Jira-ish mark."""
    x0, y0, x1, y1 = plate
    cx, cy = (x0 + x1) / 2, (y0 + y1) / 2
    s = min(x1 - x0, y1 - y0) * 0.22
    blues = [(0, 82, 204), (38, 132, 255), (87, 157, 255)]
    for i, col in enumerate(blues):
        oy = cy - s * 1.1 + i * s * 0.85
        draw.polygon(
            [
                (cx, oy - s),
                (cx + s * 1.15, oy + s * 0.35),
                (cx, oy + s * 0.15),
                (cx - s * 1.15, oy + s * 0.35),
            ],
            fill=col,
        )


MARK_DRAWERS = {
    "slack": mark_slack,
    "meet": mark_meet,
    "github": mark_github,
    "confluence": mark_confluence,
    "jira": mark_jira,
}


def ensure_integration_tiles() -> Path:
    """Write matching framed PNGs under docs/assets/integrations/ (reusable)."""
    INTEGRATION_DIR.mkdir(parents=True, exist_ok=True)
    for key, _label, frame in INTEGRATION_SPECS:
        out = INTEGRATION_DIR / f"{key}.png"
        if out.is_file():
            continue
        framed_tile(MARK_DRAWERS[key], frame).save(out, "PNG", optimize=True)
        print(f"Wrote {out}")
    return INTEGRATION_DIR


def paste_logo_row(
    canvas: Image.Image,
    draw: ImageDraw.ImageDraw,
    items: list[tuple[Image.Image, str]],
    center_y: int,
    caption: str,
    tile: int = 48,
    gap: int = 14,
) -> None:
    n = len(items)
    total_w = n * tile + (n - 1) * gap
    x0 = (W - total_w) // 2
    y0 = center_y - tile // 2 - 6

    cf = mono_font(11)
    cb = cf.getbbox(caption)
    draw.text(
        ((W - (cb[2] - cb[0])) // 2 - cb[0], y0 - 18 - cb[1]),
        caption,
        fill=SIGNAL,
        font=cf,
    )

    lf = mono_font(10)
    for i, (tile_img, label) in enumerate(items):
        x = x0 + i * (tile + gap)
        canvas.paste(tile_img, (x, y0), tile_img)
        lb = lf.getbbox(label)
        lw = lb[2] - lb[0]
        draw.text(
            (x + (tile - lw) // 2 - lb[0], y0 + tile + 4 - lb[1]),
            label,
            fill=QUIET,
            font=lf,
        )


def paste_logo_strips(canvas: Image.Image, draw: ImageDraw.ImageDraw, footer_y: int) -> None:
    missing = [name for name, _ in PROVIDER_LOGOS if not (PROVIDER_DIR / name).is_file()]
    if missing:
        raise SystemExit(f"Missing provider logos in {PROVIDER_DIR}: {', '.join(missing)}")

    ensure_integration_tiles()

    tile = 46
    models = [
        (load_provider_tile(PROVIDER_DIR / name, tile), label)
        for name, label in PROVIDER_LOGOS
    ]
    integrations = [
        (load_provider_tile(INTEGRATION_DIR / f"{key}.png", tile), label)
        for key, label, _ in INTEGRATION_SPECS
    ]

    # Two quiet readout rows above footer
    paste_logo_row(canvas, draw, models, center_y=footer_y - 168, caption="MODELS", tile=tile, gap=12)
    paste_logo_row(
        canvas, draw, integrations, center_y=footer_y - 72, caption="CONNECTED", tile=tile, gap=18
    )


def compose(cfg: dict) -> Path:
    canvas = Image.new("RGB", (W, H), FIELD)
    draw = ImageDraw.Draw(canvas)

    margin_x = 56
    brand_max_w = W - 2 * margin_x
    feature_lines = cfg.get("features") or []
    has_features = bool(feature_lines)
    has_sub = bool(cfg.get("sub"))
    has_logos = bool(cfg.get("logos"))

    line1, line2 = "NEURAL", "JUNKIE"
    # Compact brand when stacking readout + two logo rows
    brand_start = 118 if has_features else 168
    f1 = fit_width(line1, brand_max_w, brand_start, floor=64 if has_features else 48)
    f2 = fit_width(line2, brand_max_w, brand_start, floor=64 if has_features else 48)

    b1 = f1.getbbox(line1)
    b2 = f2.getbbox(line2)
    w1, h1 = b1[2] - b1[0], b1[3] - b1[1]
    w2, h2 = b2[2] - b2[0], b2[3] - b2[1]
    gap = 4 if has_features else 8
    block_h = h1 + gap + h2
    if has_features:
        brand_center = 0.135
    elif has_sub:
        brand_center = 0.20
    else:
        brand_center = 0.22
    brand_top = int(H * brand_center) - block_h // 2

    draw.text(
        (margin_x + (brand_max_w - w1) // 2 - b1[0], brand_top - b1[1]),
        line1,
        fill=INK,
        font=f1,
    )
    draw.text(
        (margin_x + (brand_max_w - w2) // 2 - b2[0], brand_top + h1 + gap - b2[1]),
        line2,
        fill=INK,
        font=f2,
    )

    if has_features:
        carrier_y = brand_top + block_h + 28
    elif has_sub:
        carrier_y = int(H * 0.54)
    else:
        carrier_y = int(H * 0.58)
    draw.line((margin_x, carrier_y, W - margin_x, carrier_y), fill=SIGNAL, width=2)
    cx = W // 2
    diamond = [
        (cx, carrier_y - 7),
        (cx + 7, carrier_y),
        (cx, carrier_y + 7),
        (cx - 7, carrier_y),
    ]
    draw.polygon(diamond, fill=SIGNAL)
    tick = 10
    draw.line((margin_x, carrier_y - tick, margin_x, carrier_y + tick), fill=SIGNAL, width=2)
    draw.line((W - margin_x, carrier_y - tick, W - margin_x, carrier_y + tick), fill=SIGNAL, width=2)

    support = cfg["support"]
    sf = mono_font(20 if has_features else (26 if len(support) > 28 else 28))
    sb = sf.getbbox(support)
    sw = sb[2] - sb[0]
    y = carrier_y + 24 if has_features else carrier_y + 36
    draw.text(((W - sw) // 2 - sb[0], y - sb[1]), support, fill=INK, font=sf)

    if has_sub:
        sub = cfg["sub"]
        uf = mono_font(20)
        ub = uf.getbbox(sub)
        uw = ub[2] - ub[0]
        draw.text(
            ((W - uw) // 2 - ub[0], y + 42 - ub[1]),
            sub,
            fill=QUIET,
            font=uf,
        )

    if has_features:
        ff_feat = mono_font(15)
        row_gap = 24
        feat_top = y + 32
        for i, line in enumerate(feature_lines):
            fb = ff_feat.getbbox(line)
            fw = fb[2] - fb[0]
            draw.text(
                ((W - fw) // 2 - fb[0], feat_top + i * row_gap - fb[1]),
                line,
                fill=QUIET,
                font=ff_feat,
            )

    footer_meta = cfg["footer"]
    ff = mono_font(14 if len(footer_meta) > 36 else 15)
    fb = ff.getbbox(footer_meta)
    fw = fb[2] - fb[0]
    footer_y = H - 88
    draw.text(
        ((W - fw) // 2 - fb[0], footer_y - fb[1]),
        footer_meta,
        fill=QUIET,
        font=ff,
    )

    rf = mono_font(13)
    rb = rf.getbbox(RELEASE)
    rw = rb[2] - rb[0]
    draw.text(
        ((W - rw) // 2 - rb[0], footer_y + 26 - rb[1]),
        RELEASE,
        fill=QUIET,
        font=rf,
    )

    id_font = mono_font(14)
    draw.text((margin_x, 40), cfg["corner"], fill=SIGNAL, font=id_font)

    # Grain behind logos so marks stay crisp
    canvas = add_grain(canvas, amount=12)
    draw = ImageDraw.Draw(canvas)

    if has_logos:
        paste_logo_strips(canvas, draw, footer_y=footer_y)

    out = OUT_DIR / cfg["out"]
    canvas.save(out, "PNG", optimize=True)
    print(f"Wrote {out}")
    return out


wanted = list(VARIANTS) if VARIANT == "all" else [VARIANT]
if VARIANT not in ("all", *VARIANTS):
    raise SystemExit(f"Unknown variant {VARIANT!r}; use baseline|team|features|all")

for key in wanted:
    compose(VARIANTS[key])
PY
