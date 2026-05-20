#!/usr/bin/env python3
"""Archive and inspect Neural Junkie hub session snapshots (~/.neural-junkie/last-session.json)."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

DEFAULT_SESSION = Path.home() / ".neural-junkie" / "last-session.json"
DEFAULT_ARCHIVE_DIR = Path.home() / ".neural-junkie" / "archives"
SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent


def load_session(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def stamp_from_saved_at(saved_at: str) -> str:
    s = (saved_at or "").strip()
    if not s:
        return datetime.now().astimezone().strftime("%Y-%m-%dT%H-%M-%S")
    return s.replace(":", "-").split(".")[0]


def cmd_archive(args: argparse.Namespace) -> int:
    src = Path(args.session).expanduser()
    if not src.is_file():
        print(f"error: session file not found: {src}", file=sys.stderr)
        return 1

    label = getattr(args, "label", None)
    note = getattr(args, "note", None)
    force = getattr(args, "force", False)
    quiet = getattr(args, "quiet", False)

    data = load_session(src)
    saved_at = data.get("saved_at") or ""
    stamp = stamp_from_saved_at(saved_at)
    if label:
        safe = "".join(c if c.isalnum() or c in "-_" else "-" for c in label.strip())
        stamp = f"{stamp}-{safe}"

    archive_dir = Path(getattr(args, "archive_dir", None) or DEFAULT_ARCHIVE_DIR).expanduser()
    archive_dir.mkdir(parents=True, exist_ok=True)
    dest = archive_dir / f"last-session-{stamp}.json"
    if dest.exists() and not force:
        dest = archive_dir / f"last-session-{stamp}-dup.json"

    shutil.copy2(src, dest)

    meta_path = dest.with_suffix(".meta.txt")
    channels = data.get("channels") or {}
    agents = data.get("agents") or []
    meta_lines = [
        f"archived_at={datetime.now().astimezone().isoformat()}",
        f"source={src}",
        f"saved_at={saved_at}",
        f"channels={len(channels)}",
        f"agents={len(agents)}",
        f"size_bytes={dest.stat().st_size}",
    ]
    if note:
        meta_lines.append(f"note={note}")
    meta_path.write_text("\n".join(meta_lines) + "\n", encoding="utf-8")

    print(dest)
    if quiet:
        return 0
    print(f"meta: {meta_path}")
    print(f"channels: {len(channels)}  agents: {len(agents)}  saved_at: {saved_at or '(missing)'}")
    return 0


def cmd_list(args: argparse.Namespace) -> int:
    archive_dir = Path(args.archive_dir).expanduser()
    if not archive_dir.is_dir():
        print(f"(no archive dir: {archive_dir})")
        return 0

    files = sorted(archive_dir.glob("last-session-*.json"), key=lambda p: p.stat().st_mtime, reverse=True)
    if not files:
        print("(no archives)")
        return 0

    limit = args.limit or len(files)
    for p in files[:limit]:
        st = p.stat()
        mtime = datetime.fromtimestamp(st.st_mtime).astimezone().isoformat(timespec="seconds")
        saved = ""
        meta = p.with_suffix(".meta.txt")
        if meta.is_file():
            for line in meta.read_text(encoding="utf-8").splitlines():
                if line.startswith("saved_at="):
                    saved = line.split("=", 1)[1]
                    break
        print(f"{p.name}\t{st.st_size}\t{mtime}\t{saved}")
    return 0


def channel_stats(
    channels: dict,
    channel_filter: str | None,
    *,
    tail: int = 0,
    max_chars: int = 100,
) -> None:
    names = sorted(channels.keys())
    if channel_filter:
        names = [n for n in names if channel_filter in n]

    for name in names:
        ch = channels.get(name) or {}
        msgs = ch.get("messages") or []
        types = Counter(m.get("type") for m in msgs)
        print(f"\n#{name} ({ch.get('type', '?')}) — {len(msgs)} messages")
        summary = (ch.get("session_summary") or "").strip()
        if summary:
            preview = summary.replace("\n", " ")[:120]
            print(f"  session_summary ({len(summary)} chars): {preview}…")
        if types:
            print("  types:", dict(types))
        for m in reversed(msgs):
            f = m.get("from") or {}
            if f.get("type") == "biology" or f.get("name") == "BiologyExpert":
                model = f.get("ai_model") or f.get("model") or "?"
                print(f"  BiologyExpert model (latest bio msg): {model}")
                break
        if tail and msgs:
            print(f"  --- last {tail} ---")
            for m in sorted(msgs, key=lambda x: x.get("timestamp", ""))[-tail:]:
                f = m.get("from") or {}
                who = f.get("name") or "?"
                ts = (m.get("timestamp") or "")[:19]
                body = (m.get("content") or "").replace("\n", " ")[:max_chars]
                print(f"    {ts} [{m.get('type')}] {who}: {body}")


def cmd_summary(args: argparse.Namespace) -> int:
    path = Path(args.session).expanduser()
    if not path.is_file():
        print(f"error: {path}", file=sys.stderr)
        return 1
    data = load_session(path)
    print(f"file: {path}")
    print(f"saved_at: {data.get('saved_at', '(missing)')}")
    print(f"channels: {len(data.get('channels') or {})}  agents: {len(data.get('agents') or [])}")

    for ag in data.get("agents") or []:
        if args.agent_filter and args.agent_filter not in (ag.get("type"), ag.get("name")):
            continue
        print(
            f"  agent {ag.get('name')}: type={ag.get('type')} "
            f"model={ag.get('ai_model') or ag.get('model')}"
        )

    channel_stats(
        data.get("channels") or {},
        args.channel,
        tail=args.tail,
        max_chars=args.max_chars,
    )
    return 0


def cmd_analyze(args: argparse.Namespace) -> int:
    debug = SCRIPT_DIR / "debug-collab.py"
    cmd = [sys.executable, str(debug), "session", "--session", str(Path(args.session).expanduser())]
    cmd.extend(args.extra)
    return subprocess.call(cmd)


def main() -> int:
    p = argparse.ArgumentParser(
        description="Archive and inspect Neural Junkie last-session.json snapshots.",
    )
    p.add_argument(
        "--session",
        default=str(DEFAULT_SESSION),
        help=f"Session JSON path (default: {DEFAULT_SESSION})",
    )
    sub = p.add_subparsers(dest="command")

    a = sub.add_parser("archive", help="Copy current session into ~/.neural-junkie/archives/")
    a.add_argument("--archive-dir", default=str(DEFAULT_ARCHIVE_DIR))
    a.add_argument("--label", help="Extra suffix on archive filename (e.g. before-bio-clear)")
    a.add_argument("--note", help="Stored in .meta.txt")
    a.add_argument("--force", action="store_true", help="Overwrite if same stamp exists")
    a.add_argument("-q", "--quiet", action="store_true", help="Print only archive path")
    a.set_defaults(func=cmd_archive)

    l = sub.add_parser("list", help="List archived sessions (newest first)")
    l.add_argument("--archive-dir", default=str(DEFAULT_ARCHIVE_DIR))
    l.add_argument("-n", "--limit", type=int, default=20)
    l.set_defaults(func=cmd_list)

    s = sub.add_parser("summary", help="Quick stats and optional channel tail")
    s.add_argument("--channel", help="Filter channels by substring (e.g. biologyexpert)")
    s.add_argument("--agent-filter", help="Filter agents by type or name substring")
    s.add_argument("--tail", type=int, default=0, help="Show last N messages per channel")
    s.add_argument("--max-chars", type=int, default=100)
    s.set_defaults(func=cmd_summary)

    an = sub.add_parser("analyze", help="Full analysis via debug-collab.py session")
    an.add_argument("extra", nargs=argparse.REMAINDER, help="Passed to debug-collab.py")
    an.set_defaults(func=cmd_analyze)

    args = p.parse_args()
    if not args.command:
        return cmd_archive(args)
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
