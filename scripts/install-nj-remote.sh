#!/usr/bin/env bash
# Install nj-remote on a remote Linux host and enable systemd service.
# Usage: ./scripts/install-nj-remote.sh [--remote user@host] [--root /path/to/repo] [--port 19876]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE=""
REPO_ROOT=""
PORT="19876"
BINARY="${ROOT}/bin/nj-remote"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --remote) REMOTE="$2"; shift 2 ;;
    --root) REPO_ROOT="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 1 ;;
  esac
done

if [[ ! -x "$BINARY" ]]; then
  echo "Building nj-remote..." >&2
  (cd "$ROOT" && make build-nj-remote)
fi

TOKEN="$(openssl rand -hex 16)"
ENV_FILE="/etc/nj-remote.env"

install_local() {
  sudo install -m 0755 "$BINARY" /usr/local/bin/nj-remote
  sudo tee "$ENV_FILE" >/dev/null <<EOF
NJ_REMOTE_TOKEN=${TOKEN}
NJ_REMOTE_ROOT=${REPO_ROOT:-$HOME}
NJ_REMOTE_ADDR=:${PORT}
EOF
  sudo tee /etc/systemd/system/nj-remote.service >/dev/null <<'UNIT'
[Unit]
Description=Neural Junkie remote sidecar
After=network.target

[Service]
EnvironmentFile=/etc/nj-remote.env
ExecStart=/usr/local/bin/nj-remote -root ${NJ_REMOTE_ROOT} -addr ${NJ_REMOTE_ADDR} -token ${NJ_REMOTE_TOKEN}
Restart=on-failure

[Install]
WantedBy=multi-user.target
UNIT
  sudo systemctl daemon-reload
  sudo systemctl enable --now nj-remote.service
  echo "Sidecar token: ${TOKEN}"
  echo "Health: curl -H \"Authorization: Bearer ${TOKEN}\" http://127.0.0.1:${PORT}/health"
}

if [[ -n "$REMOTE" ]]; then
  scp "$BINARY" "${REMOTE}:/tmp/nj-remote"
  ssh "$REMOTE" "sudo install -m 0755 /tmp/nj-remote /usr/local/bin/nj-remote && rm /tmp/nj-remote"
  ssh "$REMOTE" "NJ_REMOTE_TOKEN='${TOKEN}' NJ_REMOTE_ROOT='${REPO_ROOT:-$HOME}' NJ_REMOTE_PORT='${PORT}' bash -s" <<'REMOTE_SCRIPT'
set -euo pipefail
sudo tee /etc/nj-remote.env >/dev/null <<EOF
NJ_REMOTE_TOKEN=${NJ_REMOTE_TOKEN}
NJ_REMOTE_ROOT=${NJ_REMOTE_ROOT}
NJ_REMOTE_ADDR=:${NJ_REMOTE_PORT}
EOF
sudo tee /etc/systemd/system/nj-remote.service >/dev/null <<'UNIT'
[Unit]
Description=Neural Junkie remote sidecar
After=network.target

[Service]
EnvironmentFile=/etc/nj-remote.env
ExecStart=/usr/local/bin/nj-remote -root ${NJ_REMOTE_ROOT} -addr ${NJ_REMOTE_ADDR} -token ${NJ_REMOTE_TOKEN}
Restart=on-failure

[Install]
WantedBy=multi-user.target
UNIT
sudo systemctl daemon-reload
sudo systemctl enable --now nj-remote.service
REMOTE_SCRIPT
  echo "Remote sidecar installed on ${REMOTE}"
  echo "Token: ${TOKEN}"
else
  install_local
fi
