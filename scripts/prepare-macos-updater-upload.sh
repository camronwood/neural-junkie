#!/usr/bin/env bash
#
# Tauri macOS updater bundles are named Neural.Junkie.app.tar.gz for every arch.
# Rename with an arch suffix before gh release upload so both targets coexist.
#
# Usage: ./scripts/prepare-macos-updater-upload.sh <bundle/macos-dir> <aarch64|x64>

set -euo pipefail

DIR="${1:?Usage: $0 <bundle/macos-dir> <aarch64|x64>}"
ARCH="${2:?Usage: $0 <bundle/macos-dir> <aarch64|x64>}"

if [[ ! -d "${DIR}" ]]; then
  echo "Bundle directory not found: ${DIR}" >&2
  exit 1
fi

shopt -s nullglob
bundles=("${DIR}"/*.tar.gz)
if [[ ${#bundles[@]} -eq 0 ]]; then
  echo "No updater .tar.gz found in ${DIR}" >&2
  exit 1
fi

bundle="${bundles[0]}"
if [[ ${#bundles[@]} -gt 1 ]]; then
  echo "WARN: multiple .tar.gz in ${DIR}; using ${bundle}" >&2
fi

out="${DIR}/Neural.Junkie_${ARCH}.app.tar.gz"
cp -f "${bundle}" "${out}"
if [[ -f "${bundle}.sig" ]]; then
  cp -f "${bundle}.sig" "${out}.sig"
else
  echo "Missing signature: ${bundle}.sig" >&2
  exit 1
fi

echo "Prepared ${out} (+ .sig)"
