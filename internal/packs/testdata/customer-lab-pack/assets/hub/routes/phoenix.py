"""Phoenix TIM import routes (pack-owned)."""
from __future__ import annotations


def handle_get(handler, path: str, settings: dict, pack_dir: str) -> None:
    suffix = path.removeprefix("/api/phoenix/").strip("/")
    if suffix == "status":
        handler._json(
            200,
            {
                "authenticated": False,
                "environment": settings.get("phoenix_environment", "dev"),
                "pack_sidecar": True,
            },
        )
        return
    if suffix in ("analyses", "scan-results"):
        handler._json(200, {suffix.replace("-", "_"): []})
        return
    handler._json(404, {"error": "unknown phoenix route"})


def handle_post(handler, path: str, body: dict, settings: dict, pack_dir: str) -> None:
    suffix = path.removeprefix("/api/phoenix/").strip("/")
    if suffix in ("login/start", "logout"):
        handler._json(200, {"ok": True, "sidecar": True})
        return
    handler._json(501, {"error": f"phoenix {suffix} not implemented in sidecar stub"})
