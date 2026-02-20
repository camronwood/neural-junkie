#!/usr/bin/env bash
#
# Generates Tauri updater JSON manifests for each platform.
# Called by CI after the build job uploads release artifacts.
#
# Usage: ./scripts/generate-update-manifests.sh v1.2.0

set -euo pipefail

VERSION="${1:?Usage: $0 <version-tag>}"
VERSION_NUM="${VERSION#v}"
REPO="camronwood/neural-junkie"
RELEASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
PUB_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Fetch release assets to find signatures
ASSETS=$(gh release view "${VERSION}" --repo "${REPO}" --json assets -q '.assets[].name' 2>/dev/null || echo "")

generate_manifest() {
    local platform="$1"
    local arch="$2"
    local ext="$3"
    local output_file="update-${platform}-${arch}.json"

    # Find the artifact URL
    local artifact_pattern=""
    case "${platform}-${arch}" in
        darwin-aarch64)
            artifact_pattern="Neural.Junkie_${VERSION_NUM}_aarch64.app.tar.gz"
            ;;
        darwin-x86_64)
            artifact_pattern="Neural.Junkie_${VERSION_NUM}_x64.app.tar.gz"
            ;;
        linux-x86_64)
            artifact_pattern="neural-junkie_${VERSION_NUM}_amd64.AppImage.tar.gz"
            ;;
    esac

    local sig_file="${artifact_pattern}.sig"
    local signature=""

    # Try to download the signature file
    if echo "${ASSETS}" | grep -q "${sig_file}"; then
        signature=$(gh release download "${VERSION}" --repo "${REPO}" --pattern "${sig_file}" --output - 2>/dev/null || echo "")
    fi

    cat > "${output_file}" <<MANIFEST
{
  "version": "${VERSION_NUM}",
  "notes": "See release notes at https://github.com/${REPO}/releases/tag/${VERSION}",
  "pub_date": "${PUB_DATE}",
  "platforms": {
    "${platform}-${arch}": {
      "signature": "${signature}",
      "url": "${RELEASE_URL}/${artifact_pattern}"
    }
  }
}
MANIFEST

    echo "Generated ${output_file}"
}

generate_manifest "darwin" "aarch64" "app.tar.gz"
generate_manifest "darwin" "x86_64" "app.tar.gz"
generate_manifest "linux" "x86_64" "AppImage.tar.gz"

echo "All manifests generated."
