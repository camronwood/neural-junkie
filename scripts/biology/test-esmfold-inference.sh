#!/usr/bin/env bash
# Smoke-test ESMFold HF Inference URL (router.huggingface.co). Requires HF token in ~/.neural-junkie/config.json.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CONFIG="${NEURAL_JUNKIE_CONFIG:-$HOME/.neural-junkie/config.json}"
SEQ="${1:-MKTAYIAKQRQISFVK}"
MODEL="${2:-facebook/esmfold_v1}"

TOKEN="$(python3 -c "import json,sys; c=json.load(open(sys.argv[1])); print((c.get('hf') or {}).get('token',''))" "$CONFIG" 2>/dev/null || true)"
if [[ -z "$TOKEN" ]]; then
  echo "No hf.token in $CONFIG — set Settings → Hugging Face hub token"
  exit 1
fi

URL="https://router.huggingface.co/hf-inference/models/${MODEL}"
echo "POST $URL"
echo "sequence: $SEQ (${#SEQ} aa)"
RESP="$(curl -sS -w "\n%{http_code}" -X POST "$URL" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"inputs\":\"$SEQ\"}")"
BODY="$(echo "$RESP" | sed '$d')"
CODE="$(echo "$RESP" | tail -1)"
echo "HTTP $CODE"
echo "$BODY" | head -c 500
echo ""
if [[ "$CODE" == "200" ]] && echo "$BODY" | grep -q ATOM; then
  echo "OK: PDB-like response"
  exit 0
fi
if [[ "$CODE" == "400" ]] && echo "$BODY" | grep -qi 'not supported'; then
  echo "Expected: $MODEL is not on HF serverless (hf-inference). fold_protein will surface this in BiologyExpert."
  exit 0
fi
exit 1
