#!/usr/bin/env bash
#
# Maps git release tags to Tauri bundle versions.
#
# Usage:
#   source scripts/release-bundle-version.sh
#   bundle_version_from_tag v1.0.0-beta.26           # -> 1.0.0-beta.26
#   bundle_version_from_tag v1.0.0-beta.26 windows   # -> 1.0.0-26 (WiX-safe semver)

bundle_version_from_tag() {
  local tag="${1#v}"
  local platform="${2:-}"

  if [[ "${tag}" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)-beta\.([0-9]+)$ ]]; then
    if [[ "${platform}" == "windows" ]]; then
      echo "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${BASH_REMATCH[3]}-${BASH_REMATCH[4]}"
    else
      echo "${tag}"
    fi
    return 0
  fi

  echo "${tag}"
}

display_version_from_tag() {
  local tag="${1#v}"
  echo "${tag}"
}
