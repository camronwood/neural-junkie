"""Pack-owned MCP tool dispatch for the life-sciences hub sidecar."""
from __future__ import annotations

import json
import os
from typing import Any

from routes import biology

TOOL_TO_PATH = {
    "analyze_sequence": "/api/biology/analyze-sequence",
    "fold_protein": "/api/biology/fold",
    "structure_metadata": "/api/biology/structure-metadata",
    "blast_search": "/api/biology/blast",
    "pathway_lookup": "/api/biology/pathway",
    "validate_smiles": "/api/biology/validate-smiles",
    "mol_descriptors": "/api/biology/mol-descriptors",
}


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


def dispatch(name: str, args: dict, settings: dict, pack_dir: str) -> str:
    path = TOOL_TO_PATH.get(name)
    if path is None:
        raise ValueError(f"unknown tool: {name}")
    fn = biology.POST_ROUTES.get(path)
    if fn is None:
        raise ValueError(f"no route for tool: {name}")
    body = dict(args)
    if name == "pathway_lookup" and not body.get("gene") and body.get("query"):
        body["gene"] = body.get("query")
    result = fn(body, settings, pack_dir)
    if isinstance(result, (dict, list)):
        return json.dumps(result, indent=2, default=str)
    return str(result)
