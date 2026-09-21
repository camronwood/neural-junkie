#!/usr/bin/env bash
# Compose On Air — 1080×1080 station-ID heroes (procedural, no plates).
# Usage: ./scripts/compose-on-air-ad.sh [baseline|team|all]
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

exec "$PY" - "$OUT_DIR" "$VARIANT" <<'PY'
import random
import sys
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

OUT_DIR = Path(sys.argv[1])
VARIANT = sys.argv[2]
W, H = 1080, 1080

FIELD = (13, 15, 20)       # #0d0f14
SIGNAL = (233, 69, 96)     # #e94560
INK = (232, 234, 239)      # #e8eaef
QUIET = (139, 147, 167)    # #8b93a7
RELEASE = "github.com/camronwood/neural-junkie/releases/latest"

# baseline = brand callsign; team = differentiators beyond “runs local”
VARIANTS = {
    "baseline": {
        "out": "on-air-hero-1080.png",
        "corner": "NJ  //  ON AIR",
        "support": "Any model. On yours.",
        "sub": None,
        "footer": "OPEN SOURCE  ·  LOCAL FIRST",
    },
    "team": {
        "out": "on-air-team-1080.png",
        "corner": "NJ  //  ON AIR  ·  TEAM",
        "support": "Not another local chat.",
        "sub": "Specialists. Approvals. Your models.",
        "footer": "MULTI-AGENT  ·  HUMAN-IN-THE-LOOP  ·  BYOM",
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


def compose(cfg: dict) -> Path:
    canvas = Image.new("RGB", (W, H), FIELD)
    draw = ImageDraw.Draw(canvas)

    margin_x = 56
    brand_max_w = W - 2 * margin_x

    line1, line2 = "NEURAL", "JUNKIE"
    f1 = fit_width(line1, brand_max_w, 168)
    f2 = fit_width(line2, brand_max_w, 168)

    b1 = f1.getbbox(line1)
    b2 = f2.getbbox(line2)
    w1, h1 = b1[2] - b1[0], b1[3] - b1[1]
    w2, h2 = b2[2] - b2[0], b2[3] - b2[1]
    gap = 8
    block_h = h1 + gap + h2
    # Slightly higher brand when a sub-line is present so the block still balances
    brand_center = 0.20 if cfg["sub"] else 0.22
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

    carrier_y = int(H * 0.54) if cfg["sub"] else int(H * 0.58)
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
    sf = mono_font(26 if len(support) > 28 else 28)
    sb = sf.getbbox(support)
    sw = sb[2] - sb[0]
    y = carrier_y + 36
    draw.text(((W - sw) // 2 - sb[0], y - sb[1]), support, fill=INK, font=sf)

    if cfg["sub"]:
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

    canvas = add_grain(canvas, amount=12)
    out = OUT_DIR / cfg["out"]
    canvas.save(out, "PNG", optimize=True)
    print(f"Wrote {out}")
    return out


wanted = list(VARIANTS) if VARIANT == "all" else [VARIANT]
if VARIANT not in ("all", *VARIANTS):
    raise SystemExit(f"Unknown variant {VARIANT!r}; use baseline|team|all")

for key in wanted:
    compose(VARIANTS[key])
PY
