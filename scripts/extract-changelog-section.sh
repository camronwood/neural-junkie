#!/usr/bin/env bash
#
# Extract the CHANGELOG.md section for a release tag.
#
# Usage:
#   ./scripts/extract-changelog-section.sh v1.2.0-beta.3
#   ./scripts/extract-changelog-section.sh v1.2.0-beta.3 path/to/CHANGELOG.md
#
# Prints the section body (without the ## header) to stdout.
# Exit 0 when found, 1 when missing.

set -euo pipefail

TAG="${1:?Usage: $0 <tag> [changelog-path]}"
TAG="${TAG#v}"
CHANGELOG="${2:-docs/CHANGELOG.md}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHANGELOG_PATH="${ROOT}/${CHANGELOG}"

if [[ ! -f "${CHANGELOG_PATH}" ]]; then
  echo "changelog not found: ${CHANGELOG_PATH}" >&2
  exit 1
fi

found=0
while IFS= read -r line || [[ -n "${line}" ]]; do
  if [[ "${line}" == "## ["* ]]; then
    section="${line#*[}"
    section="${section%%]*}"
    if [[ "${found}" -eq 1 ]]; then
      break
    fi
    if [[ "${section}" == "${TAG}" ]]; then
      found=1
      continue
    fi
  elif [[ "${found}" -eq 1 ]]; then
    printf '%s\n' "${line}"
  fi
done < "${CHANGELOG_PATH}"

if [[ "${found}" -eq 0 ]]; then
  echo "no changelog section for ${TAG}" >&2
  exit 1
fi
