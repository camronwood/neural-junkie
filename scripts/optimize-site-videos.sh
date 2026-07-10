#!/usr/bin/env bash
# Re-encode marketing MP4s for GitHub Pages (H.264 + AAC, web-friendly).
# Usage: ./scripts/optimize-site-videos.sh <src.mp4> <dest.mp4>
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <src.mp4> <dest.mp4>" >&2
  exit 1
fi

src="$1"
dest="$2"
mkdir -p "$(dirname "$dest")"

ffmpeg -y -i "$src" \
  -c:v libx264 -preset slow -crf 23 -profile:v high -level 4.1 \
  -pix_fmt yuv420p -movflags +faststart \
  -vf "scale='min(1920,iw)':-2" \
  -c:a aac -b:a 128k -ac 2 \
  "$dest"

echo "Wrote $(du -h "$dest" | cut -f1) → $dest"
