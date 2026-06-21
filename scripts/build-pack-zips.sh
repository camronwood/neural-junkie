#!/usr/bin/env bash
# Build release zip bundles from official pack repos (siblings under projects/ next to neural-junkie).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROJECTS_ROOT="$(cd "${ROOT}/.." && pwd)"
OUT="${ROOT}/dist/packs"
mkdir -p "$OUT"

repos=(
  neural-junkie-pack-software-development:software-development
  neural-junkie-pack-life-sciences:life-sciences
  neural-junkie-pack-cad:cad
  neural-junkie-pack-specialist-tuning:specialist-tuning
  neural-junkie-pack-aws:aws
  neural-junkie-pack-incident-management:incident-management
  neural-junkie-pack-web-browser:web-browser
)

for spec in "${repos[@]}"; do
  repo="${spec%%:*}"
  id="${spec##*:}"
  src="${PROJECTS_ROOT}/${repo}"
  if [[ ! -d "${src}" ]]; then
    echo "skip ${repo} (not found at ${src})" >&2
    continue
  fi
  if [[ -f "${src}/Makefile" ]]; then
    make -C "${src}" pack-zip
    cp "${src}/dist/${id}-"*.zip "${OUT}/" 2>/dev/null || cp "${src}/dist/"*.zip "${OUT}/"
  else
    (cd "${src}" && zip -r "${OUT}/${id}-1.0.0.zip" pack.yaml -x '*.DS_Store')
  fi
  echo "Wrote ${OUT}/${id}-*.zip"
done

echo ""
echo "Upload dist/packs/*.zip to each pack repo GitHub release (tag v1.0.0)."
echo "See docs/PACKS.md"
