#!/usr/bin/env bash
# Compose Walk Into a Room — Week 2, seven 1080×1080 joke cards.
# Usage: ./scripts/compose-walk-into-a-room-ads.sh [1|2|3|4|5|6|7|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
VARIANT="${1:-all}"
PLATES="$ROOT/campaigns/walk-into-a-room/plates"
OUT_DIR="$ROOT/campaigns/walk-into-a-room/creatives"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

mkdir -p "$OUT_DIR"

exec "$PY" - "$PLATES" "$OUT_DIR" "$VARIANT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont, ImageEnhance

PLATES = Path(sys.argv[1])
OUT_DIR = Path(sys.argv[2])
VARIANT = sys.argv[3]

W, H = 1080, 1080
PHOTO_H = 700
PANEL = (18, 16, 14)
CREAM = (236, 228, 214)
AMBER = (212, 160, 72)
MUTED = (160, 148, 132)


def serif(bold: bool, italic: bool, size: int):
    if italic and not bold:
        paths = [
            "/System/Library/Fonts/Supplemental/Georgia Italic.ttf",
            "/System/Library/Fonts/Supplemental/Times New Roman Italic.ttf",
        ]
    elif bold:
        paths = [
            "/System/Library/Fonts/Supplemental/Georgia Bold.ttf",
            "/System/Library/Fonts/Supplemental/Times New Roman Bold.ttf",
        ]
    else:
        paths = [
            "/System/Library/Fonts/Supplemental/Georgia.ttf",
            "/System/Library/Fonts/Supplemental/Times New Roman.ttf",
        ]
    for path in paths:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def cover_crop(im: Image.Image, tw: int, th: int) -> Image.Image:
    im = im.convert("RGB")
    sw, sh = im.size
    scale = max(tw / sw, th / sh)
    nw, nh = int(sw * scale), int(sh * scale)
    im = im.resize((nw, nh), Image.Resampling.LANCZOS)
    left = (nw - tw) // 2
    top = max((nh - th) // 2 - 10, 0)
    cropped = im.crop((left, top, left + tw, top + th))
    cropped = ImageEnhance.Color(cropped).enhance(0.9)
    cropped = ImageEnhance.Contrast(cropped).enhance(1.05)
    return cropped


def wrap(draw, text: str, fnt, max_w: int):
    words = text.split()
    lines, cur = [], ""
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


def card(day: int, plate: str, setup: str, punch: str, out_name: str):
    photo = cover_crop(Image.open(PLATES / plate), W, PHOTO_H)
    canvas = Image.new("RGB", (W, H), PANEL)
    canvas.paste(photo, (0, 0))
    draw = ImageDraw.Draw(canvas)
    draw.rectangle((0, PHOTO_H, W, PHOTO_H + 3), fill=AMBER)

    y = PHOTO_H + 28
    draw.text((56, y), f"WALK INTO A ROOM  ·  {day:02d} / 07", fill=AMBER, font=serif(False, False, 16))
    y += 44

    setup_font = serif(True, False, 28 if len(setup) > 42 else 32)
    for line in wrap(draw, setup, setup_font, W - 112):
        draw.text((56, y), line, fill=CREAM, font=setup_font)
        y += 36 if len(setup) > 42 else 40

    y += 12
    punch_font = serif(False, True, 22)
    for line in wrap(draw, punch, punch_font, W - 112):
        draw.text((56, y), line, fill=MUTED, font=punch_font)
        y += 30

    draw.text((56, H - 48), "neural junkie", fill=CREAM, font=serif(False, False, 15))
    draw.text((W - 56, H - 48), "week two", fill=MUTED, font=serif(False, True, 15), anchor="rm")

    path = OUT_DIR / out_name
    canvas.save(path, "PNG")
    print(f"Wrote {path}")


DAYS = {
    "1": dict(
        plate="01-tabs.png",
        setup="A ChatGPT tab, a Claude tab, and a Notes app walk into a room.",
        punch="Nobody introduces anybody.",
        out="day-01-tabs-1080.png",
    ),
    "2": dict(
        plate="02-agent.png",
        setup="An agent walks into a room.",
        punch="It rewrites the files and leaves.",
        out="day-02-agent-1080.png",
    ),
    "3": dict(
        plate="03-models.png",
        setup="Qwen and Kimi walk into a room.",
        punch="They're both good. They don't share a desk.",
        out="day-03-models-1080.png",
    ),
    "4": dict(
        plate="04-hats.png",
        setup="A generalist walks into a room wearing six hats.",
        punch="Security still sounds like the backend.",
        out="day-04-hats-1080.png",
    ),
    "5": dict(
        plate="05-invoice.png",
        setup="A draft and a review walk into a room.",
        punch="Same invoice.",
        out="day-05-invoice-1080.png",
    ),
    "6": dict(
        plate="06-argue.png",
        setup="Two agents walk into a room and agree immediately.",
        punch="That's not a meeting. That's a mirror.",
        out="day-06-argue-1080.png",
    ),
    "7": dict(
        plate="07-you.png",
        setup="You walk into a room.",
        punch="The models are already seated. The no is still yours.",
        out="day-07-you-1080.png",
    ),
}

if VARIANT == "all":
    keys = list(DAYS)
elif VARIANT in DAYS:
    keys = [VARIANT]
else:
    raise SystemExit(f"unknown variant: {VARIANT}")

for key in keys:
    spec = DAYS[key]
    card(int(key), spec["plate"], spec["setup"], spec["punch"], spec["out"])
PY
