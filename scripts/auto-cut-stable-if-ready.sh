#!/usr/bin/env bash
# Cut v1.2.0 stable when consecutive away mornings are green + ship-bar package PASS.
#
# Usage:
#   ./scripts/auto-cut-stable-if-ready.sh           # dry-run checks
#   ./scripts/auto-cut-stable-if-ready.sh --execute # tag + push when ready
#   AUTO_STABLE_CUT=1  (from away-overnight) implies --execute when green
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

EXECUTE=0
TAG="${STABLE_CUT_TAG:-v1.2.0}"
CONSECUTIVE="${CONSECUTIVE_GREEN:-2}"

for arg in "$@"; do
  case "${arg}" in
    --execute) EXECUTE=1 ;;
    --dry-run) EXECUTE=0 ;;
  esac
done

if [[ "${AUTO_STABLE_CUT:-}" == "1" ]]; then
  EXECUTE=1
fi

chmod +x \
  scripts/check-consecutive-away-green.py \
  scripts/ship-bar-package.sh \
  scripts/cut-stable-release.sh \
  scripts/verify-desktop-version-consistency.sh

echo "=== consecutive away green (n=${CONSECUTIVE}) ==="
if ! python3 scripts/check-consecutive-away-green.py --count "${CONSECUTIVE}"; then
  echo "Not ready: need ${CONSECUTIVE} consecutive green away-morning reports."
  exit 0
fi

echo "=== ship-bar package ==="
./scripts/ship-bar-package.sh

# Already tagged?
if git rev-parse "${TAG}" >/dev/null 2>&1; then
  echo "Tag ${TAG} already exists — nothing to cut."
  exit 0
fi

if [[ "${EXECUTE}" != "1" ]]; then
  echo "Ready to cut ${TAG}. Re-run with --execute or AUTO_STABLE_CUT=1."
  ./scripts/cut-stable-release.sh --tag "${TAG}"
  exit 0
fi

echo "Cutting stable ${TAG}…"
./scripts/cut-stable-release.sh --execute --tag "${TAG}"

# Best-effort site bump (previous beta tip)
PREV="$(git tag -l 'v*-beta.*' --sort=-v:refname | head -1 || true)"
if [[ -n "${PREV}" ]]; then
  chmod +x scripts/update-website-release.sh || true
  ./scripts/update-website-release.sh "${TAG}" --bump-site "${PREV}" || \
    echo "WARN: website bump failed; run manually after CI" >&2
fi

chmod +x scripts/verify-updater-manifest.sh || true
echo "After release CI: ./scripts/verify-updater-manifest.sh ${TAG} stable"
echo "auto-cut-stable: tagged ${TAG}"
