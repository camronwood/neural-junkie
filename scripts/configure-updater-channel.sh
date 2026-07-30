#!/usr/bin/env bash
#
# Patches tauri.conf.json updater endpoints for beta or stable channel.
#
# Usage: ./scripts/configure-updater-channel.sh beta|stable

set -euo pipefail

CHANNEL="${1:?Usage: $0 beta|stable}"
CONF="desktop/src-tauri/tauri.conf.json"
REPO="camronwood/neural-junkie"
BETA_MANIFEST_BASE="https://raw.githubusercontent.com/${REPO}/main/updater/beta"

if [[ ! -f "${CONF}" ]]; then
  echo "Missing ${CONF}" >&2
  exit 1
fi

case "${CHANNEL}" in
  stable)
    endpoints_json='["https://github.com/'"${REPO}"'/releases/latest/download/update-{{target}}-{{arch}}.json"]'
    ;;
  beta)
    # Rolling beta manifests live in git (updater/beta/) — immutable GitHub releases
    # cannot replace assets, and the legacy updater-beta release tag is gone (404).
    # Use only the git-backed URL so checks do not depend on a dead first endpoint.
    endpoints_json='["'"${BETA_MANIFEST_BASE}"'/update-{{target}}-{{arch}}.json"]'
    ;;
  *)
    echo "Unknown channel: ${CHANNEL} (expected beta or stable)" >&2
    exit 1
    ;;
esac

node -e "
const fs = require('fs');
const confPath = process.argv[1];
const endpoints = JSON.parse(process.argv[2]);
const conf = JSON.parse(fs.readFileSync(confPath, 'utf8'));
conf.plugins ??= {};
conf.plugins.updater ??= {};
conf.plugins.updater.endpoints = endpoints;
fs.writeFileSync(confPath, JSON.stringify(conf, null, 2) + '\n');
" "${CONF}" "${endpoints_json}"

echo "Configured updater channel=${CHANNEL}"
echo "${endpoints_json}" | node -e "JSON.parse(require('fs').readFileSync(0,'utf8')).forEach((u,i)=>console.log('  endpoint['+i+']='+u))"
