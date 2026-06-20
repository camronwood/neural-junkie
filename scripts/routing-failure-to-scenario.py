#!/usr/bin/env python3
"""Emit a reviewable chat/collab scenario stub from debug routing or compress output."""

from __future__ import annotations

import argparse
import json
import sys
import textwrap
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone


def fetch_json(url: str) -> dict:
    try:
        with urllib.request.urlopen(url, timeout=15) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.URLError as exc:
        print(f"fetch failed: {exc}", file=sys.stderr)
        sys.exit(1)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hub", default="http://127.0.0.1:18765")
    parser.add_argument("--mode", choices=["routing", "compress"], default="routing")
    parser.add_argument("--q", help="routing classify query")
    parser.add_argument("--tool", default="grep", help="compress tool name")
    parser.add_argument("--text", help="compress sample text")
    parser.add_argument("--agent-type", default="backend")
    parser.add_argument("--out", help="write JSON stub path (default stdout)")
    args = parser.parse_args()

    hub = args.hub.rstrip("/")
    if args.mode == "routing":
        if not args.q:
            parser.error("--q required for routing mode")
        qs = urllib.parse.urlencode({"q": args.q, "agent_type": args.agent_type})
        payload = fetch_json(f"{hub}/api/debug/routing-classify?{qs}")
        name = "routing-" + args.q[:40].replace(" ", "-").lower()
        steps = [
            {"action": "send", "from": "user", "content": args.q},
            {"action": "wait", "for": "agent_reply", "timeout_s": 120},
            {
                "action": "assert_metadata",
                "routing_domain": payload.get("domain", "general"),
            },
        ]
        description = f"Routing stub for: {args.q}"
    else:
        if not args.text:
            parser.error("--text required for compress mode")
        qs = urllib.parse.urlencode({"tool": args.tool, "text": args.text})
        payload = fetch_json(f"{hub}/api/debug/context-compress?{qs}")
        name = f"compress-{args.tool}-{datetime.now(timezone.utc).strftime('%Y%m%d%H%M')}"
        steps = [
            {
                "action": "note",
                "content": textwrap.dedent(
                    f"""
                    Compress preview: strategy={payload.get('strategy')}
                    bytes {payload.get('original_bytes')} -> {payload.get('compressed_bytes')}
                    ref={payload.get('ref')}
                    """
                ).strip(),
            }
        ]
        description = f"Compression fixture stub for tool {args.tool}"

    stub = {
        "name": name,
        "description": description,
        "tags": ["generated-stub", args.mode],
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "debug_payload": payload,
        "steps": steps,
    }

    body = json.dumps(stub, indent=2) + "\n"
    if args.out:
        with open(args.out, "w", encoding="utf-8") as fh:
            fh.write(body)
        print(f"wrote {args.out}")
    else:
        print(body)


if __name__ == "__main__":
    main()
