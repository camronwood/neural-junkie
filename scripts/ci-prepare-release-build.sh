#!/usr/bin/env bash
#
# Prepares tauri.conf.json for a release build (version + updater channel).
#
# Usage: ./scripts/ci-prepare-release-build.sh v1.0.0-beta.25

set -euo pipefail

TAG="${1:?Usage: $0 <version-tag>}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "${ROOT}"
chmod +x scripts/ci-set-release-version.sh scripts/configure-updater-channel.sh

./scripts/ci-set-release-version.sh "${TAG}"

if [[ "${TAG}" == *beta* ]]; then
  ./scripts/configure-updater-channel.sh beta
else
  ./scripts/configure-updater-channel.sh stable
fi
