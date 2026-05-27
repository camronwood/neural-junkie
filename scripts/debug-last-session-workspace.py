#!/usr/bin/env python3
"""Flag bad workspace_context paths in ~/.neural-junkie/last-session.json.

Detects when /collaborate (or similar) was sent with a collaboration sandbox
as the active workspace instead of a real project checkout.

Usage:
  ./scripts/debug-last-session-workspace.py
  ./scripts/debug-last-session-workspace.py /path/to/last-session.json
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path


def default_session_path() -> Path:
    return Path.home() / ".neural-junkie" / "last-session.json"


def is_collab_assets_path(path: str) -> bool:
    normalized = path.replace("\\", "/").strip()
    return "/.neural-junkie/collaborations/" in normalized or normalized.endswith(
        "/.neural-junkie/collaborations"
    )


def iter_channel_messages(data: dict):
    for channel_id, ch in (data.get("channels") or {}).items():
        for msg in ch.get("messages") or []:
            yield channel_id, msg


def main() -> int:
    path = Path(sys.argv[1]) if len(sys.argv) > 1 else default_session_path()
    if not path.is_file():
        print(f"Session file not found: {path}", file=sys.stderr)
        return 1

    raw = path.read_text(encoding="utf-8")
    data = json.loads(raw)
    saved_at = data.get("saved_at", "?")
    print(f"Session: {path}")
    print(f"saved_at: {saved_at}")
    print()

    issues: list[str] = []
    for channel_id, msg in iter_channel_messages(data):
        content = (msg.get("content") or msg.get("text") or "").strip()
        if not content.lower().startswith("/collaborate"):
            continue
        meta = msg.get("metadata") or {}
        ws = meta.get("workspace_context") or {}
        ws_path = (ws.get("workspace_path") or "").strip()
        if not ws_path:
            continue
        if is_collab_assets_path(ws_path):
            issues.append(
                f"  [{channel_id}] /collaborate with collab sandbox workspace:\n"
                f"    path: {ws_path}\n"
                f"    name: {ws.get('workspace_name', '')}\n"
                f"    scope: {meta.get('context_scope', '?')}"
            )

    for cid, collab in (data.get("collaborations") or {}).items():
        src = (collab.get("source_repo_path") or "").strip()
        if src and is_collab_assets_path(src):
            issues.append(
                f"  collaboration {cid[:8]}... source_repo_path is a collab folder:\n"
                f"    {src}"
            )

    if not issues:
        print("OK — no /collaborate messages with collab-sandbox workspace paths found.")
        return 0

    print("ISSUES FOUND:")
    print("\n".join(issues))
    print()
    print("Fix: select a real project workspace in the desktop app, or use")
    print("  /collaborate --workspace ...")
    print("after hub/desktop guards are deployed.")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
