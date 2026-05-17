#!/usr/bin/env python3
"""
Neural Junkie — collab & conversation debug helper (dev).

Examples:
  ./scripts/debug-collab.py session
  ./scripts/debug-collab.py session --channel dm-camron-assistant --tail 8
  ./scripts/debug-collab.py live
  ./scripts/debug-collab.py live --collab ec2cdef8 --include-terminal
  ./scripts/debug-collab.py messages --channel collab-ec2cdef8-... --tail 15
  ./scripts/debug-collab.py watch
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from collections import Counter, defaultdict
from datetime import datetime
from pathlib import Path
from typing import Any

DEFAULT_SESSION = Path.home() / ".neural-junkie" / "last-session.json"
DEFAULT_HUB = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")
JOIN_RE = re.compile(r"has joined the chat", re.I)
COLLAB_CH_RE = re.compile(r"^collab-[a-f0-9-]{36}$", re.I)
COLLAB_ID_RE = re.compile(r"^[a-f0-9-]{36}$", re.I)


def load_json_path(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def hub_get(base: str, path: str, query: dict[str, str] | None = None) -> Any:
    url = f"{base}{path}"
    if query:
        url += "?" + urllib.parse.urlencode(query)
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=15) as resp:
        return json.loads(resp.read().decode())


def fmt_size(n: int) -> str:
    for unit in ("B", "KiB", "MiB", "GiB"):
        if n < 1024:
            return f"{n:.1f} {unit}" if unit != "B" else f"{n} B"
        n /= 1024
    return f"{n:.1f} TiB"


def channel_messages(ch: Any) -> list[dict[str, Any]]:
    if isinstance(ch, list):
        return [m for m in ch if isinstance(m, dict)]
    if isinstance(ch, dict):
        msgs = ch.get("messages")
        if isinstance(msgs, list):
            return [m for m in msgs if isinstance(m, dict)]
    return []


def msg_preview(m: dict[str, Any], n: int = 100) -> str:
    who = (m.get("from") or {}).get("name", "?")
    typ = m.get("type", "?")
    body = (m.get("content") or "").replace("\n", " ").strip()
    if len(body) > n:
        body = body[: n - 3] + "..."
    return f"[{typ}] {who}: {body}"


def collab_channels(snapshot: dict[str, Any]) -> dict[str, dict[str, Any]]:
    out: dict[str, dict[str, Any]] = {}
    collabs = snapshot.get("collaborations") or {}
    if isinstance(collabs, dict):
        for cid, c in collabs.items():
            if isinstance(c, dict):
                out[cid] = c
    return out


def print_header(title: str) -> None:
    print(f"\n{'=' * 72}")
    print(title)
    print("=" * 72)


def cmd_session(args: argparse.Namespace) -> int:
    path = Path(args.session).expanduser()
    if not path.is_file():
        print(f"No session file: {path}", file=sys.stderr)
        return 1

    raw = path.read_bytes()
    print_header("Session file")
    print(f"Path:     {path}")
    print(f"Size:     {fmt_size(len(raw))}")
    try:
        data = json.loads(raw.decode("utf-8"))
    except json.JSONDecodeError as e:
        print(f"JSON error: {e}")
        print("Hub may archive corrupt sessions on next boot.")
        return 1

    saved = data.get("saved_at", "?")
    print(f"saved_at: {saved}")

    channels = data.get("channels") or {}
    collabs = collab_channels(data)
    print(f"Channels: {len(channels)}  Collaborations: {len(collabs)}")

    # Join spam
    joins: Counter[str] = Counter()
    type_counts: Counter[str] = Counter()
    channel_counts: list[tuple[str, int]] = []

    for name, ch in channels.items():
        msgs = channel_messages(ch)
        channel_counts.append((name, len(msgs)))
        for m in msgs:
            type_counts[m.get("type", "?")] += 1
            if JOIN_RE.search(m.get("content") or ""):
                joins[name] += 1

    print_header("Channels (by message count)")
    for name, n in sorted(channel_counts, key=lambda x: -x[1])[:20]:
        flag = ""
        if joins.get(name, 0) > 2:
            flag = f"  ⚠ {joins[name]} join lines"
        if COLLAB_CH_RE.match(name):
            flag += "  [collab]"
        print(f"  {n:5d}  {name}{flag}")

    if joins:
        print_header("Join announcements (noise)")
        for name, n in joins.most_common():
            print(f"  {n:4d}  {name}")

    print_header("Message types (all channels)")
    for typ, n in type_counts.most_common(15):
        print(f"  {n:5d}  {typ}")

    print_header("Collaborations (persisted)")
    if not collabs:
        print("  (none)")
    for cid, c in collabs.items():
        ch_name = c.get("channel") or f"collab-{cid[:8]}"
        disc = c.get("discussion") or {}
        disc_msgs = disc.get("messages") if isinstance(disc.get("messages"), list) else []
        plan = c.get("plan") or {}
        plan_len = len((plan.get("content") or "")) if isinstance(plan, dict) else 0
        cfg = c.get("config") or {}
        print(f"\n  id:      {cid}")
        print(f"  channel: {ch_name}")
        print(f"  phase:   {c.get('phase')}  title: {(c.get('title') or '')[:70]}")
        print(
            f"  limits:  rounds={cfg.get('max_rounds')} turn_budget={cfg.get('turn_budget')} "
            f"max_msgs={cfg.get('max_total_messages')}"
        )
        print(
            f"  flags:   workspace_ack={c.get('workspace_acknowledged')} "
            f"tasks_dispatched={c.get('tasks_dispatched')} "
            f"exec_msgs={c.get('execution_message_count', 0)}"
        )
        print(
            f"  discuss: status={disc.get('status')} "
            f"round={disc.get('current_round')}/{disc.get('max_rounds')} "
            f"msgs={disc.get('total_message_count', len(disc_msgs))}/{disc.get('max_total_messages')}"
        )
        print(f"  plan:    {plan_len} chars  tasks: {len(c.get('tasks') or [])}")
        agents = c.get("agents") or []
        if agents:
            names = ", ".join(f"@{a.get('agent_name', '?')}" for a in agents[:8])
            print(f"  agents:  {names}")

    # Optional channel tail from session
    if args.channel:
        ch = channels.get(args.channel)
        if not ch:
            print(f"\nChannel not in session: {args.channel}", file=sys.stderr)
            return 1
        msgs = channel_messages(ch)
        print_header(f"Last {args.tail} messages in session — {args.channel}")
        for m in msgs[-args.tail :]:
            print(" ", msg_preview(m, args.width))

    # Hints
    print_header("Dev commands")
    print(f"  Live hub:     ./scripts/debug-collab.py live --hub {DEFAULT_HUB}")
    print(f"  Messages:     ./scripts/debug-collab.py messages --channel <name> --tail 20")
    print(f"  Hub memory:   NEURAL_JUNKIE_DEBUG=1 make server-debug  &&  ./scripts/dump-hub-memory.sh")
    print(f"  Watch logs:   ./scripts/debug-collab.py watch")
    return 0


def print_collab_live(c: dict[str, Any]) -> None:
    cid = c.get("id", "?")
    print(f"\n  id:      {cid}")
    print(f"  channel: {c.get('channel')}")
    print(f"  phase:   {c.get('phase')}  updated: {c.get('updated_at', '')}")
    print(f"  title:   {(c.get('title') or '')[:80]}")
    disc = c.get("discussion") or {}
    if disc:
        print(
            f"  discuss: status={disc.get('status')} "
            f"msgs={disc.get('total_message_count')}/{disc.get('max_total_messages')}"
        )
    print(
        f"  exec:    count={c.get('execution_message_count', 0)} "
        f"ack={c.get('workspace_acknowledged')} dispatched={c.get('tasks_dispatched')}"
    )


def cmd_live(args: argparse.Namespace) -> int:
    base = args.hub.rstrip("/")
    try:
        health = hub_get(base, "/api/health")
    except Exception as e:
        print(f"Hub not reachable at {base}: {e}", file=sys.stderr)
        print("Start with: make server   or   make server-debug", file=sys.stderr)
        return 1

    print_header("Live hub")
    print(f"URL:     {base}")
    print(f"status:  {health.get('status')}  agents: {health.get('agent_count')}  uptime: {health.get('uptime_secs')}s")
    feats = health.get("features") or []
    if feats:
        print(f"features: {', '.join(feats)}")

    q = {"include_terminal": "true" if args.include_terminal else "false"}
    if args.channel:
        q["channel"] = args.channel
    try:
        collabs = hub_get(base, "/api/collaborations", q)
    except Exception as e:
        print(f"Failed /api/collaborations: {e}", file=sys.stderr)
        return 1

    if not isinstance(collabs, list):
        collabs = []

    print_header(f"Collaborations ({len(collabs)})")
    needle = (args.collab or "").strip().lower()
    for c in collabs:
        if not isinstance(c, dict):
            continue
        cid = (c.get("id") or "").lower()
        if needle and needle not in cid and needle not in (c.get("channel") or "").lower():
            continue
        print_collab_live(c)

    if args.channel:
        try:
            msgs = hub_get(
                base,
                "/api/messages",
                {"channel": args.channel, "limit": str(args.tail)},
            )
            print_header(f"Last {args.tail} messages (live) — {args.channel}")
            if isinstance(msgs, list):
                for m in msgs[-args.tail :]:
                    if isinstance(m, dict):
                        print(" ", msg_preview(m, args.width))
        except Exception as e:
            print(f"messages: {e}", file=sys.stderr)

    print_header("Dev commands")
    print("  Session file:  ./scripts/debug-collab.py session")
    print("  Grant hub data for agents: use desktop modal or POST /api/hub-data/read")
    return 0


def cmd_messages(args: argparse.Namespace) -> int:
    if args.live:
        base = args.hub.rstrip("/")
        if not args.channel:
            print("--channel required for live messages", file=sys.stderr)
            return 1
        msgs = hub_get(base, "/api/messages", {"channel": args.channel, "limit": str(args.tail)})
        print_header(f"Live messages — {args.channel}")
        if isinstance(msgs, list):
            for m in msgs:
                if isinstance(m, dict):
                    print(msg_preview(m, args.width))
        return 0

    path = Path(args.session).expanduser()
    data = load_json_path(path)
    ch = (data.get("channels") or {}).get(args.channel)
    if not ch:
        print(f"Channel not in session: {args.channel}", file=sys.stderr)
        return 1
    msgs = channel_messages(ch)
    print_header(f"Session messages — {args.channel} (last {args.tail})")
    for m in msgs[-args.tail :]:
        print(msg_preview(m, args.width))
    return 0


def cmd_watch(args: argparse.Namespace) -> int:
    log = Path(args.log).expanduser()
    patterns = args.pattern or [
        "Collaboration",
        "collab",
        "budget",
        "discussion",
        "dispatch",
        "history_resync",
        "Session file",
    ]
    print(f"Tailing {log} (Ctrl+C to stop)")
    print(f"Filters: {', '.join(patterns)}")
    if not log.is_file():
        print(f"\nLog not found. Start hub with:\n  make server 2>&1 | tee {log}", file=sys.stderr)
        return 1
    import subprocess

    pat = "|".join(re.escape(p) for p in patterns)
    subprocess.run(["bash", "-c", f"tail -f '{log}' | grep --line-buffered -E '{pat}'"])
    return 0


def main() -> int:
    p = argparse.ArgumentParser(description="Debug Neural Junkie collabs and conversations")
    p.add_argument("--hub", default=DEFAULT_HUB, help="Hub base URL")
    p.add_argument("--session", default=str(DEFAULT_SESSION), help="Path to last-session.json")
    sub = p.add_subparsers(dest="cmd", required=True)

    s = sub.add_parser("session", help="Analyze persisted last-session.json")
    s.add_argument("--channel", help="Show last N messages for this channel")
    s.add_argument("--tail", type=int, default=10)
    s.add_argument("--width", type=int, default=120)

    live = sub.add_parser("live", help="Query running hub API")
    live.add_argument("--collab", help="Filter by collaboration id prefix")
    live.add_argument("--channel", help="Also fetch messages for channel")
    live.add_argument("--include-terminal", action="store_true")
    live.add_argument("--tail", type=int, default=15)
    live.add_argument("--width", type=int, default=120)

    msg = sub.add_parser("messages", help="Dump channel messages")
    msg.add_argument("--channel", required=True)
    msg.add_argument("--tail", type=int, default=20)
    msg.add_argument("--width", type=int, default=120)
    msg.add_argument("--live", action="store_true", help="From hub API instead of session file")

    w = sub.add_parser("watch", help="Tail hub log with collab-related lines")
    w.add_argument("--log", default="/tmp/nj-hub.log")
    w.add_argument("--pattern", nargs="*", help="Extra grep patterns")

    args = p.parse_args()
    args.hub = getattr(args, "hub", DEFAULT_HUB) or DEFAULT_HUB

    if args.cmd == "session":
        return cmd_session(args)
    if args.cmd == "live":
        return cmd_live(args)
    if args.cmd == "messages":
        return cmd_messages(args)
    if args.cmd == "watch":
        return cmd_watch(args)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
