#!/usr/bin/env bash
# Benchmark CAD chat models: generate SCAD via Ollama and compile with OpenSCAD.
# Usage: ./scripts/eval-cad-models.sh [model_tag]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROMPTS="${ROOT}/scenarios/cad/model-eval/prompts.json"
MODEL="${1:-qwen2.5-coder:14b}"
OLLAMA="${OLLAMA_HOST:-http://127.0.0.1:11434}"
TMP="${TMPDIR:-/tmp}/nj-cad-eval-$$"
mkdir -p "$TMP"

if ! command -v openscad >/dev/null 2>&1; then
  echo "openscad not found on PATH — install from https://openscad.org" >&2
  exit 1
fi

pass=0
fail=0
total=0

while IFS= read -r line; do
  id=$(echo "$line" | python3 -c "import sys,json; print(json.loads(sys.stdin.read())['id'])")
  prompt=$(echo "$line" | python3 -c "import sys,json; print(json.loads(sys.stdin.read())['prompt'])")
  total=$((total + 1))
  scad="${TMP}/${id}.scad"
  stl="${TMP}/${id}.stl"
  echo "=== ${MODEL} :: ${id} ==="
  body=$(python3 - <<PY
import json
print(json.dumps({"model": "${MODEL}", "prompt": """Write only valid OpenSCAD code for: ${prompt}""", "stream": False}))
PY
)
  curl -sf "${OLLAMA}/api/generate" -d "$body" | python3 -c "
import sys, json
d=json.load(sys.stdin)
text=d.get('response','')
# strip markdown fences if present
if '\`\`\`' in text:
  parts=text.split('\`\`\`')
  for p in parts:
    if 'module' in p or 'cube' in p or '=' in p:
      text=p
      break
open('${scad}','w').write(text.strip())
"
  if openscad -o "$stl" "$scad" 2>"${TMP}/${id}.log"; then
    echo "  compile: OK"
    pass=$((pass + 1))
  else
    echo "  compile: FAIL (see ${TMP}/${id}.log)"
    fail=$((fail + 1))
  fi
done < <(python3 -c "import json; [print(json.dumps(p)) for p in json.load(open('${PROMPTS}'))['prompts']]")

echo ""
echo "Model: ${MODEL}  Passed: ${pass}/${total}  Failed: ${fail}/${total}"
echo "Artifacts: ${TMP}"
