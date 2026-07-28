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
    r"\[feature name\]",
    r"\[brief description",
    r"\[step 1",
    r"\[explanation of",
    r"\[use case",
    r"insert file name",
    r"insert issues",
    r"lorem ipsum",
    r"--- title:",
    r"# app name",
    r"overview of the app",
    r"feature 1",
    r"achievement 1",
    r"key achievements",
]

PRIOR_REFERENCE_PATTERNS = [
    r"few messages back",
    r"what you wrote",
    r"you created",
    r"that art(?:icle|ical)",
    r"artical content",
    r"earlier you",
    r"from before",
]

FILE_EXPORT_PATTERNS = [
    r"store that",
    r"store it",
    r"save it",
    r"fill the file",
    r"create that file",
    r"please create that file",
    r"markdown file",
    r"\.md\b",
]

PREMATURE_APPLY_PATTERNS = [
    r"created and saved",
    r"has been saved",
    r"has been created",
    r"written to disk",
    r"file is ready",
]

STALE_SUMMARY_MARKERS = [
    r"still needed",
    r"open questions",
    r"sections to include",
    r"what sections",
]

JSON_DISCUSSION_RE = re.compile(r"^\s*\{")
BRACKET_PLACEHOLDER_RE = re.compile(r"\[[a-z][a-z0-9 _-]{2,}\]", re.I)
ABSOLUTE_FILE_CHANGE_RE = re.compile(r'\[FILE_CHANGE path="/Users/', re.I)

VAGUE_HANDOFF_RESULT_PATTERNS = [
    r"no specific sub-question was provided",
    r"please specify which of (?:the |those |these )?(?:two )?topics",
    r"please specify which concept",
    r"your prompt .+ is too broad",
    r"how can i assist you with your devops",
]

VAGUE_HANDOFF_TASK_PATTERNS = [
    r"whats?\s+up(?:\s+\w+){0,4}",
    r"\bor any other\b",
    r"\bor other\b",
    r"^(?:assist(?:_|\s+)?user|explain|help|assist)$",
    r".+\bor\b.+\bor\b.",
    r"debugging .+\bor\b reviewing",
    r"data pipelines,.+model serving",
    r"reviewing api designs,.+concurrency",
]

HANDOFF_STORM_THRESHOLD = 3
DEFAULT_HANDOFFS = Path.home() / ".neural-junkie" / "capability-handoffs.json"


