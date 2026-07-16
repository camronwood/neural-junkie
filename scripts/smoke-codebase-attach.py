#!/usr/bin/env python3
"""Smoke: @codebase attachments for minimal-repo ComputeObscureWidget."""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from lib import collab_hub as hub
from lib.workspace_context import enrich_send_metadata

FIXTURE = ROOT / "scenarios" / "fixtures" / "minimal-repo"
SCENARIO = {
    "workspace": {
        "fixture": "minimal-repo",
        "file_tree": "core/sample/main.go\ncore/obscure/internal/widget.go\n",
    }
}
BASE = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")
CHANNEL = "dm-smoke-codebase"


def main() -> int:
    hub.ensure_channel(BASE, CHANNEL)
    meta = enrich_send_metadata(
        {"conversation_mode": "code", "context_scope": "focus"},
        SCENARIO,
        content="@codebase What does ComputeObscureWidget return?",
    )
    code, data = hub.send_message(
        BASE,
        CHANNEL,
        "@codebase What does ComputeObscureWidget return?",
        metadata=meta,
        from_name="SmokeTest",
    )
    print(f"send HTTP {code}")
    if code != 200:
        return 1
    msgs = hub.list_messages(BASE, CHANNEL, 50)
    for m in reversed(msgs):
        if (m.get("type") or "") == "question":
            md = m.get("metadata") or {}
            atts = md.get("prompt_attachments") or []
            print(f"prompt_attachments count={len(atts)}")
            for a in atts[:5]:
                if isinstance(a, dict):
                    print(f"  - {a.get('path','?')}: {(a.get('content') or '')[:80]!r}")
            print("injected_codebase_count", md.get("injected_codebase_count"))
            body = json.dumps(md, indent=2)[:2000]
            print(body)
            if any(
                isinstance(a, dict)
                and "ComputeObscureWidget" in (a.get("content") or "")
                for a in atts
            ):
                print("OK: widget symbol in attachments")
                return 0
            print("FAIL: ComputeObscureWidget not in prompt_attachments")
            return 2
    print("FAIL: no question message found")
    return 3


if __name__ == "__main__":
    raise SystemExit(main())
