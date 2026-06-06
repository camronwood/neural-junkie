#!/usr/bin/env bash
# Verify Specialist tuning bootstrap LoRA presets compose in Ollama.
# Usage: ./scripts/verify-bootstrap-loras.sh [--skip-pull]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SKIP_PULL=0
if [[ "${1:-}" == "--skip-pull" ]]; then
  SKIP_PULL=1
fi

if ! command -v ollama >/dev/null 2>&1; then
  echo "ollama not found in PATH" >&2
  exit 1
fi

# agent_type|repo_id|base_ollama_tag|ollama_tag
PRESETS=(
  "security|scthornton/llama-3.2-3b-securecode|llama3.2:3b|nj-security:14b"
  "code-review|juzhengz/LoRI-D_code_llama3_rank_64|llama3:8b|nj-code-review:14b"
  "backend|visheshgupta/mistral-7b-text2sql-qlora|mistral:7b|nj-backend:14b"
  "biology|Pk3112/medmcqa-lora-llama3-8b-instruct|llama3:8b|nj-biology:8b"
)

pull_base() {
  local base="$1"
  if [[ "$SKIP_PULL" -eq 1 ]]; then
    return 0
  fi
  echo "📥 ollama pull $base"
  ollama pull "$base"
}

download_adapter() {
  local repo="$1"
  local dest="$2"
  mkdir -p "$dest"
  python3 - <<PY
import sys
from pathlib import Path
try:
    from huggingface_hub import hf_hub_download
except ImportError:
    import subprocess
    subprocess.check_call([sys.executable, "-m", "pip", "install", "-q", "huggingface_hub"])
    from huggingface_hub import hf_hub_download

repo = "${repo}"
dest = Path("${dest}")
for name in ("adapter_model.safetensors", "adapter_config.json"):
    path = hf_hub_download(repo_id=repo, filename=name, local_dir=str(dest))
    print(f"  downloaded {name} -> {path}")
PY
}

compose_one() {
  local agent="$1" repo="$2" base="$3" tag="$4"
  echo ""
  echo "=== $agent ($repo) base=$base tag=$tag ==="
  pull_base "$base"

  local work
  work="$(mktemp -d "${TMPDIR:-/tmp}/nj-lora-verify.XXXXXX")"
  trap 'rm -rf "$work"' RETURN

  download_adapter "$repo" "$work"

  go run ./cmd/verify-bootstrap-lora/main.go \
    -base "$base" \
    -adapter "$work" \
    -tag "$tag"
}

pass=0
fail=0
for row in "${PRESETS[@]}"; do
  IFS='|' read -r agent repo base tag <<<"$row"
  if compose_one "$agent" "$repo" "$base" "$tag"; then
    echo "✅ PASS $tag"
    pass=$((pass + 1))
  else
    echo "❌ FAIL $tag"
    fail=$((fail + 1))
  fi
done

echo ""
echo "Results: $pass passed, $fail failed"
[[ "$fail" -eq 0 ]]
