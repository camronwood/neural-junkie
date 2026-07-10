#!/usr/bin/env python3
"""Batch-encode marketing MP4s for GitHub Pages."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ENCODER = ROOT / "scripts" / "optimize-site-videos.sh"
DEFAULT_SRC = Path.home() / "Desktop" / "NJ videos"
DEFAULT_DEST = ROOT / "docs" / "media"

MAPPING = {
    "FeatureFlyThrough.mp4": "feature-flythrough.mp4",
    "GeminiCopilotCursorEdgeAssistantChat.mp4": "gemini-copilot-cursor-edge-assistant-chat.mp4",
    "AskTheAssistant.mp4": "ask-the-assistant.mp4",
    "NJSlack30Seconds.mp4": "nj-slack.mp4",
    "LocalImageGenForFree.mp4": "local-image-gen-free.mp4",
    "YourAgentsRespondWhenYouAreAway.mp4": "agents-respond-when-away.mp4",
    "WebSiteCollab.mp4": "website-collab.mp4",
    "SlackMessageForwarding.mp4": "slack-message-forwarding.mp4",
    "AskTheAssistantMobile.mp4": "ask-the-assistant-mobile.mp4",
}


def main() -> int:
    src_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else DEFAULT_SRC
    dest_dir = Path(sys.argv[2]) if len(sys.argv) > 2 else DEFAULT_DEST
    if not ENCODER.is_file():
        print(f"Missing encoder script: {ENCODER}", file=sys.stderr)
        return 1
    if not src_dir.is_dir():
        print(f"Source directory not found: {src_dir}", file=sys.stderr)
        return 1
    dest_dir.mkdir(parents=True, exist_ok=True)

    for src_name, dest_name in MAPPING.items():
        src = src_dir / src_name
        dest = dest_dir / dest_name
        if not src.is_file():
            print(f"SKIP missing source: {src}", file=sys.stderr)
            continue
        print(f">>> {src_name} -> {dest_name}")
        subprocess.run([str(ENCODER), str(src), str(dest)], check=True)

    print("=== DONE ===")
    for path in sorted(dest_dir.glob("*.mp4")):
        size_mb = path.stat().st_size / (1024 * 1024)
        print(f"  {path.name:50s} {size_mb:6.1f} MB")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
