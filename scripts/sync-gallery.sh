#!/usr/bin/env bash
# Copy campaign creatives + screenshots into docs/media/gallery and refresh manifest.json
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GALLERY="${ROOT}/docs/media/gallery"
CAMPAIGNS="${ROOT}/campaigns"
SHOTS_SRC="${ROOT}/assets/screenshots"

mkdir -p "${GALLERY}/ads" "${GALLERY}/screenshots" "${GALLERY}/misc"

shopt -s nullglob
# Prefer ad squares / named ads; also sync campaign creatives that match ad/cover patterns.
for f in "${CAMPAIGNS}"/*/creatives/*-ad-*.png \
         "${CAMPAIGNS}"/*/creatives/*-ad-1080.png \
         "${CAMPAIGNS}"/*/creatives/ide-v4-*.png \
         "${CAMPAIGNS}"/*/creatives/edge-ide-*.png; do
  [[ -f "$f" ]] || continue
  cp -f "$f" "${GALLERY}/ads/"
done

# Deduped copy of remaining campaign PNGs that look like published ads/covers
# (already-copied files are overwritten with the same content — fine).
for f in "${CAMPAIGNS}"/*/creatives/*.png; do
  [[ -f "$f" ]] || continue
  base="$(basename "$f")"
  case "$base" in
    *-ad-*.png|*-1200.png|ide-v4-*.png|edge-ide-*.png) cp -f "$f" "${GALLERY}/ads/" ;;
  esac
done

if [[ -d "${SHOTS_SRC}" ]]; then
  for f in "${SHOTS_SRC}"/*.{png,jpg,jpeg,webp}; do
    [[ -f "$f" ]] || continue
    cp -f "$f" "${GALLERY}/screenshots/"
  done
fi

python3 "${ROOT}/scripts/generate-gallery-manifest.py"
echo "Gallery ready — open docs/gallery/index.html or deploy to GitHub Pages"