def load_session(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def _msg_meta(msg: dict) -> dict:
    meta = msg.get("metadata") or {}
    return meta if isinstance(meta, dict) else {}


def _is_user(msg: dict) -> bool:
    sender = msg.get("from") or {}
    typ = (sender.get("type") or "").lower()
    name = (sender.get("name") or "").lower()
    if typ in ("human", "user", ""):
        return True
    # Desktop DM messages often use type=general for the human user.
    return typ == "general" and name not in ("system", "assistant")


def _assistant_msgs_before(messages: list[dict], idx: int, min_len: int = 400) -> bool:
    for prior in reversed(messages[:idx]):
        if _is_user(prior):
            continue
        content = (prior.get("content") or "").strip()
        if len(content) >= min_len:
            return True
    return False


def _has_later_apply(messages: list[dict], start_idx: int, path_hint: str = "") -> bool:
    for later in messages[start_idx + 1 :]:
        content = later.get("content") or ""
        meta = _msg_meta(later)
        if later.get("type") == "system_info" and "Applied change" in content:
            if not path_hint or path_hint in content:
                return True
        if meta.get("file_change_approved"):
            return True
    return False


def _user_requests_file_export(content: str) -> bool:
    lower = content.lower()
    for pat in FILE_EXPORT_PATTERNS:
        if re.search(pat, lower):
            return True
    has_file = ".md" in lower or "markdown file" in lower or "the file" in lower
    has_verb = bool(re.search(r"\b(store|save|create|fill|write)\b", lower))
    return has_file and has_verb


def _looks_like_vague_handoff_task(task: str) -> bool:
    lower = (task or "").strip().lower()
    if not lower:
        return True
    for pat in VAGUE_HANDOFF_TASK_PATTERNS:
        if re.search(pat, lower):
            return True
    if " or " in lower and "?" not in lower:
        return True
    return False


def _analyze_capability_handoffs(path: Path | None, issues: dict, stats: Counter) -> None:
    """Optional sidecar scan of ~/.neural-junkie/capability-handoffs.json."""
    handoffs_path = path
    if handoffs_path is None:
        handoffs_path = DEFAULT_HANDOFFS
    if not handoffs_path.is_file():
        return
    try:
        rows = json.loads(handoffs_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return
    if not isinstance(rows, list):
        return

    # Group by source channel within a short wall-clock window for storm detection.
    by_channel: dict[str, list[dict]] = defaultdict(list)
    for row in rows:
        if not isinstance(row, dict):
            continue
        task = (row.get("task") or "").strip()
        result = (row.get("result") or "").strip()
        channel = (row.get("source_channel") or row.get("channel") or "?").strip() or "?"
        by_channel[channel].append(row)
        if _looks_like_vague_handoff_task(task):
            issues["vague_capability_handoff"].append(
                {
                    "source": "capability-handoffs.json",
                    "channel": channel,
                    "task": task[:160],
                    "result": result[:160],
                }
            )
            stats["vague_capability_handoff"] += 1
            continue
        lower_result = result.lower()
        for pat in VAGUE_HANDOFF_RESULT_PATTERNS:
            if re.search(pat, lower_result):
                issues["vague_capability_handoff"].append(
                    {
                        "source": "capability-handoffs.json",
                        "channel": channel,
                        "task": task[:160],
                        "result": result[:160],
                    }
                )
                stats["vague_capability_handoff"] += 1
                break

    for channel, channel_rows in by_channel.items():
        if len(channel_rows) < HANDOFF_STORM_THRESHOLD:
            continue
        # Prefer timestamp clustering when available; otherwise count whole channel.
        stamped = []
        for row in channel_rows:
            ts = row.get("created_at") or row.get("CreatedAt") or ""
            stamped.append((ts, row))
        stamped.sort(key=lambda item: item[0] or "")
        # Any channel with >= threshold handoffs counts as a storm for the detector.
        issues["handoff_storm"].append(
            {
                "source": "capability-handoffs.json",
                "channel": channel,
                "count": len(channel_rows),
                "tasks": [((r.get("task") or "")[:80]) for r in channel_rows[:6]],
            }
        )
        stats["handoff_storm"] += 1


def analyze_session(data: dict, handoffs_path: Path | None = None) -> dict:
    issues: dict[str, list[dict]] = defaultdict(list)
    stats = Counter()
    handoff_events_by_channel: dict[str, int] = Counter()

    for ch_name, ch in (data.get("channels") or {}).items():
        messages = ch.get("messages") or []
        session_summary = (ch.get("session_summary") or "").lower()

        for idx, msg in enumerate(messages):
            content = msg.get("content") or ""
            meta = _msg_meta(msg)
            agent = (msg.get("from") or {}).get("name", "?")
            msg_type = msg.get("type", "?")

            handoff_event = meta.get("handoff_event")
            if handoff_event in ("handoff_started", "handoff_completed", "handoff_failed"):
                handoff_events_by_channel[ch_name] += 1
            if handoff_event == "handoff_completed" or (
                msg_type == "system_info" and "capability handoff" in content.lower()
            ):
                lower_content = content.lower()
                for pat in VAGUE_HANDOFF_RESULT_PATTERNS:
                    if re.search(pat, lower_content):
                        issues["vague_capability_handoff"].append(
                            {
                                "source": "session",
                                "channel": ch_name,
                                "agent": agent,
                                "content": content[:160],
                            }
                        )
                        stats["vague_capability_handoff"] += 1
                        break
                task = (meta.get("task") or "").strip()
                if task and _looks_like_vague_handoff_task(task):
                    issues["vague_capability_handoff"].append(
                        {
                            "source": "session",
                            "channel": ch_name,
                            "agent": agent,
                            "task": task[:160],
                        }
                    )
                    stats["vague_capability_handoff"] += 1

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
            bracket_hits = BRACKET_PLACEHOLDER_RE.findall(content)
            if len(bracket_hits) > 2 and len(content) < 4000 and msg_type in (
                "chat",
                "answer",
                "file_change",
            ):
                issues["placeholder_deliverables"].append(
                    {
                        "channel": ch_name,
                        "agent": agent,
                        "content": content[:160],
                        "placeholders": bracket_hits[:4],
                    }
                )
                stats["placeholder_deliverables"] += 1

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
                prop_brackets = BRACKET_PLACEHOLDER_RE.findall(new_content or "")
                if len(prop_brackets) > 2 and len(new_content or "") < 4000:
                    issues["placeholder_proposals"].append(
                        {
                            "channel": ch_name,
                            "agent": agent,
                            "path": prop.get("file_path") if isinstance(prop, dict) else "",
                        }
                    )
                    stats["placeholder_proposals"] += 1

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

            if _is_user(msg) and msg_type in ("chat", "question"):
                for pat in PRIOR_REFERENCE_PATTERNS:
                    if re.search(pat, lower):
                        if not _assistant_msgs_before(messages, idx):
                            issues["missing_prior_reference"].append(
                                {
                                    "channel": ch_name,
                                    "content": content[:160],
                                }
                            )
                            stats["missing_prior_reference"] += 1
                        break

                if _user_requests_file_export(content) and meta.get("conversation_mode") == "chat":
                    issues["file_export_chat_mode"].append(
                        {
                            "channel": ch_name,
                            "content": content[:160],
                            "conversation_mode": meta.get("conversation_mode"),
                        }
                    )
                    stats["file_export_chat_mode"] += 1

            if msg_type in ("chat", "answer") and not _is_user(msg):
                contradictory_save = bool(
                    re.search(r"successfully saved|has been saved|has been created", lower)
                    and re.search(
                        r"provide more details|what are the (name|purpose)|key features|sections to include",
                        lower,
                    )
                )
                for pat in PREMATURE_APPLY_PATTERNS:
                    if re.search(pat, lower):
                        if "proposal for your approval" in lower or "file change proposal" in lower:
                            break
                        if contradictory_save or not _has_later_apply(messages, idx):
                            issues["premature_file_apply_claim"].append(
                                {
                                    "channel": ch_name,
                                    "agent": agent,
                                    "content": content[:160],
                                }
                            )
                            stats["premature_file_apply_claim"] += 1
                        break
                else:
                    if contradictory_save:
                        issues["premature_file_apply_claim"].append(
                            {
                                "channel": ch_name,
                                "agent": agent,
                                "content": content[:160],
                            }
                        )
                        stats["premature_file_apply_claim"] += 1

            if ABSOLUTE_FILE_CHANGE_RE.search(content):
                issues["absolute_path_in_chat"].append(
                    {
                        "channel": ch_name,
                        "agent": agent,
                        "content": content[:160],
                    }
                )
                stats["absolute_path_in_chat"] += 1

        if session_summary:
            stale = any(re.search(pat, session_summary) for pat in STALE_SUMMARY_MARKERS)
            completed_export = any(
                _user_requests_file_export((m.get("content") or ""))
                or _msg_meta(m).get("file_change_approved")
                or (
                    m.get("type") == "system_info"
                    and "Applied change" in (m.get("content") or "")
                )
                for m in messages
            )
            if stale and completed_export:
                issues["stale_session_summary"].append(
                    {
                        "channel": ch_name,
                        "summary_excerpt": session_summary[:160],
                    }
                )
                stats["stale_session_summary"] += 1

    for ch_name, count in handoff_events_by_channel.items():
        if count >= HANDOFF_STORM_THRESHOLD:
            issues["handoff_storm"].append(
                {
                    "source": "session",
                    "channel": ch_name,
                    "count": count,
                }
            )
            stats["handoff_storm"] += 1

    _analyze_capability_handoffs(handoffs_path, issues, stats)

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


FIXTURE_SESSION = {
    "channels": {
        "dm-fixture": {
            "session_summary": "still needed: what sections to include in the article",
            "messages": [
                {
                    "type": "chat",
                    "from": {"name": "User", "type": "human"},
                    "content": "use the artical from a few messages back",
                    "metadata": {},
                },
                {
                    "type": "chat",
                    "from": {"name": "User", "type": "human"},
                    "content": "store that artical in a markdown file",
                    "metadata": {"conversation_mode": "chat"},
                },
                {
                    "type": "chat",
                    "from": {"name": "Assistant", "type": "assistant"},
                    "content": "The article has been successfully saved. Next, could you provide more details about the app name and key features?",
                    "metadata": {},
                },
                {
                    "type": "question",
                    "from": {"name": "camronwood", "type": "general"},
                    "content": "save that artical to nj-artical-1.md",
                    "metadata": {"conversation_mode": "code"},
                },
                {
                    "type": "system_info",
                    "from": {"name": "System", "type": "general"},
                    "content": "Applied change `fc1` to `new-artical-test.md`.",
                    "metadata": {},
                },
                {
                    "type": "file_change",
                    "from": {"name": "Assistant", "type": "assistant"},
                    "content": "proposal",
                    "metadata": {
                        "file_change_proposal": {
                            "file_path": "new-artical-test.md",
                            "new_content": "# App Name\n\nOverview of the app.\n\n## Features\n\n- Feature 1\n",
                        }
                    },
                },
                {
                    "type": "chat",
                    "from": {"name": "Assistant", "type": "assistant"},
                    "content": '[FILE_CHANGE path="/Users/test/proj/foo.md"]\nbody',
                    "metadata": {},
                },
                {
                    "type": "system_info",
                    "from": {"name": "System", "type": "general"},
                    "content": "Opening temporary capability handoff with @PlatformEngineer.",
                    "metadata": {"handoff_event": "handoff_started", "task": "whats up every body?"},
                },
                {
                    "type": "system_info",
                    "from": {"name": "System", "type": "general"},
                    "content": "Capability handoff result from @PlatformEngineer:\n\nNo specific sub-question was provided in your request.",
                    "metadata": {
                        "handoff_event": "handoff_completed",
                        "task": "debugging a failing pod or reviewing your CI/CD pipeline security",
                    },
                },
                {
                    "type": "system_info",
                    "from": {"name": "System", "type": "general"},
                    "content": "Opening temporary capability handoff with @DataMLEngineer.",
                    "metadata": {"handoff_event": "handoff_started"},
                },
            ],
        }
    }
}

FIXTURE_HANDOFFS = [
    {
        "source_channel": "general",
        "task": "whats up every body?",
        "result": "I am PlatformEngineer. How can I assist you with your DevOps tasks today?",
        "created_at": "2026-07-28T15:43:43",
    },
    {
        "source_channel": "general",
        "task": "debugging a failing pod or reviewing your CI/CD pipeline security",
        "result": "Please specify which of the two topics you would like to address first.",
        "created_at": "2026-07-28T15:44:29",
    },
    {
        "source_channel": "general",
        "task": "data pipelines, model serving, or any other ML tasks",
        "result": "No specific sub-question was provided in your request.",
        "created_at": "2026-07-28T15:45:18",
    },
]


def run_self_test() -> int:
    import tempfile

    with tempfile.TemporaryDirectory() as tmp:
        handoffs_path = Path(tmp) / "capability-handoffs.json"
        handoffs_path.write_text(json.dumps(FIXTURE_HANDOFFS), encoding="utf-8")
        report = analyze_session(FIXTURE_SESSION, handoffs_path=handoffs_path)
    issues = report.get("issues") or {}
    expected = {
        "missing_prior_reference",
        "file_export_chat_mode",
        "placeholder_proposals",
        "premature_file_apply_claim",
        "stale_session_summary",
        "absolute_path_in_chat",
        "vague_capability_handoff",
        "handoff_storm",
    }
    missing = sorted(expected - set(issues.keys()))
    if missing:
        print(f"self-test missing detectors: {missing}", file=sys.stderr)
        print(f"detected: {sorted(issues)}", file=sys.stderr)
        return 1
    print("self-test ok:", ", ".join(sorted(expected)))
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "session",
        nargs="?",
        default=str(DEFAULT_SESSION),
        help=f"Path to last-session.json (default: {DEFAULT_SESSION})",
    )
    parser.add_argument("--json", action="store_true", help="Emit JSON report")
    parser.add_argument(
        "--handoffs",
        default="",
        help=f"Optional capability-handoffs.json path (default: {DEFAULT_HANDOFFS} when present)",
    )
    parser.add_argument(
        "--self-test",
        action="store_true",
        help="Run built-in fixture detectors (missing_prior_reference, file_export_chat_mode, premature_file_apply_claim, placeholder_proposals, stale_session_summary, absolute_path_in_chat, vague_capability_handoff, handoff_storm)",
    )
    args = parser.parse_args()

    if args.self_test:
        return run_self_test()

    path = Path(args.session).expanduser()
    if not path.is_file():
        print(f"Session file not found: {path}", file=sys.stderr)
        return 2

    handoffs_path = Path(args.handoffs).expanduser() if args.handoffs else None
    report = analyze_session(load_session(path), handoffs_path=handoffs_path)
    if args.json:
        print(json.dumps(report, indent=2))
        return 1 if report.get("issues") else 0
    return print_report(path, report)


if __name__ == "__main__":
    raise SystemExit(main())
