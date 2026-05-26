#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Skipping macOS bundle signature verification on $(uname -s)."
  exit 0
fi

apps=()
if [[ $# -gt 0 ]]; then
  for path in "$@"; do
    apps+=("$path")
  done
else
  shopt -s nullglob
  apps=(desktop/src-tauri/target/*/release/bundle/macos/*.app desktop/src-tauri/target/release/bundle/macos/*.app)
  shopt -u nullglob
fi

if [[ ${#apps[@]} -eq 0 ]]; then
  echo "No macOS .app bundles found to verify." >&2
  exit 1
fi

for app in "${apps[@]}"; do
  if [[ ! -d "$app" ]]; then
    echo "macOS .app bundle not found: $app" >&2
    exit 1
  fi

  echo "Verifying code signature: $app"
  codesign --verify --deep --strict --verbose=4 "$app"
done
