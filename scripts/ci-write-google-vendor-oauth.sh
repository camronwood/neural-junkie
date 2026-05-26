#!/usr/bin/env bash
# Write gitignored vendor/oauth.json from env (GitHub Actions secrets).
# Required for release builds: go build -tags googlevendor ./cmd/server
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/internal/google/meetnotes/vendor/oauth.json"

need=(GOOGLE_VENDOR_CLIENT_ID GOOGLE_VENDOR_CLIENT_SECRET)
missing=()
for v in "${need[@]}"; do
  if [[ -z "${!v:-}" ]]; then
    missing+=("$v")
  fi
done
if [[ ${#missing[@]} -gt 0 ]]; then
  echo "Missing GitHub Actions secrets (repo Settings -> Secrets -> Actions):" >&2
  printf '  - %s\n' "${missing[@]}" >&2
  echo "Values: Neural Junkie Google OAuth client_id and client_secret." >&2
  exit 1
fi

python3 - "$OUT" <<'PY'
import json, os, sys
from pathlib import Path

out = Path(sys.argv[1])
payload = {
    "client_id": os.environ["GOOGLE_VENDOR_CLIENT_ID"].strip(),
    "client_secret": os.environ["GOOGLE_VENDOR_CLIENT_SECRET"].strip(),
    "redirect_url": os.environ.get(
        "GOOGLE_VENDOR_REDIRECT_URL",
        "http://localhost:18765/api/assistant/google/callback",
    ).strip(),
}
for key, val in payload.items():
    if not val:
        sys.stderr.write(f"Empty Google vendor {key}\n")
        sys.exit(1)
    if "YOUR_" in val:
        sys.stderr.write(f"Placeholder value in {key}\n")
        sys.exit(1)
if not payload["client_id"].endswith(".apps.googleusercontent.com"):
    sys.stderr.write("client_id must end with .apps.googleusercontent.com\n")
    sys.exit(1)

out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(payload, indent=2) + "\n")
print(f"Wrote {out} for googlevendor build")
PY
