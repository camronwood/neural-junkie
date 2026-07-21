#!/usr/bin/env bash
#
# Generates and uploads immutable updater manifests for a release.
# Rolling channel pointers are advanced only after the release is public.
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

echo "Updater manifests published."
