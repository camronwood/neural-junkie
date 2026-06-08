#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Skipping macOS bundle signature verification on $(uname -s)."
  exit 0
fi

REQUIRE_DEVELOPER_ID="${MACOS_REQUIRE_DEVELOPER_ID:-0}"
if [[ -n "${APPLE_SIGNING_IDENTITY:-}" && "${APPLE_SIGNING_IDENTITY}" != "-" ]]; then
  REQUIRE_DEVELOPER_ID=1
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

  if [[ "${REQUIRE_DEVELOPER_ID}" == "1" ]]; then
    authority="$(codesign -dv --verbose=4 "$app" 2>&1 | grep 'Authority=' | head -1 || true)"
    if [[ "${authority}" != *"Developer ID Application"* ]]; then
      echo "Expected Developer ID Application signature when APPLE_SIGNING_IDENTITY is set." >&2
      echo "${authority}" >&2
      exit 1
    fi
    if ! xcrun stapler validate "$app" >/dev/null 2>&1; then
      echo "Notarization staple missing on $app (run notarize-macos-artifacts.sh)." >&2
      exit 1
    fi
    echo "Developer ID + notarization staple OK: $app"
  fi
done
