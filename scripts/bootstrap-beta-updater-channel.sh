#!/usr/bin/env bash
#
# Bootstrap git-backed beta updater manifests from a published release.
#
# Usage: ./scripts/bootstrap-beta-updater-channel.sh [version-tag] [repo]
#   version-tag: defaults to latest beta prerelease

set -euo pipefail

VERSION="${1:-}"
REPO="${2:-camronwood/neural-junkie}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

if [[ -z "${VERSION}" ]]; then
  VERSION="$(gh release list --repo "${REPO}" --limit 20 --json tagName,isPrerelease \
    -q '.[] | select(.isPrerelease) | .tagName' | head -1)"
fi
if [[ -z "${VERSION}" ]]; then
  echo "No beta release found; pass a version tag explicitly." >&2
  exit 1
fi

echo "Syncing beta updater manifests from ${VERSION}..."
cd "${ROOT}"
chmod +x scripts/publish-updater-manifests.sh
# Upload to versioned release (no-op if already present) and sync updater/beta/.
./scripts/publish-updater-manifests.sh "${VERSION}" "${REPO}"

echo "Beta updater channel:"
echo "  https://raw.githubusercontent.com/${REPO}/main/updater/beta/update-darwin-aarch64.json"
