#!/usr/bin/env bash
# Import Apple Developer ID certificate into a CI keychain (macOS only).
# Skips when APPLE_CERTIFICATE_BASE64 is unset (ad-hoc signing fallback).
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Skipping Apple certificate import on $(uname -s)."
  exit 0
fi

if [[ -z "${APPLE_CERTIFICATE_BASE64:-}" ]]; then
  echo "APPLE_CERTIFICATE_BASE64 not set — using ad-hoc macOS signing."
  exit 0
fi

KEYCHAIN="${RUNNER_TEMP:-/tmp}/app-signing.keychain-db"
KEYCHAIN_PASSWORD="${KEYCHAIN_PASSWORD:-actions}"

echo "${APPLE_CERTIFICATE_BASE64}" | base64 --decode > /tmp/certificate.p12

security create-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN}"
security set-keychain-settings -lut 21600 "${KEYCHAIN}"
security unlock-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN}"
security import /tmp/certificate.p12 -P "${APPLE_CERTIFICATE_PASSWORD:-}" -A -t cert -f pkcs12 -k "${KEYCHAIN}"
security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "${KEYCHAIN_PASSWORD}" "${KEYCHAIN}"
security list-keychains -d user -s "${KEYCHAIN}" login.keychain-db

rm -f /tmp/certificate.p12
echo "Apple certificate imported into ${KEYCHAIN}"
