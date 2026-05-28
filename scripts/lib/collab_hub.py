"""HTTP helpers for live-hub collaboration scenario tests."""

from __future__ import annotations

import json
import os
import re
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

DEFAULT_HUB = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")
POLL_INTERVAL = 1.0
MAX_CONCURRENT_COLLABS = 3

AGENT_PROFILES: dict[str, str] = {
    "fast": "@ChatModerator @Assistant",
    "realistic": "@SoftwareArchitect @BackendEngineer",
}

DISCUSSION_TYPES = frozenset(
    {
        "collaboration_discussion",
        "answer",
        "chat",
    }
)


def hub_request(
    base: str,
    method: str,
    path: str,
    body: dict | None = None,
) -> tuple[int, Any]:
    url = f"{base.rstrip('/')}{path}"
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            raw = resp.read().decode()
            if resp.status == 204 or not raw.strip():
                return resp.status, None
            return resp.status, json.loads(raw)
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            parsed = json.loads(raw) if raw.strip() else raw
        except json.JSONDecodeError:
            parsed = raw
        return e.code, parsed


def check_health(base: str) -> dict | None:
    code, data = hub_request(base, "GET", "/api/health")
    if code != 200 or not isinstance(data, dict):
        return None
    return data


def ensure_channel(base: str, name: str, description: str = "Automated collab scenario tests") -> bool:
    code, chs = hub_request(base, "GET", "/api/channels")
    if code == 200 and isinstance(chs, list):
        if any(isinstance(c, dict) and c.get("name") == name for c in chs):
            return True
    code, _out = hub_request(
        base,
        "POST",
        "/api/channels/create",
        {
            "name": name,
            "description": description,
            "type": "public",
            "created_by": "CollabScenario",
        },
    )
    return code in (200, 201)


def send_message(
    base: str,
    channel: str,
    content: str,
    *,
    metadata: dict | None = None,
    from_name: str = "CollabScenario",
) -> tuple[int, dict | None]:
    payload: dict[str, Any] = {
        "channel": channel,
        "content": content,
        "type": "question",
        "from": {"name": from_name, "type": "human"},
    }
    if metadata:
        payload["metadata"] = metadata
    code, data = hub_request(base, "POST", "/api/send", payload)
    if code == 200 and isinstance(data, dict):
        return code, data
    return code, data if isinstance(data, dict) else None


def workspace_ack(
    base: str,
    collab_id: str,
    source_repo_path: str = "",
) -> int:
    body: dict[str, str] = {"collaboration_id": collab_id}
    if source_repo_path.strip():
        body["source_repo_path"] = source_repo_path.strip()
    code, _ = hub_request(base, "POST", "/api/collaboration-workspace-ack", body)
    return code


def list_messages(base: str, channel: str, limit: int = 200) -> list[dict]:
    q = urllib.parse.urlencode({"channel": channel, "limit": str(limit)})
    code, data = hub_request(base, "GET", f"/api/messages?{q}")
    if code != 200 or not isinstance(data, list):
        return []
    return [m for m in data if isinstance(m, dict)]


def list_active_collaborations(base: str) -> list[dict]:
    code, data = hub_request(base, "GET", "/api/collaborations?include_terminal=false")
    if code != 200 or not isinstance(data, list):
        return []
    return [c for c in data if isinstance(c, dict)]


def find_collab(collabs: list, collab_id: str) -> dict | None:
    for c in collabs:
        if isinstance(c, dict) and c.get("id") == collab_id:
            return c
    return None


def fetch_collab(base: str, channel: str, collab_id: str) -> dict | None:
    q = urllib.parse.urlencode({"channel": channel, "include_terminal": "true"})
    code, data = hub_request(base, "GET", f"/api/collaborations?{q}")
    if code != 200 or not isinstance(data, list):
        return None
    return find_collab(data, collab_id)


def collab_phase(base: str, channel: str, collab_id: str) -> str | None:
    c = fetch_collab(base, channel, collab_id)
    if not c:
        return None
    return c.get("phase")


def wait_phase(
    base: str,
    channel: str,
    collab_id: str,
    want: str,
    timeout: float,
) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        if collab_phase(base, channel, collab_id) == want:
            return True
        time.sleep(POLL_INTERVAL)
    return False


