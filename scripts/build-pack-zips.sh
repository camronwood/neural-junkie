#!/usr/bin/env bash
# Build release zip bundles for official domain packs (manifest + assets).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/dist/packs"
mkdir -p "$OUT"
for id in software-development life-sciences; do
  src="${ROOT}/internal/packs/builtin/${id}"
  (cd "$src" && zip -r "${OUT}/${id}-1.0.0.zip" .)
  echo "Wrote ${OUT}/${id}-1.0.0.zip"
done
echo ""
echo "Upload dist/packs/*.zip to GitHub release tag packs-v1.0.0 on camronwood/neural-junkie"
echo "See docs/PACKS.md"
