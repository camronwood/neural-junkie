#!/usr/bin/env bash
# Install LoRA training Python stack into .venv-lora (used by make deps / gui-install).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VENV="$ROOT/.venv-lora"
PY="$VENV/bin/python"

if [[ ! -x "$PY" ]]; then
  echo "📦 Creating LoRA training venv at .venv-lora ..."
  python3 -m venv "$VENV"
fi

echo "📦 Installing LoRA training deps (requirements-lora.txt) ..."
"$PY" -m pip install -q --upgrade pip
"$PY" -m pip install -r "$ROOT/requirements-lora.txt"
echo "✅ LoRA training Python deps ready (.venv-lora)"
