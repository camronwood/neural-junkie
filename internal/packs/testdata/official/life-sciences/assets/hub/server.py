#!/usr/bin/env python3
"""Life sciences pack hub sidecar entrypoint."""
from __future__ import annotations

import argparse
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse

from routes import biology


def load_settings() -> dict:
    raw = os.environ.get("NJ_PACK_SETTINGS_JSON", "{}")
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return {}


class PackHubHandler(BaseHTTPRequestHandler):
    settings = load_settings()
    pack_id = os.environ.get("NJ_PACK_ID", "")
    pack_dir = os.environ.get("NJ_PACK_DIR", "")

    def log_message(self, fmt, *args):  # noqa: D401
        return

    def _json(self, code: int, payload: dict) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:
        path = urlparse(self.path).path
        if path == "/health":
            self._json(200, {"ok": True, "pack_id": self.pack_id})
            return
        if path.startswith("/api/biology/"):
            biology.handle_get(self, path, self.settings, self.pack_dir)
            return
        self._json(404, {"error": "not found"})

    def do_POST(self) -> None:
        path = urlparse(self.path).path
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length else b""
        try:
            body = json.loads(raw.decode("utf-8")) if raw else {}
        except json.JSONDecodeError:
            self._json(400, {"error": "invalid json"})
            return
        if path.startswith("/api/biology/"):
            biology.handle_post(self, path, body, self.settings, self.pack_dir)
            return
        self._json(404, {"error": "not found"})


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=0)
    args = parser.parse_args()
    port = args.port or int(os.environ.get("NJ_SIDECAR_PORT", "0"))
    server = ThreadingHTTPServer(("127.0.0.1", port), PackHubHandler)
    host, actual_port = server.server_address
    print(json.dumps({"ok": True, "port": actual_port}))
    server.serve_forever()


if __name__ == "__main__":
    main()
