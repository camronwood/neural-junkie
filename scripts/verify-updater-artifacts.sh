#!/usr/bin/env bash
#
# Verifies signed updater bundles exist after tauri build.
#
# Usage:
#   ./scripts/verify-updater-artifacts.sh macos-aarch64 <bundle-dir>
#   ./scripts/verify-updater-artifacts.sh macos-x86_64 <bundle-dir>
#   ./scripts/verify-updater-artifacts.sh linux <bundle-dir>
#   ./scripts/verify-updater-artifacts.sh windows <bundle-dir>

set -euo pipefail

PLATFORM="${1:?Usage: $0 macos-aarch64|macos-x86_64|linux|windows <bundle-dir>}"
DIR="${2:?Usage: $0 <platform> <bundle-dir>}"

if [[ ! -d "${DIR}" ]]; then
  echo "Bundle directory not found: ${DIR}" >&2
  exit 1
fi

shopt -s nullglob

case "${PLATFORM}" in
  macos-aarch64|macos-x86_64)
    bundles=("${DIR}"/*.tar.gz)
    ;;
  linux)
    bundles=("${DIR}"/*.AppImage.tar.gz)
    ;;
  windows)
    bundles=("${DIR}"/*.msi.zip)
    ;;
  *)
    echo "Unknown platform: ${PLATFORM}" >&2
    exit 1
    ;;
esac

if [[ ${#bundles[@]} -eq 0 ]]; then
  echo "No updater bundle found in ${DIR} for ${PLATFORM}" >&2
  exit 1
fi

missing=0
for bundle in "${bundles[@]}"; do
  sig="${bundle}.sig"
  if [[ ! -f "${sig}" ]]; then
    echo "Missing signature: ${sig}" >&2
    missing=1
  else
    echo "OK: ${bundle} + ${sig}"
  fi
done

if [[ "${missing}" -ne 0 ]]; then
  exit 1
fi

echo "Updater artifacts verified for ${PLATFORM}"