def wait_planning_recap(base: str, channel: str, collab_id: str, timeout: float) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        collab = fetch_collab(base, channel, collab_id)
        if collab is None:
            return False
        status = (collab.get("planning_recap_status") or "").strip().lower()
        if status in ("complete", "failed"):
            return True
        time.sleep(POLL_INTERVAL)
    return False


def wait_tasks_status(
    base: str,
    channel: str,
    collab_id: str,
    *,
    want_status: str = "completed",
    min_completed: int = 0,
    all_match: bool = True,
    timeout: float = 120,
) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        collab = fetch_collab(base, channel, collab_id)
        if collab is None:
            time.sleep(POLL_INTERVAL)
            continue
        tasks = collab.get("tasks") or []
        if not tasks:
            time.sleep(POLL_INTERVAL)
            continue
        statuses = [t.get("status") for t in tasks if isinstance(t, dict)]
        completed = sum(1 for s in statuses if s == "completed")
        if all_match and statuses and all(s == want_status for s in statuses):
            return True
        if min_completed > 0 and completed >= min_completed:
            return True
        time.sleep(POLL_INTERVAL)
    return False


def cancel_collab(base: str, collab: dict) -> bool:
    cid = collab.get("id") or ""
    ch = collab.get("channel") or ""
    if not cid or not ch:
        return False
    code, _ = send_message(base, ch, f"/cancel-plan {cid[:8]}")
    return code == 200


def free_scenario_capacity(
    base: str,
    channel: str,
    markers: tuple[str, ...] = ("collab-scenario", "nj collab scenario", "collab scenario"),
) -> bool:
    active = list_active_collaborations(base)
    cancelled = 0
    for c in active:
        title = (c.get("title") or "").lower()
        desc = (c.get("description") or "").lower()
        ch = (c.get("channel") or "").lower()
        if ch == channel.lower() or any(m in title or m in desc for m in markers):
            if cancel_collab(base, c):
                cancelled += 1
    if cancelled:
        time.sleep(0.5)

    active = list_active_collaborations(base)
    if len(active) < MAX_CONCURRENT_COLLABS:
        return True
    return False


def last_system_error(base: str, channel: str) -> str:
    for m in reversed(list_messages(base, channel, 10)):
        if m.get("type") != "system_info":
            continue
        body = (m.get("content") or "").strip()
        if body.startswith("❌") or "Ignored active workspace" in body:
            return body
    return ""


def agent_messages(
    messages: list[dict],
    *,
    types: frozenset[str] | None = None,
    exclude_system: bool = True,
) -> list[dict]:
    out: list[dict] = []
    for m in messages:
        if exclude_system and (m.get("from") or {}).get("name") == "System":
            if m.get("type") == "collaboration_discussion":
                continue
        typ = m.get("type") or ""
        if types and typ not in types:
            continue
        if typ in DISCUSSION_TYPES or typ == "collaboration_discussion":
            who = (m.get("from") or {}).get("name") or ""
            if who and who != "System":
                out.append(m)
    return out


