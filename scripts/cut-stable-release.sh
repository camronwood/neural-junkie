#!/usr/bin/env bash
# Maintainer helper: cut v1.0.0 stable after STABLE_RELEASE_CHECKLIST gates pass.
#
# Usage:
#   ./scripts/cut-stable-release.sh           # dry-run (shows steps)
#   ./scripts/cut-stable-release.sh --execute # tag, push, refresh site
#
# Prerequisites:
#   - docs/STABLE_RELEASE_CHECKLIST.md gates signed off
#   - Apple notarization verified on a beta tag first
#   - gh authenticated, on main, clean tree
set -euo pipefail

EXECUTE=false
TAG="v1.0.0"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --execute) EXECUTE=true; shift ;;
    --tag) TAG="${2:?}"; shift 2 ;;
    *) echo "Unknown: $1" >&2; exit 1 ;;
  esac
done

cd "${ROOT}"

echo "Stable release cut: ${TAG}"
echo ""
echo "Pre-flight (run manually if not done):"
echo "  1. Review docs/STABLE_RELEASE_CHECKLIST.md — all gates PASS"
echo "  2. ./scripts/verify-updater-manifest.sh v1.0.0-beta.33 beta"
echo "  3. macOS notarized .dmg verified on clean machine"
echo ""

if [[ "${EXECUTE}" != "true" ]]; then
  echo "Dry run. Re-run with --execute to:"
  echo "  git tag -a ${TAG} -m \"Neural Junkie ${TAG} stable\""
  echo "  git push origin ${TAG}"
  echo "  ./scripts/update-website-release.sh ${TAG} --bump-site v1.0.0-beta.33"
  echo "  ./scripts/verify-updater-manifest.sh ${TAG} stable"
  exit 0
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Working tree not clean. Commit or stash first." >&2
  exit 1
fi

git tag -a "${TAG}" -m "Neural Junkie ${TAG} stable"
git push origin "${TAG}"

echo "Waiting for release CI (check GitHub Actions)..."
echo "After CI completes:"
echo "  ./scripts/verify-updater-manifest.sh ${TAG} stable"
echo "  ./scripts/update-website-release.sh ${TAG} --bump-site v1.0.0-beta.33"
