#!/usr/bin/env bash
#
# Regenerate Casks/neural-junkie.rb in the homebrew tap from a GitHub release tag.
#
# Usage:
#   ./scripts/bump-homebrew-cask.sh v1.2.0-beta.5 [tap-dir]
#
# Default tap-dir: ../homebrew-tap (sibling of neural-junkie repo)

set -euo pipefail

TAG="${1:?Usage: $0 <version-tag> [tap-dir]}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAP_DIR="${2:-${ROOT}/../homebrew-tap}"
CASK_FILE="${TAP_DIR}/Casks/neural-junkie.rb"
REPO="${NJ_RELEASE_REPO:-camronwood/neural-junkie}"

if [[ "${TAG}" != v* ]]; then
  echo "Tag must start with v (got: ${TAG})" >&2
  exit 1
fi

VERSION="${TAG#v}"

if [[ ! -d "${TAP_DIR}/Casks" ]]; then
  echo "Tap Casks directory not found: ${TAP_DIR}/Casks" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

sha_for() {
  local suffix="$1"
  local asset="Neural.Junkie_${VERSION}_${suffix}.dmg"
  local url="https://github.com/${REPO}/releases/download/${TAG}/${asset}"
  echo "Fetching ${asset}..." >&2
  curl -fsSL -o "${tmpdir}/${asset}" "${url}"
  shasum -a 256 "${tmpdir}/${asset}" | awk '{print $1}'
}

SHA_ARM="$(sha_for aarch64)"
SHA_INTEL="$(sha_for x64)"

cat >"${CASK_FILE}" <<EOF
cask "neural-junkie" do
  version "${VERSION}"

  on_arm do
    sha256 "${SHA_ARM}"

    url "https://github.com/${REPO}/releases/download/v#{version}/Neural.Junkie_#{version}_aarch64.dmg",
        verified: "github.com/${REPO}/"
  end
  on_intel do
    sha256 "${SHA_INTEL}"

    url "https://github.com/${REPO}/releases/download/v#{version}/Neural.Junkie_#{version}_x64.dmg",
        verified: "github.com/${REPO}/"
  end

  name "Neural Junkie"
  desc "Multi-agent orchestration workspace with local-first AI"
  homepage "https://camronwood.github.io/neural-junkie/"

  depends_on macos: :big_sur

  app "Neural Junkie.app"

  zap trash: "~/.neural-junkie"
end
EOF

echo "Wrote ${CASK_FILE}"
echo "  version ${VERSION}"
echo "  arm64   ${SHA_ARM}"
echo "  x64     ${SHA_INTEL}"
