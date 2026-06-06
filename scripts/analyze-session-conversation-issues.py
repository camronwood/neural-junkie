#!/usr/bin/env python3
"""Analyze ~/.neural-junkie/last-session.json for conversational quality issues."""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts" / "lib"))

from scenario_assert import looks_like_read_only_inspection_command  # noqa: E402

DEFAULT_SESSION = Path.home() / ".neural-junkie" / "last-session.json"

UNAWARE_PATTERNS = [
    r"i don't have access to your workspace",
    r"i cannot see your files",
    r"i don't have access to your files",
    r"no workspace context",
    r"share your workspace",
    r"i can't see your code",
]

PLACEHOLDER_PATTERNS = [
    r"\[insert ",
    r"insert file name",
    r"insert issues",
    r"lorem ipsum",
]

JSON_DISCUSSION_RE = re.compile(r"^\s*\{")


def load_session(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def analyze_session(data: dict) -> dict:
    issues: dict[str, list[dict]] = defaultdict(list)
    stats = Counter()

    for ch_name, ch in (data.get("channels") or {}).items():
        for msg in ch.get("messages") or []:
            content = msg.get("content") or ""
            meta = msg.get("metadata") or {}
            agent = (msg.get("from") or {}).get("name", "?")
            msg_type = msg.get("type", "?")

            if meta.get("generation_error") or meta.get("error_code"):
                issues["generation_errors"].append(
                    {
                        "channel": ch_name,
                        "agent": agent,
                        "type": msg_type,
                        "error_code": meta.get("error_code"),
                        "content": content[:160],
                    }
                )
                stats["generation_errors"] += 1

            if msg_type == "collaboration_discussion" and JSON_DISCUSSION_RE.match(content.strip()):
                issues["json_discussion"].append(
                    {"channel": ch_name, "agent": agent, "content": content[:160]}
                )
                stats["json_discussion"] += 1

            lower = content.lower()
            for pat in UNAWARE_PATTERNS:
                if re.search(pat, lower):
                    issues["workspace_unaware"].append(
                        {"channel": ch_name, "agent": agent, "content": content[:160]}
                    )
                    stats["workspace_unaware"] += 1
                    break

            for pat in PLACEHOLDER_PATTERNS:
                if re.search(pat, lower):
                    issues["placeholder_deliverables"].append(
                        {"channel": ch_name, "agent": agent, "content": content[:160]}
                    )
                    stats["placeholder_deliverables"] += 1
                    break

            if msg_type == "file_change":
                prop = meta.get("file_change_proposal") or {}
                new_content = (prop.get("new_content") or "") if isinstance(prop, dict) else ""
                phase = meta.get("collaboration_phase") or ""
                if phase == "cancelled":
                    issues["file_change_after_cancel"].append(
                        {
                            "channel": ch_name,
                            "agent": agent,
                            "path": prop.get("file_path") if isinstance(prop, dict) else "",
                        }
                    )
                    stats["file_change_after_cancel"] += 1
                for pat in PLACEHOLDER_PATTERNS:
                    if re.search(pat, (new_content or "").lower()):
                        issues["placeholder_proposals"].append(
                            {
                                "channel": ch_name,
                                "agent": agent,
                                "path": prop.get("file_path") if isinstance(prop, dict) else "",
                            }
                        )
                        stats["placeholder_proposals"] += 1
                        break

            if msg_type in ("chat", "answer", "collaboration_discussion"):
                for item in meta.get("suggested_commands") or []:
                    if not isinstance(item, dict):
                        continue
                    cmd = (item.get("command") or "").strip()
                    is_safe = bool(item.get("is_safe"))
                    if cmd and looks_like_read_only_inspection_command(cmd) and not is_safe:
                        issues["readonly_cmd_needs_approval"].append(
                            {
                                "channel": ch_name,
                                "agent": agent,
                                "command": cmd[:120],
                            }
                        )
                        stats["readonly_cmd_needs_approval"] += 1

            if "Cancelled" in content and msg_type == "system_info":
                stats["cancelled_collabs"] += 1

    return {"stats": dict(stats), "issues": dict(issues)}


def print_report(path: Path, report: dict) -> int:
    stats = report.get("stats") or {}
    issues = report.get("issues") or {}

    print(f"Session: {path}")
    print("Stats:")
    for key in sorted(stats):
        print(f"  {key}: {stats[key]}")

    exit_code = 0
    for kind, rows in sorted(issues.items()):
        if not rows:
            continue
        print(f"\n[{kind}] ({len(rows)})")
        for row in rows[:8]:
            print(f"  - {row}")
        if len(rows) > 8:
            print(f"  … {len(rows) - 8} more")
        exit_code = 1
    if exit_code == 0:
        print("\nNo conversational issue patterns detected.")
    return exit_code


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "session",
        nargs="?",
        default=str(DEFAULT_SESSION),
        help=f"Path to last-session.json (default: {DEFAULT_SESSION})",
    )
    parser.add_argument("--json", action="store_true", help="Emit JSON report")
    args = parser.parse_args()

    path = Path(args.session).expanduser()
    if not path.is_file():
        print(f"Session file not found: {path}", file=sys.stderr)
        return 2

    report = analyze_session(load_session(path))
    if args.json:
        print(json.dumps(report, indent=2))
        return 1 if report.get("issues") else 0
    return print_report(path, report)


if __name__ == "__main__":
    raise SystemExit(main())
