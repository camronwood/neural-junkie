#!/usr/bin/env bash
#
# Patches tauri.conf.json updater endpoints for beta or stable channel.
#
# Usage: ./scripts/configure-updater-channel.sh beta|stable

set -euo pipefail

CHANNEL="${1:?Usage: $0 beta|stable}"
CONF="desktop/src-tauri/tauri.conf.json"
REPO="camronwood/neural-junkie"

if [[ ! -f "${CONF}" ]]; then
  echo "Missing ${CONF}" >&2
  exit 1
fi

case "${CHANNEL}" in
  stable)
    endpoint="https://github.com/${REPO}/releases/latest/download/update-{{target}}-{{arch}}.json"
    ;;
  beta)
    endpoint="https://github.com/${REPO}/releases/download/updater-beta/update-{{target}}-{{arch}}.json"
    ;;
  *)
    echo "Unknown channel: ${CHANNEL} (expected beta or stable)" >&2
    exit 1
    ;;
esac

node -e "
const fs = require('fs');
const confPath = process.argv[1];
const endpoint = process.argv[2];
const conf = JSON.parse(fs.readFileSync(confPath, 'utf8'));
conf.tauri.updater.endpoints = [endpoint];
fs.writeFileSync(confPath, JSON.stringify(conf, null, 2) + '\n');
" "${CONF}" "${endpoint}"

echo "Configured updater channel=${CHANNEL} endpoint=${endpoint}"
