#!/usr/bin/env bash
#
# Generates and uploads updater manifests for a release.
# Creates/maintains the rolling updater-beta prerelease for beta tags.
#
# Usage: ./scripts/publish-updater-manifests.sh v1.0.0-beta.25 [repo]

set -euo pipefail

VERSION="${1:?Usage: $0 <version-tag> [repo]}"
REPO="${2:-camronwood/neural-junkie}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

chmod +x scripts/generate-update-manifests.sh

./scripts/generate-update-manifests.sh "${VERSION}" "${REPO}"

shopt -s nullglob
manifests=(update-*.json)
if [[ ${#manifests[@]} -eq 0 ]]; then
  echo "No manifests generated" >&2
  exit 1
fi

echo "Uploading manifests to ${VERSION}..."
gh release upload "${VERSION}" "${manifests[@]}" --repo "${REPO}" --clobber

if [[ "${VERSION}" == *beta* ]]; then
  echo "Publishing beta channel manifests to updater-beta..."
  if ! gh release view updater-beta --repo "${REPO}" >/dev/null 2>&1; then
    gh release create updater-beta \
      --repo "${REPO}" \
      --title "Beta updater channel" \
      --notes "Rolling updater manifests for beta builds. Do not install manually." \
      --prerelease
  fi
  gh release upload updater-beta "${manifests[@]}" --repo "${REPO}" --clobber
fi

echo "Updater manifests published."
