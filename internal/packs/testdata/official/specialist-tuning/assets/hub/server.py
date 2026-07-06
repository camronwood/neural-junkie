#!/usr/bin/env python3
"""Specialist-tuning pack hub sidecar."""
from __future__ import annotations

import argparse
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse


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
        if path.startswith("/api/lora/sidecar"):
            self._handle_train_get(path)
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
        if path.startswith("/api/lora/sidecar"):
            self._handle_train_post(path, body)
            return
        self._json(404, {"error": "not found"})

    def _handle_train_get(self, path: str) -> None:
        from routes import train

        train.handle_get(self, path, self.settings, self.pack_dir)

    def _handle_train_post(self, path: str, body: dict) -> None:
        from routes import train

        train.handle_post(self, path, body, self.settings, self.pack_dir)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=8765)
    args = parser.parse_args()
    hub_dir = os.path.dirname(os.path.abspath(__file__))
    os.chdir(hub_dir)
    server = ThreadingHTTPServer(("127.0.0.1", args.port), PackHubHandler)
    server.serve_forever()


if __name__ == "__main__":
    main()
