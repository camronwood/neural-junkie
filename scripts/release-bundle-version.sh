#!/usr/bin/env bash
#
# Maps git release tags to Tauri bundle versions (WiX-safe) and display semver.
#
# Usage:
#   source scripts/release-bundle-version.sh
#   bundle_version_from_tag v1.0.0-beta.26  # -> 1.0.0.26
#   display_version_from_tag v1.0.0-beta.26 # -> 1.0.0-beta.26

bundle_version_from_tag() {
  local tag="${1#v}"

  if [[ "${tag}" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)-beta\.([0-9]+)$ ]]; then
    echo "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${BASH_REMATCH[3]}.${BASH_REMATCH[4]}"
    return 0
  fi

  echo "${tag}"
}

display_version_from_tag() {
  local tag="${1#v}"
  echo "${tag}"
}
