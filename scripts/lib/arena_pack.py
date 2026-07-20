"""Ensure Model Arena pack is installed + enabled for live arena benchmarks."""

from __future__ import annotations

from lib import collab_hub as hub


def ensure_model_arena_pack(hub_url: str) -> tuple[bool, str]:
    """Install and enable model-arena when missing. Returns (ok, detail)."""
    base = hub_url.rstrip("/")
    code, data = hub.hub_request(base, "GET", "/api/packs")
    if code != 200 or not isinstance(data, dict):
        return False, f"GET /api/packs failed ({code})"

    packs = data.get("packs") or []
    arena = next((p for p in packs if isinstance(p, dict) and p.get("id") == "model-arena"), None)
    if arena is None:
        code_i, out_i = hub.hub_request(base, "POST", "/api/packs/model-arena/install", None)
        if code_i not in (200, 201):
            return False, f"install model-arena failed ({code_i}): {out_i}"
        arena = {"id": "model-arena", "enabled": False}

    if not arena.get("enabled"):
        code_e, out_e = hub.hub_request(base, "PUT", "/api/packs/model-arena", {"enabled": True})
        if code_e != 200:
            return False, f"enable model-arena failed ({code_e}): {out_e}"

    code_c, _ = hub.hub_request(base, "GET", "/api/arena/challenges")
    if code_c != 200:
        return False, f"/api/arena/challenges still {code_c} after enable"
    return True, "model-arena pack installed and enabled"
