#!/usr/bin/env bash
# Fetch Ollama runtime binaries for Tauri bundle (install-and-go local AI).
# Usage:
#   ./scripts/fetch-ollama.sh                    # current host triple
#   ./scripts/fetch-ollama.sh aarch64-apple-darwin x86_64-unknown-linux-gnu
#   OLLAMA_VERSION=v0.32.0 ./scripts/fetch-ollama.sh all
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_ROOT="${ROOT}/desktop/src-tauri/ollama"
VERSION="${OLLAMA_VERSION:-v0.32.0}"
BASE_URL="https://github.com/ollama/ollama/releases/download/${VERSION}"

extract_tar_zst() {
  local archive="$1"
  local dest="$2"
  if command -v zstd >/dev/null 2>&1; then
    zstd -d "${archive}" -o "${dest}/ollama.tar"
    tar -xf "${dest}/ollama.tar" -C "${dest}"
    rm -f "${dest}/ollama.tar"
    return
  fi
  if python3 -c "import zstandard" 2>/dev/null; then
    python3 - <<PY
import tarfile, zstandard, pathlib
archive = pathlib.Path("${archive}")
dest = pathlib.Path("${dest}")
raw = archive.read_bytes()
data = zstandard.ZstdDecompressor().decompress(raw)
tar_path = dest / "ollama.tar"
tar_path.write_bytes(data)
with tarfile.open(tar_path) as tf:
    tf.extractall(dest)
tar_path.unlink()
PY
    return
  fi
  echo "❌ Need zstd or python3+zstandard to extract ${archive}" >&2
  exit 1
}

fetch_target() {
  local triple="$1"
  local dest="${OUT_ROOT}/${triple}"
  rm -rf "${dest}"
  mkdir -p "${dest}"

  case "${triple}" in
    aarch64-apple-darwin|x86_64-apple-darwin)
      echo "⬇️  ${triple}: ollama-darwin.tgz"
      local tgz="${dest}/ollama.tgz"
      curl -fsSL "${BASE_URL}/ollama-darwin.tgz" -o "${tgz}"
      tar -xzf "${tgz}" -C "${dest}"
      rm -f "${tgz}"
      chmod +x "${dest}/ollama"
      ;;
    x86_64-unknown-linux-gnu)
      echo "⬇️  ${triple}: ollama-linux-amd64.tar.zst (large — GPU libs included)"
      local archive="${dest}/ollama-linux-amd64.tar.zst"
      curl -fsSL "${BASE_URL}/ollama-linux-amd64.tar.zst" -o "${archive}"
      extract_tar_zst "${archive}" "${dest}"
      rm -f "${archive}"
      chmod +x "${dest}/bin/ollama"
      ;;
    x86_64-pc-windows-msvc)
      echo "⬇️  ${triple}: ollama-windows-amd64.zip (large — GPU libs included)"
      local zip="${dest}/ollama.zip"
      curl -fsSL "${BASE_URL}/ollama-windows-amd64.zip" -o "${zip}"
      if command -v unzip >/dev/null 2>&1; then
        unzip -q "${zip}" -d "${dest}"
      else
        python3 -c "import zipfile; zipfile.ZipFile('${zip}').extractall('${dest}')"
      fi
      rm -f "${zip}"
      ;;
    *)
      echo "❌ Unknown target triple: ${triple}" >&2
      return 1
      ;;
  esac

  echo "✅ ${triple} → ${dest}"
}

ALL_TARGETS=(
  aarch64-apple-darwin
  x86_64-apple-darwin
  x86_64-unknown-linux-gnu
  x86_64-pc-windows-msvc
)

host_triple() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"
  case "${os}-${arch}" in
    Darwin-arm64) echo "aarch64-apple-darwin" ;;
    Darwin-x86_64) echo "x86_64-apple-darwin" ;;
    Linux-x86_64) echo "x86_64-unknown-linux-gnu" ;;
    MINGW*|MSYS*|CYGWIN*) echo "x86_64-pc-windows-msvc" ;;
    *)
      echo "❌ Unsupported host ${os}/${arch}" >&2
      exit 1
      ;;
  esac
}

if [[ $# -eq 0 ]]; then
  fetch_target "$(host_triple)"
elif [[ $# -eq 1 && "$1" == "all" ]]; then
  for t in "${ALL_TARGETS[@]}"; do
    fetch_target "${t}"
  done
else
  for t in "$@"; do
    fetch_target "${t}"
  done
fi

echo "📦 Ollama ${VERSION} runtimes ready under ${OUT_ROOT}"
