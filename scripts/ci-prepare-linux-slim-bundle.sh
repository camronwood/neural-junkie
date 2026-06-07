#!/usr/bin/env bash
#
# Linux release installers omit bundled Ollama — the full linux-amd64
# runtime (GPU libs) pushes .deb/.AppImage over GitHub's 2 GiB asset limit.
# Users install Ollama separately (https://ollama.com) or use a cloud provider;
# the hub detects system ollama via PATH (see internal/ollama/manager.go).
#
# Usage: ./scripts/ci-prepare-linux-slim-bundle.sh

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LINUX_OLLAMA="${ROOT}/desktop/src-tauri/ollama/x86_64-unknown-linux-gnu"

rm -rf "${LINUX_OLLAMA}"
mkdir -p "${LINUX_OLLAMA}"

cat > "${LINUX_OLLAMA}/README.txt" <<'TXT'
Neural Junkie Linux releases do not bundle the Ollama runtime (GitHub asset size limit).

Install Ollama before first use:
  curl -fsSL https://ollama.com/install.sh | sh

Or use a cloud API in Settings → AI Providers.
TXT

echo "Prepared slim Linux bundle (no bundled Ollama under ${LINUX_OLLAMA})"
