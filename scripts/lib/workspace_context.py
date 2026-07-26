"""Build workspace_context and prompt_attachments for scenario harness sends."""

from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]


def default_scenario_repo(root: Path = ROOT) -> str:
    """Default workspace root for scenario harnesses when env is unset."""
    return str((root / "scenarios" / "fixtures" / "minimal-repo").resolve())


def build_outline_file_tree(root: Path, *, max_depth: int = 3) -> str:
    """Shallow directory outline for workspace_context (aligned with hub file_tree_outline)."""
    root = root.resolve()
    if not root.is_dir():
        return ""
    lines: list[str] = []

    def walk(dir_path: Path, depth: int) -> None:
        if depth >= max_depth:
            return
        try:
            entries = sorted(dir_path.iterdir(), key=lambda p: (not p.is_dir(), p.name.lower()))
        except OSError:
            return
        indent = "  " * depth
        for entry in entries:
            name = entry.name
            if name.startswith(".") and name != ".":
                continue
            if entry.is_dir():
                lines.append(f"{indent}{name}/")
                if depth + 1 < max_depth:
                    walk(entry, depth + 1)
            elif depth + 1 >= max_depth - 1:
                lines.append(f"{indent}{name}")

    walk(root, 0)
    if not lines:
        return ".\n"
    return "\n".join(lines) + "\n"


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
    root = os.environ.get("NEURAL_JUNKIE_SCENARIO_REPO", "").strip() or default_scenario_repo()
    fixture = (ws_cfg.get("fixture") or scenario.get("workspace_fixture") or "").strip()
    if fixture:
        root = str((ROOT / "scenarios" / "fixtures" / fixture).resolve())
    return Path(root)


def build_workspace_context_for_path(repo_path: str | Path, scenario: dict) -> dict[str, Any]:
    """Build workspace_context for a resolved repo path (collab harness outbound metadata)."""
    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    root = Path(repo_path).resolve()
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
    file_tree = (ws_cfg.get("file_tree") or "").strip()
    if not file_tree:
        file_tree = build_outline_file_tree(root)
    return {
        "workspace_name": root.name,
        "workspace_path": str(root),
        "file_tree": file_tree,
        "open_files": open_files,
        "unchanged_files": list(ws_cfg.get("unchanged_files") or []),
    }


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
        "unchanged_files": list(ws_cfg.get("unchanged_files") or []),
    }


_AGENT_TYPE_BY_NAME = {
    "backendengineer": "backend",
    "frontendengineer": "frontend",
    "softwarearchitect": "architecture",
    "securityreviewer": "security",
    "assistant": "assistant",
}


def ide_route_for_target_agent(target_agent: str) -> str:
    """Map scenario target_agent display name to ide_route_agent_type."""
    key = (target_agent or "").strip().lstrip("@").lower()
    return _AGENT_TYPE_BY_NAME.get(key, "")


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
    # Public implement/parity channels keep multiple specialists subscribed. Without an
    # explicit route, a prior scenario's agent (e.g. FrontendEngineer) can answer first
    # and starve wait_reply for the intended target.
    if not str(out.get("ide_route_agent_type") or "").strip():
        target = (scenario.get("target_agent") or "").strip().lstrip("@")
        route = ide_route_for_target_agent(target)
        if route:
            out["ide_route_agent_type"] = route

    ws_cfg = scenario.get("workspace") if isinstance(scenario.get("workspace"), dict) else {}
    if not out.get("workspace_context"):
        ctx = build_workspace_context(scenario)
        if default_file_tree and not ctx.get("file_tree"):
            ctx["file_tree"] = default_file_tree
        out["workspace_context"] = ctx

    linked_cfg = ws_cfg.get("linked_workspaces")
    if isinstance(linked_cfg, list) and linked_cfg:
        linked_out: list[dict[str, Any]] = []
        for item in linked_cfg:
            if not isinstance(item, dict):
                continue
            fixture = (item.get("fixture") or "").strip()
            if not fixture:
                continue
            link_root = (ROOT / "scenarios" / "fixtures" / fixture).resolve()
            linked_out.append(
                {
                    "workspace_name": item.get("name") or link_root.name,
                    "workspace_path": str(link_root),
                    "source": item.get("source") or "open_tab",
                    "file_tree": item.get("file_tree")
                    or build_outline_file_tree(link_root, max_depth=2),
                    "open_files": load_open_files(
                        link_root,
                        item.get("open_files") or [],
                        active=(item.get("active_file") or None),
                    ),
                }
            )
        if linked_out:
            out["linked_workspaces"] = linked_out

    attachments = parse_file_folder_attachments(content)
    if attachments:
        existing = out.get("prompt_attachments")
        if isinstance(existing, list):
            out["prompt_attachments"] = existing + attachments
        else:
            out["prompt_attachments"] = attachments
    return out
