#!/usr/bin/env bash
#
# Sets the Tauri v2 top-level version from a release tag.
# Windows uses WiX-safe semver (1.0.0-beta.26 -> 1.0.0-26).
#
# Usage: ./scripts/ci-set-release-version.sh v1.0.0-beta.25 [windows]

set -euo pipefail

TAG="${1:?Usage: $0 <version-tag> [windows]}"
PLATFORM="${2:-}"
CONF="desktop/src-tauri/tauri.conf.json"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# shellcheck source=scripts/release-bundle-version.sh
source "${ROOT}/scripts/release-bundle-version.sh"

BUNDLE_VERSION="$(bundle_version_from_tag "${TAG}" "${PLATFORM}")"

if [[ ! -f "${CONF}" ]]; then
  echo "Missing ${CONF}" >&2
  exit 1
fi

node -e "
const fs = require('fs');
const confPath = process.argv[1];
const pkgPath = process.argv[2];
const cargoPath = process.argv[3];
const version = process.argv[4];
const conf = JSON.parse(fs.readFileSync(confPath, 'utf8'));
conf.version = version;
fs.writeFileSync(confPath, JSON.stringify(conf, null, 2) + '\n');
const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
pkg.version = version;
fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
let cargo = fs.readFileSync(cargoPath, 'utf8');
cargo = cargo.replace(/^version = \".*\"/m, 'version = \"' + version + '\"');
fs.writeFileSync(cargoPath, cargo);
" "${CONF}" "${ROOT}/desktop/package.json" "${ROOT}/desktop/src-tauri/Cargo.toml" "${BUNDLE_VERSION}"

echo "Set desktop versions to ${BUNDLE_VERSION} (from tag ${TAG}, platform=${PLATFORM:-default})"
