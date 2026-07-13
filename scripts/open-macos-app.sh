#!/usr/bin/env bash
# Remove macOS quarantine and open Neural Junkie (free Gatekeeper workaround).
set -euo pipefail

APP="${1:-/Applications/Neural Junkie.app}"

if [[ ! -d "$APP" ]]; then
  echo "App not found: $APP" >&2
  echo "Usage: $0 [/path/to/Neural Junkie.app]" >&2
  exit 1
fi

if ! command -v xattr >/dev/null 2>&1; then
  echo "xattr not found — this script is for macOS." >&2
  exit 1
fi

echo "Clearing quarantine on: $APP"
xattr -dr com.apple.quarantine "$APP" 2>/dev/null || true

echo "Opening app..."
open "$APP"
