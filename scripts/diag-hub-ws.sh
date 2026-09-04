#!/usr/bin/env bash
# Quick hub WebSocket diagnostics (loopback hub on 18765 by default).
set -euo pipefail

HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
HUB="${HUB%/}"
WS_HOST="$(python3 - <<'PY'
import os, urllib.parse
u = urllib.parse.urlparse(os.environ.get("HUB", "http://127.0.0.1:18765"))
host = u.hostname or "127.0.0.1"
if host == "127.0.0.1":
    host = "localhost"
port = u.port or (443 if u.scheme == "https" else 80)
print(f"{host}:{port}")
PY
)"
HUB="$HUB" WS_HOST="$WS_HOST" python3 - <<'PY'
import json, os, subprocess, sys, urllib.request

hub = os.environ["HUB"]
ws_host = os.environ["WS_HOST"]

def curl_ws(origin: str | None) -> str:
    cmd = [
        "curl", "-s", "--max-time", "2", "-i",
        "-H", "Connection: Upgrade",
        "-H", "Upgrade: websocket",
        "-H", "Sec-WebSocket-Version: 13",
        "-H", "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
    ]
    if origin:
        cmd += ["-H", f"Origin: {origin}"]
    cmd.append(f"ws://{ws_host}/ws?channel=general")
    try:
        out = subprocess.check_output(cmd, text=True)
        return out.split("\r\n", 1)[0]
    except subprocess.CalledProcessError as e:
        return e.output.split("\n", 1)[0] if e.output else str(e)

print(f"Hub: {hub}")
try:
    with urllib.request.urlopen(f"{hub}/api/health", timeout=2) as resp:
        health = json.load(resp)
    print(f"Health: {health.get('status', health)}")
except Exception as e:
    print(f"Health: FAIL ({e})")
    sys.exit(1)

try:
    req = urllib.request.Request(
        f"{hub}/api/auth/session",
        data=json.dumps({"username": "diag"}).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=2) as resp:
        sess = json.load(resp)["token"]
    print(f"Session: ok ({sess[:12]}…)")
except Exception as e:
    print(f"Session: FAIL ({e})")
    sess = None

origins = [None, "http://localhost:1420", "https://tauri.localhost", "http://tauri.localhost"]
print("\nWebSocket upgrade probes:")
for origin in origins:
    label = origin or "<none>"
    status = curl_ws(origin)
    print(f"  Origin {label}: {status}")

if sess:
    cmd = [
        "curl", "-s", "--max-time", "2", "-i",
        "-H", "Connection: Upgrade",
        "-H", "Upgrade: websocket",
        "-H", "Sec-WebSocket-Version: 13",
        "-H", "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
        "-H", "Origin: https://tauri.localhost",
        f"ws://{ws_host}/ws?channel=general&nj_session={sess}",
    ]
    out = subprocess.check_output(cmd, text=True)
    print(f"  With session: {out.split(chr(13)+chr(10), 1)[0]}")
PY

echo
echo "Tip: packaged app prefers ws://localhost:18765 (not 127.0.0.1) for WebView compatibility."
