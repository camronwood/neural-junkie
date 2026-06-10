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
required=(update-darwin-aarch64.json update-darwin-x86_64.json update-windows-x86_64.json)
if [[ ${#manifests[@]} -eq 0 ]]; then
  echo "No manifests generated" >&2
  exit 1
fi
for need in "${required[@]}"; do
  if [[ ! -f "${need}" ]]; then
    echo "Missing required manifest: ${need}" >&2
    exit 1
  fi
done
if [[ ! -f update-linux-x86_64.json ]]; then
  echo "WARN: Linux updater manifest skipped (AppImage bundle not published yet)" >&2
fi

echo "Uploading manifests to ${VERSION}..."
gh release upload "${VERSION}" "${manifests[@]}" --repo "${REPO}" --clobber

if [[ "${VERSION}" == *beta* ]]; then
  echo "Publishing beta channel manifests to updater-beta..."
  # Prefer upload-only: repo rules may block delete/recreate of immutable updater-beta tag.
  if gh release view updater-beta --repo "${REPO}" >/dev/null 2>&1; then
    echo "updater-beta release exists — uploading manifests (--clobber)"
    gh release upload updater-beta "${manifests[@]}" --repo "${REPO}" --clobber
  else
    echo "Creating updater-beta rolling release (first time)"
    if ! gh release create updater-beta \
      --repo "${REPO}" \
      --title "Beta updater channel" \
      --notes "Rolling updater manifests for beta builds. Do not install manually." \
      --prerelease 2>&1; then
      echo "WARN: Could not create updater-beta release (repo rules may block tag creation)." >&2
      echo "WARN: Beta in-app updates may not work until updater-beta exists; versioned manifests were uploaded to ${VERSION}." >&2
    else
      gh release upload updater-beta "${manifests[@]}" --repo "${REPO}" --clobber
    fi
  fi
fi

echo "Updater manifests published."
