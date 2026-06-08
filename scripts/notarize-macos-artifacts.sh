#!/usr/bin/env bash
# Notarize and staple macOS .app bundles and .dmg installers (CI or local).
# Requires: APPLE_ID, APPLE_APP_SPECIFIC_PASSWORD (or notarytool profile),
#           APPLE_TEAM_ID, and a Developer ID-signed app/dmg.
#
# Usage:
#   ./scripts/notarize-macos-artifacts.sh path/to/App.app [path/to/App.dmg ...]
#
# Skips when APPLE_ID is unset.
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Skipping macOS notarization on $(uname -s)."
  exit 0
fi

if [[ -z "${APPLE_ID:-}" ]]; then
  echo "APPLE_ID not set — skipping notarization."
  exit 0
fi

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path.app|path.dmg> [...]" >&2
  exit 1
fi

NOTARY_ARGS=()
if [[ -n "${APPLE_APP_SPECIFIC_PASSWORD:-}" ]]; then
  NOTARY_ARGS+=(--apple-id "${APPLE_ID}" --password "${APPLE_APP_SPECIFIC_PASSWORD}" --team-id "${APPLE_TEAM_ID:?APPLE_TEAM_ID required}")
elif [[ -n "${NOTARY_KEYCHAIN_PROFILE:-}" ]]; then
  NOTARY_ARGS+=(--keychain-profile "${NOTARY_KEYCHAIN_PROFILE}")
else
  echo "Set APPLE_APP_SPECIFIC_PASSWORD or NOTARY_KEYCHAIN_PROFILE for notarization." >&2
  exit 1
fi

submit_and_staple() {
  local artifact="$1"
  if [[ ! -e "${artifact}" ]]; then
    echo "Artifact not found: ${artifact}" >&2
    return 1
  fi

  local submit_path="${artifact}"
  local tmp_zip=""
  if [[ "${artifact}" == *.app ]]; then
    tmp_zip="$(mktemp -t nj-notarize).zip"
    ditto -c -k --keepParent "${artifact}" "${tmp_zip}"
    submit_path="${tmp_zip}"
  fi

  echo "Submitting to notarytool: ${artifact}"
  xcrun notarytool submit "${submit_path}" "${NOTARY_ARGS[@]}" --wait

  if [[ -n "${tmp_zip}" ]]; then
    rm -f "${tmp_zip}"
  fi

  echo "Stapling: ${artifact}"
  xcrun stapler staple "${artifact}"
}

for path in "$@"; do
  submit_and_staple "${path}"
done

echo "Notarization complete."
