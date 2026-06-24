"""Secondary analysis API routes (pack-owned)."""
from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def _tools_path(settings: dict, pack_dir: str) -> Path:
    rel = settings.get("secondary_analysis_tools_path", "assets/secondary-analysis-tools")
    return Path(pack_dir) / rel


def handle_get(handler, path: str, settings: dict, pack_dir: str) -> None:
    suffix = path.removeprefix("/api/secondary-analysis/").strip("/")
    if suffix.startswith("jobs/"):
        handler._json(200, {"status": "pending", "sidecar": True})
        return
    handler._json(404, {"error": "unknown secondary-analysis route"})


def handle_post(handler, path: str, body: dict, settings: dict, pack_dir: str) -> None:
    suffix = path.removeprefix("/api/secondary-analysis/").strip("/")
    tools = _tools_path(settings, pack_dir)
    python = settings.get("python_executable", sys.executable)
    if suffix == "run" and tools.is_dir():
        handler._json(200, {"job_id": "sidecar-stub", "status": "queued"})
        return
    if suffix == "12plex-qc":
        handler._json(200, {"sidecar": True, "result": "stub"})
        return
    handler._json(501, {"error": f"secondary-analysis {suffix} not implemented"})
