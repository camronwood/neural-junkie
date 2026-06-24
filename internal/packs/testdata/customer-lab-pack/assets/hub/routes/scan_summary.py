"""Scan summary API routes (pack-owned)."""
from __future__ import annotations


def handle_get(handler, path: str, settings: dict, pack_dir: str) -> None:
    if "well-image" in path:
        handler._json(501, {"error": "well-image requires workspace context from desktop"})
        return
    handler._json(404, {"error": "unknown scan-summary route"})
