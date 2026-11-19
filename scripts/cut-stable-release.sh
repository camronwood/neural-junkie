#!/usr/bin/env bash
# Maintainer helper: cut v1.0.0 stable after STABLE_RELEASE_CHECKLIST gates pass.
#
# Usage:
#   ./scripts/cut-stable-release.sh           # dry-run (shows steps)
#   ./scripts/cut-stable-release.sh --execute # tag, push, refresh site
#
# Prerequisites:
#   - docs/STABLE_RELEASE_CHECKLIST.md Definition of ready signed off
#   - macOS ad-hoc install smoke PASS (notarization optional for v1.0.0; target v1.0.1)
#   - gh authenticated, on main, clean tree
set -euo pipefail

EXECUTE=false
TAG="v1.2.0"
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
PREV_BETA="$(git tag -l 'v*-beta.*' --sort=-v:refname | head -1 || echo v1.2.0-beta.34)"
echo "Pre-flight (automated path: make ship-bar + 2× green away mornings):"
echo "  1. docs/PRE_STABLE_RELIABILITY.md + docs/STABLE_RELEASE_CHECKLIST.md"
echo "  2. make ship-bar-package / verify-updater-manifest on last beta"
echo "  3. Interactive Gate 5 replaced by ship-bar (KNOWN_ISSUES #18)"
echo "  4. macOS remains ad-hoc until notarization secrets in CI (v1.2.1)"
echo ""

if [[ "${EXECUTE}" != "true" ]]; then
  echo "Dry run. Re-run with --execute to:"
  echo "  git tag -a ${TAG} -m \"Neural Junkie ${TAG} stable\""
  echo "  git push origin ${TAG}"
  echo "  ./scripts/update-website-release.sh ${TAG} --bump-site ${PREV_BETA}"
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
