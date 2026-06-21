"""Build workspace_context and prompt_attachments for scenario harness sends."""

from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]


def language_for_path(path: str) -> str:
    ext = Path(path).suffix.lower()
    return {
        ".go": "go",
        ".ts": "typescript",
        ".tsx": "typescriptreact",
        ".js": "javascript",
        ".jsx": "javascriptreact",
        ".py": "python",
        ".md": "markdown",
        ".css": "css",
        ".html": "html",
    }.get(ext, "text")


def parse_file_folder_attachments(content: str) -> list[dict[str, str]]:
    """Mirror desktop/src/utils/ideComposer.ts parseFileFolderAttachments."""
    out: list[dict[str, str]] = []
    if not content:
        return out
    for m in re.finditer(r"@file:([^\s]+)", content, re.I):
        out.append({"type": "file_ref", "path": m.group(1)})
    for m in re.finditer(r"@folder:([^\s]+)", content, re.I):
        out.append({"type": "folder_outline", "path": m.group(1)})
    return out


def resolve_fixture_root(scenario: dict) -> Path:
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    root = os.environ.get("NEURAL_JUNKIE_SCENARIO_REPO", str(ROOT)).strip()
    fixture = (ws_cfg.get("fixture") or scenario.get("workspace_fixture") or "").strip()
    if fixture:
        root = str((ROOT / "scenarios" / "fixtures" / fixture).resolve())
    return Path(root)


def load_open_files(
    workspace_root: Path,
    rel_paths: list,
    *,
    active: str | None = None,
    selection: dict[str, Any] | None = None,
) -> list[dict]:
    active_rel = (active or (rel_paths[0] if rel_paths else "")).strip()
    sel = selection if isinstance(selection, dict) else {}
    sel_start = int(sel.get("start_line") or sel.get("selection_start_line") or 0)
    sel_end = int(sel.get("end_line") or sel.get("selection_end_line") or 0)
    sel_text = str(sel.get("selected_text") or "").strip()

    out: list[dict] = []
    for rel in rel_paths:
        rel = str(rel).strip()
        if not rel:
            continue
        full = (workspace_root / rel).resolve()
        content = ""
        if full.is_file():
            try:
                content = full.read_text(encoding="utf-8", errors="replace")[:10000]
            except OSError:
                content = ""
        is_active = rel == active_rel or (not active_rel and len(out) == 0)
        entry: dict[str, Any] = {
            "path": str(full),
            "language": language_for_path(rel),
            "content": content,
            "is_active": is_active,
        }
        if is_active and (sel_start or sel_end or sel_text):
            if sel_start:
                entry["selection_start_line"] = sel_start
            if sel_end:
                entry["selection_end_line"] = sel_end
            if sel_text:
                entry["selected_text"] = sel_text
            elif sel_start and sel_end and content:
                lines = content.splitlines()
                lo = max(0, sel_start - 1)
                hi = min(len(lines), sel_end)
                entry["selected_text"] = "\n".join(lines[lo:hi])
        out.append(entry)
    return out


def build_workspace_context(scenario: dict) -> dict[str, Any]:
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    root = resolve_fixture_root(scenario)
    open_rel = ws_cfg.get("open_files") or []
    active = (ws_cfg.get("active_file") or "").strip() or None
    if not active and open_rel:
        active = str(open_rel[0]).strip()
    selection = ws_cfg.get("selection")
    open_files = (
        load_open_files(root, open_rel, active=active, selection=selection)
        if open_rel
        else []
    )
    return {
        "workspace_name": root.name,
        "workspace_path": str(root),
        "file_tree": ws_cfg.get("file_tree") or "",
        "open_files": open_files,
    }


def enrich_send_metadata(
    meta: dict | None,
    scenario: dict,
    *,
    content: str = "",
    default_file_tree: str = "",
) -> dict | None:
    """Attach workspace_context and @file/@folder prompt_attachments like the desktop client."""
    if not meta:
        return None
    out = dict(meta)
    if not out.get("workspace_context"):
        ctx = build_workspace_context(scenario)
        if default_file_tree and not ctx.get("file_tree"):
            ctx["file_tree"] = default_file_tree
        out["workspace_context"] = ctx

    attachments = parse_file_folder_attachments(content)
    if attachments:
        existing = out.get("prompt_attachments")
        if isinstance(existing, list):
            out["prompt_attachments"] = existing + attachments
        else:
            out["prompt_attachments"] = attachments
    return out
