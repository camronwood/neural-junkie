#!/usr/bin/env bash
# Smoke-checklist for Neural Junkie release installers.
# Usage: ./scripts/smoke-release-install.sh [v1.0.0-beta.1]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-v1.0.0-beta.5}"
REPO="${SMOKE_REPO:-camronwood/neural-junkie}"
HUB_PORT="${HUB_PORT:-18765}"

echo "==> Neural Junkie release smoke: ${VERSION}"
echo "    Repo: ${REPO}"
echo ""

if command -v gh >/dev/null 2>&1; then
  echo "==> GitHub release assets"
  if gh release view "${VERSION}" --repo "${REPO}" >/dev/null 2>&1; then
    gh release view "${VERSION}" --repo "${REPO}" \
      --json name,isPrerelease,publishedAt,assets \
      -q '"\(.name) prerelease=\(.isPrerelease) published=\(.publishedAt)\nAssets:\n" + (.assets[] | "  - \(.name) (\(.size) bytes)\n")'
  else
    echo "WARN: Release ${VERSION} not found on GitHub yet."
    echo "      Run after CI uploads: gh release view ${VERSION} --repo ${REPO}"
  fi
  echo ""
else
  echo "WARN: gh CLI not installed — skip remote asset check"
  echo ""
fi

echo "==> Manual checklist (run on a clean machine after install)"
cat <<'CHECKLIST'

macOS:
  [ ] Mount .dmg (aarch64 or x86_64 for your Mac)
  [ ] Right-click → Open if Gatekeeper blocks (unsigned)
  [ ] App launches; setup wizard appears when no providers configured
  [ ] Complete wizard (Ollama local or cloud API key)
  [ ] Login screen → connect to channel
  [ ] Ask @Moderator "What can Neural Junkie do?" — get a reply
  [ ] Command palette opens; /help runs

Windows:
  [ ] Install .msi or setup .exe
  [ ] Install Ollama from https://ollama.com OR use cloud key in wizard
  [ ] Same wizard → login → Moderator reply flow as macOS

Linux:
  [ ] Run .AppImage or install .deb
  [ ] Same wizard → login → Moderator reply flow

Hub (all platforms):
  [ ] curl -sf "http://localhost:${HUB_PORT}/api/settings" returns JSON (when app running)

CHECKLIST

echo "==> Optional: hub health (if app already running)"
if curl -sf "http://localhost:${HUB_PORT}/api/settings" >/dev/null 2>&1; then
  echo "OK: hub responding on port ${HUB_PORT}"
else
  echo "SKIP: hub not reachable on port ${HUB_PORT} (start the installed app first)"
fi

echo ""
echo "Done. Do not post to social until macOS checklist passes."