def count_by_agent(messages: list[dict]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for m in messages:
        who = (m.get("from") or {}).get("name") or "?"
        counts[who] = counts.get(who, 0) + 1
    return counts


def messages_matching(messages: list[dict], pattern: str) -> list[dict]:
    rx = re.compile(pattern, re.IGNORECASE | re.MULTILINE)
    return [m for m in messages if rx.search(m.get("content") or "")]


def resolve_agents(profile: str | None, override: str | None) -> str:
    if override and override.strip():
        return override.strip()
    key = (profile or os.environ.get("NJ_SCENARIO_PROFILE") or "fast").strip().lower()
    return AGENT_PROFILES.get(key, AGENT_PROFILES["fast"])


def parse_timeout(value: float | int | str, default: float = 90) -> float:
    if isinstance(value, (int, float)):
        return float(value)
    s = str(value).strip().lower()
    if s.endswith("s"):
        try:
            return float(s[:-1])
        except ValueError:
            return default
    try:
        return float(s)
    except ValueError:
        return default


def list_pending_file_changes(base: str, user_id: str = "default") -> list[dict]:
    q = urllib.parse.urlencode({"user_id": user_id})
    code, data = hub_request(base, "GET", f"/api/file-changes?{q}")
    if code != 200 or not isinstance(data, list):
        return []
    return [c for c in data if isinstance(c, dict)]


def approve_file_change(base: str, change_id: str, user_id: str = "default") -> tuple[int, Any]:
    q = urllib.parse.urlencode({"user_id": user_id})
    return hub_request(base, "POST", f"/api/file-changes/approve/{change_id}?{q}")


def pending_change_ids_for_channel(
    base: str,
    channel: str,
    *,
    path_contains: str = "",
    user_id: str = "default",
) -> list[str]:
    """Collect pending change IDs from the API and from message metadata."""
    ids: list[str] = []
    seen: set[str] = set()

    def add(cid: str) -> None:
        cid = (cid or "").strip()
        if not cid or cid in seen:
            return
        seen.add(cid)
        ids.append(cid)

    needle = path_contains.lower()
    for change in list_pending_file_changes(base, user_id):
        ch = (change.get("channel") or "").strip()
        if ch != channel:
            continue
        fp = (change.get("file_path") or "").lower()
        if needle and needle not in fp:
            continue
        add(change.get("id") or "")

    for msg in list_messages(base, channel, 200):
        meta = msg.get("metadata") or {}
        if isinstance(meta, dict):
            add(str(meta.get("registered_change_id") or ""))
        if (msg.get("type") or "") != "file_change":
            continue
        proposal = meta.get("file_change_proposal") if isinstance(meta, dict) else None
        if isinstance(proposal, dict):
            fp = (proposal.get("file_path") or proposal.get("FilePath") or "").lower()
            if needle and needle not in fp:
                continue
            add(str(proposal.get("change_id") or proposal.get("ChangeID") or ""))

    return ids


def wait_and_approve_file_changes(
    base: str,
    channel: str,
    *,
    path_contains: str = "",
    min_approved: int = 1,
    timeout: float = 60,
    user_id: str = "default",
) -> tuple[int, list[str]]:
    """Poll pending file changes on a channel and approve matching proposals."""
    deadline = time.time() + timeout
    approved: list[str] = []
    errors: list[str] = []
    while time.time() < deadline:
        for cid in pending_change_ids_for_channel(
            base, channel, path_contains=path_contains, user_id=user_id
        ):
            code, data = approve_file_change(base, cid, user_id)
            if code == 200:
                if cid not in approved:
                    approved.append(cid)
            else:
                err = data if isinstance(data, str) else json.dumps(data)
                errors.append(f"{cid}: HTTP {code} {err}")
        if len(approved) >= min_approved:
            return len(approved), approved
        time.sleep(POLL_INTERVAL)
    return len(approved), approved


# Loose agent formats (discussion-only; not registered with FileChangeManager)
_LOOSE_FILE_CHANGE_FENCED = re.compile(
    r"\[FILE_CHANGE\]\s*([^\s`\n]+).*?```(?:markdown|md|new)?\s*\n(.*?)```",
    re.DOTALL | re.IGNORECASE,
)
_LOOSE_FILE_CHANGE_INLINE = re.compile(
    r"\[FILE_CHANGE\]\s*([^\s`\n]+)\s*(.+)",
    re.DOTALL | re.IGNORECASE,
)


def _extract_loose_file_changes(content: str) -> list[tuple[str, str]]:
    out: list[tuple[str, str]] = []
    for m in _LOOSE_FILE_CHANGE_FENCED.finditer(content):
        out.append((m.group(1).strip().strip("`\"'"), m.group(2).strip()))
    if out:
        return out
    m = _LOOSE_FILE_CHANGE_INLINE.search(content)
    if m:
        out.append((m.group(1).strip().strip("`\"'"), m.group(2).strip()))
    return out


_TASK_STATUS_LINE = re.compile(r"(?im)^\s*TASK_STATUS:\s*\S+.*$")


def sanitize_deliverable_body(text: str) -> str:
    """Remove machine-readable task lines from file deliverable content."""
    cleaned = _TASK_STATUS_LINE.sub("", text)
    return re.sub(r"\n{3,}", "\n\n", cleaned).strip()


def _collect_bullet_findings(messages: list[dict], limit: int = 8) -> str:
    lines: list[str] = []
    for msg in messages:
        who = (msg.get("from") or {}).get("name") or ""
        if who in ("System", "ChatModerator"):
            continue
        typ = msg.get("type") or ""
        if typ not in DISCUSSION_TYPES and typ != "collaboration_discussion":
            continue
        for raw in (msg.get("content") or "").splitlines():
            line = raw.strip()
            if not line or line.startswith("[FILE_CHANGE]"):
                continue
            if re.match(r"^(\d+\.|[-*])\s+", line) or line.startswith("**"):
                lines.append(line)
            if len(lines) >= limit:
                break
        if len(lines) >= limit:
            break
    if not lines:
        return ""
    return sanitize_deliverable_body("# Findings\n\n" + "\n".join(lines) + "\n")


def write_loose_file_change_from_messages(
    messages: list[dict],
    workspace_root: str,
    *,
    path_contains: str = "",
    target_rel: str = "",
    min_bytes: int = 20,
) -> str | None:
    """
    When agents embed a non-canonical [FILE_CHANGE] block in discussion only,
    extract path + body and write under workspace_root (scenario fallback).
    """
    root = Path(workspace_root)
    needle = path_contains.lower()
    for msg in reversed(messages):
        typ = msg.get("type") or ""
        if typ not in DISCUSSION_TYPES and typ != "collaboration_discussion":
            continue
        content = msg.get("content") or ""
        blocks = _extract_loose_file_changes(content)
        if not blocks:
            continue
        for rel_path, body in blocks:
            if target_rel:
                use_path = target_rel
            elif needle and needle not in rel_path.lower():
                continue
            else:
                use_path = rel_path
            if len(body.encode()) < min_bytes:
                body = _collect_bullet_findings(messages) or body
            body = sanitize_deliverable_body(body)
            if len(body.encode()) < min_bytes:
                continue
            dest = root / use_path
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_text(body if body.endswith("\n") else body + "\n", encoding="utf-8")
            return str(dest)
    if target_rel:
        body = _collect_bullet_findings(messages)
        body = sanitize_deliverable_body(body)
        if len(body.encode()) >= min_bytes:
            dest = root / target_rel
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_text(body if body.endswith("\n") else body + "\n", encoding="utf-8")
            return str(dest)
    return None


def discussion_diagnosis(
    base: str,
    channel: str,
    *,
    required_agents: list[str] | None = None,
) -> str:
    """Human-readable summary of who spoke and common failure signals."""
    msgs = list_messages(base, channel, 200)
    pool = agent_messages(msgs)
    counts = count_by_agent(pool)
    lines = [f"agent discussion: total={len(pool)} counts={counts}"]
    for name in required_agents or []:
        n = counts.get(name, 0)
        if n < 1:
            lines.append(f"  FAIL: @{name} — no collaboration_discussion (silent or shouldRespond blocked)")
        else:
            lines.append(f"  ok: @{name} — {n} message(s)")
    handoffs = sum(
        1
        for m in msgs
        if isinstance(m, dict)
        and "Collaboration turn handoff" in (m.get("content") or "")
    )
    pending = len(list_pending_file_changes(base))
    lines.append(f"  system turn handoffs in channel: {handoffs}")
    lines.append(f"  pending file changes (hub): {pending}")
    return "\n".join(lines)


def verify_agents_online(base: str, names: list[str]) -> tuple[bool, list[str]]:
    """Return (ok, missing_names) for required agent display names (no @)."""
    want = {n.strip().lstrip("@") for n in names if n.strip()}
    if not want:
        return True, []
    code, data = hub_request(base, "GET", "/api/agents")
    if code != 200 or not isinstance(data, list):
        return False, sorted(want)
    online = {
        (a.get("name") or "").strip()
        for a in data
        if isinstance(a, dict) and not a.get("is_paused")
    }
    missing = sorted(n for n in want if n not in online)
    return len(missing) == 0, missing


def discover_agents(base: str, min_count: int = 2) -> str:
    env = os.environ.get("NJ_COLLAB_SCENARIO_AGENTS", "").strip()
    if env:
        return env
    code, data = hub_request(base, "GET", "/api/agents")
    if code != 200 or not isinstance(data, list):
        return AGENT_PROFILES["fast"]
    picks: list[str] = []
    for a in data:
        if not isinstance(a, dict) or a.get("is_paused"):
            continue
        if (a.get("type") or "").lower() == "moderator":
            continue
        name = (a.get("name") or "").strip()
        if not name:
            continue
        mention = f"@{name}"
        if mention not in picks:
            picks.append(mention)
        if len(picks) >= min_count:
            break
    return " ".join(picks) if len(picks) >= min_count else AGENT_PROFILES["fast"]
