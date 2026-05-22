#!/usr/bin/env bash
# Copy ads + screenshots into docs/media/gallery and refresh manifest.json
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GALLERY="${ROOT}/docs/media/gallery"
ADS_SRC="${ROOT}/assets"
SHOTS_SRC="${ROOT}/assets/screenshots"

mkdir -p "${GALLERY}/ads" "${GALLERY}/screenshots" "${GALLERY}/misc"

shopt -s nullglob
for f in "${ADS_SRC}"/neural-junkie-*-ad-1080.png "${ADS_SRC}"/neural-junkie-*-ad-*.png; do
  [[ -f "$f" ]] || continue
  cp -f "$f" "${GALLERY}/ads/"
done

if [[ -d "${SHOTS_SRC}" ]]; then
  for f in "${SHOTS_SRC}"/*.{png,jpg,jpeg,webp}; do
    [[ -f "$f" ]] || continue
    cp -f "$f" "${GALLERY}/screenshots/"
  done
fi

python3 "${ROOT}/scripts/generate-gallery-manifest.py"
echo "Gallery ready — open docs/gallery/index.html or deploy to GitHub Pages"
