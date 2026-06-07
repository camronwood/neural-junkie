#!/usr/bin/env bash
#
# Sets package.version in tauri.conf.json from a release tag.
# Called by CI before tauri build so updater semver checks match the tag.
#
# Usage: ./scripts/ci-set-release-version.sh v1.0.0-beta.25

set -euo pipefail

TAG="${1:?Usage: $0 <version-tag>}"
VERSION="${TAG#v}"
CONF="desktop/src-tauri/tauri.conf.json"

if [[ ! -f "${CONF}" ]]; then
  echo "Missing ${CONF}" >&2
  exit 1
fi

node -e "
const fs = require('fs');
const confPath = process.argv[1];
const version = process.argv[2];
const conf = JSON.parse(fs.readFileSync(confPath, 'utf8'));
conf.package.version = version;
fs.writeFileSync(confPath, JSON.stringify(conf, null, 2) + '\n');
" "${CONF}" "${VERSION}"

echo "Set ${CONF} package.version to ${VERSION}"
