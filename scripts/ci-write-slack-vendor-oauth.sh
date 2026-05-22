#!/usr/bin/env bash
# Write gitignored vendor/oauth.json from env (GitHub Actions secrets).
# Required for release builds: go build -tags slackvendor ./cmd/server
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/internal/integrations/slack/vendor/oauth.json"

need=(SLACK_VENDOR_CLIENT_ID SLACK_VENDOR_CLIENT_SECRET SLACK_VENDOR_APP_TOKEN)
missing=()
for v in "${need[@]}"; do
  if [[ -z "${!v:-}" ]]; then
    missing+=("$v")
  fi
done
if [[ ${#missing[@]} -gt 0 ]]; then
  echo "Missing GitHub Actions secrets (repo Settings → Secrets → Actions):" >&2
  printf '  - %s\n' "${missing[@]}" >&2
  echo "Values: Neural Junkie Slack app client_id, client_secret, app_token (xapp-)." >&2
  exit 1
fi

python3 - "$OUT" <<'PY'
import json, os, sys
from pathlib import Path

out = Path(sys.argv[1])
payload = {
    "client_id": os.environ["SLACK_VENDOR_CLIENT_ID"].strip(),
    "client_secret": os.environ["SLACK_VENDOR_CLIENT_SECRET"].strip(),
    "app_token": os.environ["SLACK_VENDOR_APP_TOKEN"].strip(),
}
for key, val in payload.items():
    if not val:
        sys.stderr.write(f"Empty SLACK_VENDOR_{key.upper()}\n")
        sys.exit(1)
    if "YOUR_" in val:
        sys.stderr.write(f"Placeholder value in {key}\n")
        sys.exit(1)
if not payload["app_token"].startswith("xapp-"):
    sys.stderr.write("app_token must start with xapp-\n")
    sys.exit(1)

out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(payload, indent=2) + "\n")
print(f"Wrote {out} for slackvendor build")
PY
