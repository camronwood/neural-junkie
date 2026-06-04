#!/usr/bin/env bash
# Build customer pack zip(s) for sideload install.
# Brightest Bio pack: use neural-junkie-brightest-bio-lab repo (make pack-zip).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BB_PACK="${NEURAL_JUNKIE_BB_PACK:-$(cd "${ROOT}/../.." && pwd)/neural-junkie-brightest-bio-lab}"
if [[ -f "${BB_PACK}/scripts/build-pack-zip.sh" ]]; then
  exec "${BB_PACK}/scripts/build-pack-zip.sh"
fi
echo "Set NEURAL_JUNKIE_BB_PACK to a customer pack repo, or run make pack-zip there." >&2
echo "Example: ${ROOT}/../neural-junkie-brightest-bio-lab" >&2
exit 1
