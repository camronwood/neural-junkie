#!/usr/bin/env bash
# Regenerate docs/articles/ from campaigns/* LinkedIn article sources.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="${ROOT}/.venv-icon/bin/python"
if [[ ! -x "$PY" ]]; then
  python3 -m venv "${ROOT}/.venv-icon"
  "${ROOT}/.venv-icon/bin/pip" install -q markdown
fi
"$PY" "${ROOT}/scripts/generate-articles-site.py"
python3 "${ROOT}/scripts/sync-site-nav.py"
echo "Articles ready — covers in docs/media/articles/covers/, pages in docs/articles/"
