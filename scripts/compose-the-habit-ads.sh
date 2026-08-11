#!/usr/bin/env bash
# Compose The Habit — seven 1080×1080 gallery cards (photo plate + didactic panel).
# Usage: ./scripts/compose-the-habit-ads.sh [1|2|3|4|5|6|7|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="$ROOT/.venv-icon/bin/python"
VARIANT="${1:-all}"
PLATES="$ROOT/campaigns/the-habit/plates"
OUT_DIR="$ROOT/campaigns/the-habit/creatives"

if [[ ! -x "$PY" ]]; then
  python3 -m venv "$ROOT/.venv-icon"
  "$ROOT/.venv-icon/bin/pip" install -q Pillow
fi

mkdir -p "$OUT_DIR"

exec "$PY" - "$PLATES" "$OUT_DIR" "$VARIANT" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont, ImageEnhance, ImageFilter

PLATES = Path(sys.argv[1])
OUT_DIR = Path(sys.argv[2])
VARIANT = sys.argv[3]

W, H = 1080, 1080
PHOTO_H = 720
PANEL_H = H - PHOTO_H
INK = (22, 18, 14)
PAPER = (236, 228, 214)
RULE = (168, 42, 36)
MUTED = (92, 82, 72)


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
    top = max((nh - th) // 2 - 20, 0)
    cropped = im.crop((left, top, left + tw, top + th))
    cropped = ImageEnhance.Color(cropped).enhance(0.88)
    cropped = ImageEnhance.Contrast(cropped).enhance(1.06)
    return cropped


def card(day: int, plate: str, title: str, dek: str, out_name: str):
    photo = cover_crop(Image.open(PLATES / plate), W, PHOTO_H)
    canvas = Image.new("RGB", (W, H), PAPER)
    canvas.paste(photo, (0, 0))

    # hairline between plate and panel
    draw = ImageDraw.Draw(canvas)
    draw.rectangle((0, PHOTO_H, W, PHOTO_H + 3), fill=RULE)

    y = PHOTO_H + 36
    acc = f"THE HABIT  ·  {day:02d} / 07"
    draw.text((64, y), acc, fill=RULE, font=serif(False, False, 18))
    y += 52
    draw.text((64, y), title, fill=INK, font=serif(True, False, 48))
    y += 62
    draw.text((64, y), dek, fill=MUTED, font=serif(False, True, 24))
    draw.text((64, H - 52), "neural junkie", fill=INK, font=serif(False, False, 16))
    draw.text((W - 64, H - 52), "a working habit", fill=MUTED, font=serif(False, True, 16), anchor="rm")

    path = OUT_DIR / out_name
    canvas.save(path, "PNG")
    print(f"Wrote {path}")


DAYS = {
    "1": dict(
        plate="01-named.png",
        title="YOU NAMED IT.",
        dek="A regular, not a website.",
        out="day-01-named-1080.png",
    ),
    "2": dict(
        plate="02-room.png",
        title="SAME ROOM.",
        dek="A new chat is amnesia.",
        out="day-02-room-1080.png",
    ),
    "3": dict(
        plate="03-no.png",
        title="KEEP THE NO.",
        dek="Patience is the whole craft.",
        out="day-03-no-1080.png",
    ),
    "4": dict(
        plate="04-building.png",
        title="IT NEVER LEFT.",
        dek="Some work is a room problem.",
        out="day-04-building-1080.png",
    ),
    "5": dict(
        plate="05-tool.png",
        title="ONE JOB.",
        dek="Don't send a generalist.",
        out="day-05-tool-1080.png",
    ),
    "6": dict(
        plate="06-argue.png",
        title="LET THEM ARGUE.",
        dek="Agreement is cheap.",
        out="day-06-argue-1080.png",
    ),
    "7": dict(
        plate="07-place.png",
        title="A PLACE FOR IT.",
        dek="The habit needed a door.",
        out="day-07-place-1080.png",
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
    card(int(key), spec["plate"], spec["title"], spec["dek"], spec["out"])
PY
