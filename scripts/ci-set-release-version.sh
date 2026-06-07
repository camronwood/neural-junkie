#!/usr/bin/env bash
#
# Sets package.version in tauri.conf.json from a release tag.
# Beta tags map to WiX-safe 4-part versions (1.0.0-beta.26 -> 1.0.0.26).
#
# Usage: ./scripts/ci-set-release-version.sh v1.0.0-beta.25

set -euo pipefail

TAG="${1:?Usage: $0 <version-tag>}"
CONF="desktop/src-tauri/tauri.conf.json"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# shellcheck source=scripts/release-bundle-version.sh
source "${ROOT}/scripts/release-bundle-version.sh"

BUNDLE_VERSION="$(bundle_version_from_tag "${TAG}")"

if [[ ! -f "${CONF}" ]]; then
  echo "Missing ${CONF}" >&2
  exit 1
fi

node -e "
const fs = require('fs');
const confPath = process.argv[1];
const version = process.argv[2];
const conf = JSON.parse(fs.readFileSync(confPath, 'utf8'));
conf.package.version = version;
fs.writeFileSync(confPath, JSON.stringify(conf, null, 2) + '\n');
" "${CONF}" "${BUNDLE_VERSION}"

echo "Set ${CONF} package.version to ${BUNDLE_VERSION} (from tag ${TAG})"
