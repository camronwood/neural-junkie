#!/usr/bin/env bash
# Push Slack vendor credentials to GitHub Actions secrets (from sandbox scripts/.slack-creds).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SANDBOX="$(cd "${ROOT}/../sandbox" 2>/dev/null && pwd)"
CREDS="${SLACK_CREDS:-${SANDBOX}/scripts/.slack-creds}"
REPO="${GITHUB_REPO:-camronwood/neural-junkie}"

if [[ ! -f "$CREDS" ]]; then
  echo "Missing ${CREDS}" >&2
  exit 1
fi
if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI required" >&2
  exit 1
fi

eval "$(python3 - "$CREDS" <<'PY'
import json, re, sys
from pathlib import Path

text = Path(sys.argv[1]).read_text()
fields = {}

def grab(label, pattern):
    m = re.search(pattern, text, re.I | re.M)
    if m:
        fields[label] = m.group(1).strip()

grab("client_id", r"Client\s*ID[:\s]+(\S+)")
grab("client_secret", r"Client\s*Secret[:\s]+(\S+)")
grab("app_token", r"NJ app token:\s*\n(xapp-\S+)",)
if "app_token" not in fields:
    grab("app_token", r"(?:slack\s*)?app\s*token[:\s]+(xapp-\S+)",)

missing = [k for k in ("client_id", "client_secret", "app_token") if k not in fields]
if missing:
    sys.stderr.write(f"Could not parse from {sys.argv[1]}: {missing}\n")
    sys.exit(1)

for k, v in fields.items():
    print(f'export SLACK_{k.upper()}={json.dumps(v)}')
PY
)"

echo "Setting GitHub Actions secrets on ${REPO} ..."
printf '%s' "$SLACK_CLIENT_ID" | gh secret set SLACK_VENDOR_CLIENT_ID --repo "$REPO"
printf '%s' "$SLACK_CLIENT_SECRET" | gh secret set SLACK_VENDOR_CLIENT_SECRET --repo "$REPO"
printf '%s' "$SLACK_APP_TOKEN" | gh secret set SLACK_VENDOR_APP_TOKEN --repo "$REPO"
if [[ -n "${SLACK_VENDOR_OAUTH_RELAY_BASE:-}" ]]; then
  printf '%s' "$SLACK_VENDOR_OAUTH_RELAY_BASE" | gh secret set SLACK_VENDOR_OAUTH_RELAY_BASE --repo "$REPO"
  echo "Done (SLACK_VENDOR_OAUTH_RELAY_BASE=${SLACK_VENDOR_OAUTH_RELAY_BASE}). Re-run Release workflow or tag a new version."
else
  echo "Done. Set relay after Cloudflare deploy:"
  echo "  make slack-oauth-relay-deploy-cf"
  echo "  SLACK_VENDOR_OAUTH_RELAY_BASE=https://nj-slack-oauth-relay.<subdomain>.workers.dev gh secret set SLACK_VENDOR_OAUTH_RELAY_BASE --repo $REPO"
fi
