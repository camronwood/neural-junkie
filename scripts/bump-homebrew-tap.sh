#!/usr/bin/env bash
#
# Regenerate Casks/neural-junkie.rb (macOS) and Formula/neural-junkie.rb (Linux)
# in the homebrew tap from a GitHub release tag.
#
# Usage:
#   ./scripts/bump-homebrew-tap.sh v1.2.0-beta.6 [tap-dir]
#
# Default tap-dir: ../homebrew-tap (sibling of neural-junkie repo)

set -euo pipefail

TAG="${1:?Usage: $0 <version-tag> [tap-dir]}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAP_DIR="${2:-${ROOT}/../homebrew-tap}"
CASK_FILE="${TAP_DIR}/Casks/neural-junkie.rb"
FORMULA_FILE="${TAP_DIR}/Formula/neural-junkie.rb"
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

mkdir -p "${TAP_DIR}/Formula"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

sha_for_dmg() {
  local suffix="$1"
  local asset="Neural.Junkie_${VERSION}_${suffix}.dmg"
  local url="https://github.com/${REPO}/releases/download/${TAG}/${asset}"
  echo "Fetching ${asset}..." >&2
  curl -fsSL -o "${tmpdir}/${asset}" "${url}"
  shasum -a 256 "${tmpdir}/${asset}" | awk '{print $1}'
}

sha_for_deb() {
  local asset="neural-junkie_${VERSION}_amd64.deb"
  local url="https://github.com/${REPO}/releases/download/${TAG}/${asset}"
  echo "Fetching ${asset}..." >&2
  curl -fsSL -o "${tmpdir}/${asset}" "${url}"
  shasum -a 256 "${tmpdir}/${asset}" | awk '{print $1}'
}

SHA_ARM="$(sha_for_dmg aarch64)"
SHA_INTEL="$(sha_for_dmg x64)"
SHA_LINUX="$(sha_for_deb)"

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

cat >"${FORMULA_FILE}" <<EOF
class NeuralJunkie < Formula
  desc "Multi-agent orchestration workspace with local-first AI"
  homepage "https://camronwood.github.io/neural-junkie/"
  version "${VERSION}"
  license :cannot_represent

  on_macos do
    odie <<~EOS
      Neural Junkie on macOS is installed as a cask:

        brew tap camronwood/tap
        brew install --cask neural-junkie
    EOS
  end

  on_linux do
    url "https://github.com/${REPO}/releases/download/v#{version}/neural-junkie_#{version}_amd64.deb"
    sha256 "${SHA_LINUX}"

    depends_on "at-spi2-core"
    depends_on "cairo"
    depends_on "dbus"
    depends_on "gdk-pixbuf"
    depends_on "glib"
    depends_on "gtk+3"
    depends_on "librsvg"
    depends_on "pango"
    depends_on "webkit2gtk"

    def install
      extract = buildpath/"deb-extract"
      extract.mkpath
      cd extract do
        system "ar", "x", cached_download
        system "tar", "-xzf", "data.tar.gz"
        cd "usr" do
          bin.install "bin/neural-junkie", "bin/nj-server"
          (lib/"neural-junkie").install Dir["lib/neural-junkie/*"]
          (share/"applications").install "share/applications/neural-junkie.desktop"
          (share/"icons").install "share/icons/hicolor"
        end
      end
    end

    def caveats
      <<~EOS
        First launch runs the setup wizard (install Ollama via wizard if needed — internet required).
        User data: ~/.neural-junkie
      EOS
    end
  end
end
EOF

echo "Wrote ${CASK_FILE}"
echo "  version ${VERSION}"
echo "  arm64   ${SHA_ARM}"
echo "  x64     ${SHA_INTEL}"
echo "Wrote ${FORMULA_FILE}"
echo "  linux   ${SHA_LINUX}"
