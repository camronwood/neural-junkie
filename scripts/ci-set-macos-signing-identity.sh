#!/usr/bin/env bash
# Set tauri.conf.json macOS signingIdentity for release CI.
# Uses APPLE_SIGNING_IDENTITY when set; otherwise ad-hoc ("-").
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONF="${ROOT}/desktop/src-tauri/tauri.conf.json"
IDENTITY="${APPLE_SIGNING_IDENTITY:--}"

if [[ -n "${APPLE_CERTIFICATE_BASE64:-}" && -n "${APPLE_SIGNING_IDENTITY:-}" ]]; then
  IDENTITY="${APPLE_SIGNING_IDENTITY}"
  echo "Using Developer ID signing identity: ${IDENTITY}"
else
  IDENTITY="-"
  echo "Using ad-hoc signing (no Apple certificate secrets)."
fi

node -e "
const fs = require('fs');
const confPath = process.argv[1];
const identity = process.argv[2];
const conf = JSON.parse(fs.readFileSync(confPath, 'utf8'));
conf.bundle ??= {};
conf.bundle.macOS ??= {};
conf.bundle.macOS.signingIdentity = identity;
fs.writeFileSync(confPath, JSON.stringify(conf, null, 2) + '\n');
" "${CONF}" "${IDENTITY}"
