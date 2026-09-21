"""Pack-owned MCP tool dispatch for the maps hub sidecar."""
from __future__ import annotations

import json
import os
from typing import Any

from routes import maps


def tools_catalog(pack_dir: str) -> list[dict[str, Any]]:
    path = os.path.join(pack_dir, "assets", "mcp", "tools.json")
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    return list(data.get("tools") or [])


def handle_tools_get(handler, pack_dir: str) -> None:
    try:
        tools = tools_catalog(pack_dir)
        handler._json(200, {"ok": True, "tools": tools})
    except Exception as exc:  # noqa: BLE001
        handler._json(500, {"ok": False, "error": str(exc)})


def handle_call(handler, body: dict, settings: dict, pack_dir: str) -> None:
    name = str(body.get("name") or "").strip()
    args = body.get("arguments") if isinstance(body.get("arguments"), dict) else {}
    if not name:
        handler._json(400, {"ok": False, "error": "missing tool name"})
        return
    try:
        text = dispatch(name, args, settings, pack_dir)
        handler._json(200, {"ok": True, "text": text})
    except ValueError as exc:
        handler._json(400, {"ok": False, "error": str(exc)})
    except RuntimeError as exc:
        handler._json(503, {"ok": False, "error": str(exc)})
    except Exception as exc:  # noqa: BLE001
        handler._json(500, {"ok": False, "error": str(exc)})


class _Capture:
    """Minimal handler stand-in so maps routes can return JSON without HTTP."""

    def __init__(self) -> None:
        self.code = 500
        self.payload: dict[str, Any] = {}

    def _json(self, code: int, payload: dict) -> None:
        self.code = code
        self.payload = payload


def dispatch(name: str, args: dict, settings: dict, pack_dir: str) -> str:
    cap = _Capture()
    if name == "maps_geocode":
        maps.handle_post(cap, "/api/maps/geocode", dict(args), settings, pack_dir)
    elif name == "maps_route":
        maps.handle_post(cap, "/api/maps/route", dict(args), settings, pack_dir)
    else:
        raise ValueError(f"unknown tool: {name}")
    if cap.code >= 400:
        err = cap.payload.get("error") if isinstance(cap.payload, dict) else None
        raise RuntimeError(str(err or f"maps tool failed ({cap.code})"))
    return json.dumps(cap.payload, indent=2, default=str)
