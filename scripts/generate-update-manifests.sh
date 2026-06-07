#!/usr/bin/env bash
#
# Generates Tauri updater JSON manifests for each platform.
# Called by CI after build jobs upload release artifacts.
#
# Usage: ./scripts/generate-update-manifests.sh v1.2.0 [repo]

set -euo pipefail

VERSION="${1:?Usage: $0 <version-tag> [repo]}"
TAG="${VERSION}"
VERSION_NUM="${VERSION#v}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=scripts/release-bundle-version.sh
source "${ROOT}/scripts/release-bundle-version.sh"
REPO="${2:-camronwood/neural-junkie}"
RELEASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
PUB_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

ASSETS=$(gh release view "${VERSION}" --repo "${REPO}" --json assets -q '.assets[].name')

find_asset() {
  local pattern="$1"
  echo "${ASSETS}" | grep -E "${pattern}" | head -n 1 || true
}

fetch_signature() {
  local sig_file="$1"
  if [[ -z "${sig_file}" ]]; then
    echo ""
    return
  fi
  gh release download "${VERSION}" --repo "${REPO}" --pattern "${sig_file}" --output - 2>/dev/null || echo ""
}

write_manifest() {
  local output_file="$1"
  local platform_key="$2"
  local signature="$3"
  local url="$4"

  node -e "
const fs = require('fs');
const payload = {
  version: process.argv[1],
  notes: process.argv[2],
  pub_date: process.argv[3],
  platforms: {
    [process.argv[4]]: {
      signature: process.argv[5],
      url: process.argv[6],
    },
  },
};
fs.writeFileSync(process.argv[7], JSON.stringify(payload, null, 2) + '\n');
" "${manifest_version}" "See release notes at https://github.com/${REPO}/releases/tag/${VERSION}" "${PUB_DATE}" "${platform_key}" "${signature}" "${url}" "${output_file}"
}

generate_manifest() {
  local platform="$1"
  local arch="$2"
  local asset_pattern="$3"
  local output_file="update-${platform}-${arch}.json"
  local manifest_platform=""
  case "${platform}" in
    windows) manifest_platform="windows" ;;
    *) manifest_platform="" ;;
  esac

  local manifest_version
  manifest_version="$(bundle_version_from_tag "${TAG}" "${manifest_platform}")"

  local artifact
  artifact="$(find_asset "${asset_pattern}")"
  if [[ -z "${artifact}" ]]; then
    echo "ERROR: No artifact matching ${asset_pattern} on ${VERSION}; cannot build ${output_file}" >&2
    return 1
  fi

  local signature
  signature="$(fetch_signature "${artifact}.sig")"
  if [[ -z "${signature}" ]]; then
    echo "ERROR: Missing signature for ${artifact}" >&2
    return 1
  fi

  write_manifest "${output_file}" "${platform}-${arch}" "${signature}" "${RELEASE_URL}/${artifact}"
  echo "Generated ${output_file} -> ${artifact}"
}

generate_manifest "darwin" "aarch64" 'Neural\.Junkie_aarch64\.app\.tar\.gz$'
generate_manifest "darwin" "x86_64" 'Neural\.Junkie_x64\.app\.tar\.gz$'
generate_manifest "linux" "x86_64" '.*_amd64\.AppImage\.tar\.gz$'
generate_manifest "windows" "x86_64" '.*\.msi\.zip$'

echo "Manifest generation complete."
