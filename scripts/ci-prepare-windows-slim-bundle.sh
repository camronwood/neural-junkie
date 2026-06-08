#!/usr/bin/env bash
#
# Windows release installers omit bundled Ollama — the full windows-amd64
# runtime pushes .msi over GitHub's 2 GiB asset limit and is slow to update.
# The setup wizard auto-installs Ollama on first launch (winget or OllamaSetup.exe).
#
# Usage: ./scripts/ci-prepare-windows-slim-bundle.sh

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WINDOWS_OLLAMA="${ROOT}/desktop/src-tauri/ollama/x86_64-pc-windows-msvc"

rm -rf "${WINDOWS_OLLAMA}"
mkdir -p "${WINDOWS_OLLAMA}"

cat > "${WINDOWS_OLLAMA}/README.txt" <<'TXT'
Neural Junkie Windows releases do not bundle the Ollama runtime.

On first launch, choose Local Models in the setup wizard and click Install Ollama
(internet required). Or install from https://ollama.com manually.

Or use a cloud API in Settings → AI Providers.
TXT

echo "Prepared slim Windows bundle (no bundled Ollama under ${WINDOWS_OLLAMA})"
