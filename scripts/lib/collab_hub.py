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
    "fast": "@Assistant",
    "realistic": "@SoftwareArchitect @BackendEngineer",
}

DISCUSSION_TYPES = frozenset(
    {
        "collaboration_discussion",
        "answer",
        "chat",
    }
)

CHAT_REPLY_TYPES = frozenset({"chat", "answer"})


def hub_request(
    base: str,
    method: str,
    path: str,
    body: dict | None = None,
    *,
    max_retries: int = 3,
) -> tuple[int, Any]:
    from lib.hub_auth import ensure_hub_auth_headers

    url = f"{base.rstrip('/')}{path}"
    data = None
    headers = {"Accept": "application/json", **ensure_hub_auth_headers(base)}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    last_err: Exception | None = None
    for attempt in range(max_retries):
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
            if e.code in (429, 500, 502, 503, 504) and attempt + 1 < max_retries:
                time.sleep(2.0 * (attempt + 1))
                continue
            return e.code, parsed
        except (urllib.error.URLError, TimeoutError) as e:
            last_err = e
            if attempt + 1 < max_retries:
                time.sleep(2.0 * (attempt + 1))
                continue
    if last_err is not None:
        return 0, str(last_err)
    return 0, "request failed"


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


def channel_has_agent(base: str, channel: str, agent_name: str) -> bool:
    want = agent_name.strip().lstrip("@")
    code, chs = hub_request(base, "GET", "/api/channels")
    if code != 200 or not isinstance(chs, list):
        return False
    for ch in chs:
        if not isinstance(ch, dict) or ch.get("name") != channel:
            continue
        for ag in ch.get("agents") or []:
            if isinstance(ag, dict) and (ag.get("name") or "").strip() == want:
                return True
    return False


def join_agent_to_channel(base: str, channel: str, agent_name: str, *, max_retries: int = 3) -> bool:
    """Join an online agent to channel by display name (hub membership + discoverChannels)."""
    if channel_has_agent(base, channel, agent_name):
        return True
    agent_id = resolve_agent_id(base, agent_name)
    if not agent_id:
        return False
    payload = {"agent_id": agent_id, "channel": channel}
    for attempt in range(max_retries):
        code, _ = hub_request(base, "POST", "/api/channels/join", payload)
        if code == 200:
            return True
        if code == 429 and attempt + 1 < max_retries:
            time.sleep(2.0 * (attempt + 1))
            continue
        return False
    return False


def join_agents_to_channel(base: str, channel: str, agent_names: list[str]) -> tuple[bool, list[str]]:
    """Join all named agents; return (ok, failed_names)."""
    failed: list[str] = []
    for name in agent_names:
        want = name.strip().lstrip("@")
        if not want:
            continue
        if not join_agent_to_channel(base, channel, want):
            failed.append(want)
    return len(failed) == 0, failed


def ensure_channel_with_agents(
    base: str,
    name: str,
    agent_names: list[str],
    description: str = "Automated scenario tests",
) -> tuple[bool, list[str]]:
    """Ensure public channel exists and required agents are joined."""
    if not ensure_channel(base, name, description):
        return False, agent_names
    need_join = [
        n
        for n in agent_names
        if n.strip() and not channel_has_agent(base, name, n)
    ]
    ok, failed = join_agents_to_channel(base, name, agent_names)
    if ok and need_join:
        # In-process agents pick up new channels via discoverChannels (~1s).
        time.sleep(2.0)
    return ok, failed


def send_message(
    base: str,
    channel: str,
    content: str,
    *,
    metadata: dict | None = None,
    from_name: str = "CollabScenario",
    max_retries: int = 3,
) -> tuple[int, dict | None]:
    payload: dict[str, Any] = {
        "channel": channel,
        "content": content,
        "type": "question",
        "from": {"name": from_name, "type": "human"},
    }
    if metadata:
        payload["metadata"] = metadata
    for attempt in range(max_retries):
        code, data = hub_request(base, "POST", "/api/send", payload)
        if code == 429 and attempt + 1 < max_retries:
            time.sleep(2.0 * (attempt + 1))
            continue
        if code == 200 and isinstance(data, dict):
            return code, data
        return code, data if isinstance(data, dict) else None
    return 429, None


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
    cid = (collab_id or "").strip()
    if cid:
        code, data = hub_request(base, "GET", f"/api/collaborations/{urllib.parse.quote(cid, safe='')}")
        if code == 200 and isinstance(data, dict) and data.get("id") == cid:
            return data
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
    nudged = False
    while time.time() < deadline:
        collab = fetch_collab(base, channel, collab_id)
        if collab is None:
            return False
        status = (collab.get("planning_recap_status") or "").strip().lower()
        if status in ("complete", "failed"):
            return True
        agent_id = (collab.get("planning_recap_agent_id") or "").strip()
        if status == "pending" and agent_id and not nudged and (deadline - time.time()) < timeout * 0.5:
            name = agent_id
            code, data = hub_request(base, "GET", "/api/agents")
            if code == 200 and isinstance(data, list):
                for ag in data:
                    if isinstance(ag, dict) and (ag.get("id") or "").strip() == agent_id:
                        name = (ag.get("name") or agent_id).strip()
                        break
            send_message(base, channel, f"@{name} — please post the planning session recap for the user.")
            nudged = True
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
    markers: tuple[str, ...] = ("collab-scenarios", "planning-two-agent probe"),
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
        if body.startswith("❌") or body.startswith("⚠️"):
            return body
        if "Ignored" in body and "workspace" in body:
            return body
        if "deliverables folder" in body:
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


def chat_agent_messages(
    messages: list[dict],
    *,
    exclude_system: bool = True,
) -> list[dict]:
    """Agent chat/answer messages for general conversation scenarios."""
    return agent_messages(messages, types=CHAT_REPLY_TYPES, exclude_system=exclude_system)


def count_chat_agent_messages(messages: list[dict], from_agent: str | None = None) -> int:
    pool = chat_agent_messages(messages)
    if not from_agent:
        return len(pool)
    want = from_agent.strip().lstrip("@")
    return sum(1 for m in pool if (m.get("from") or {}).get("name") == want)


def is_agent_failure_message(msg: dict) -> bool:
    """True for system_info or chat rows that report generation/timeout failures."""
    if not isinstance(msg, dict):
        return False
    body = (msg.get("content") or "").lower()
    if not body:
        return False
    markers = (
        "encountered an error while generating",
        "timed out before completion",
        "provider_error",
        "try again",
    )
    if msg.get("type") == "system_info":
        return any(m in body for m in markers[:3])
    if msg.get("type") in CHAT_REPLY_TYPES:
        return markers[0] in body
    return False


def wait_chat_reply(
    base: str,
    channel: str,
    *,
    from_agent: str,
    baseline_count: int,
    timeout: float,
    max_new: int = 1,
    detect_failures: bool = False,
) -> tuple[bool, str]:
    """Poll until from_agent posts max_new chat/answer messages above baseline_count."""
    want = from_agent.strip().lstrip("@")
    deadline = time.time() + timeout
    while time.time() < deadline:
        msgs = list_messages(base, channel, 200)
        pool = chat_agent_messages(msgs)
        agent_msgs = [m for m in pool if (m.get("from") or {}).get("name") == want]
        new_count = len(agent_msgs) - baseline_count
        if new_count >= max_new:
            last = agent_msgs[-1]
            if is_agent_failure_message(last):
                if detect_failures:
                    return False, f"@{want} returned failure reply"
            else:
                return True, f"{want} replied ({new_count} new)"
        if detect_failures:
            failures = [
                m
                for m in msgs
                if (m.get("from") or {}).get("name") == want and is_agent_failure_message(m)
            ]
            if len(failures) > 0:
                return False, f"@{want} posted failure system message"
        time.sleep(POLL_INTERVAL)
    counts = count_by_agent(chat_agent_messages(list_messages(base, channel, 50)))
    return False, f"timeout waiting for @{want} (baseline={baseline_count}, counts={counts})"


def channel_interject(
    base: str,
    channel: str,
    *,
    held_by: str = "ChatScenario",
) -> tuple[bool, str]:
    """POST /api/channels/:channel/interject — hold agents until the user sends again."""
    ch = urllib.parse.quote(channel.strip(), safe="")
    code, data = hub_request(
        base,
        "POST",
        f"/api/channels/{ch}/interject",
        {"held_by": held_by},
    )
    if code != 200:
        detail = data if isinstance(data, str) else json.dumps(data)
        return False, f"interject HTTP {code}: {detail}"
    return True, f"channel {channel!r} held"


def wait_no_new_chat_replies(
    base: str,
    channel: str,
    *,
    from_agent: str,
    baseline_count: int,
    duration: float,
) -> tuple[bool, str]:
    """Fail if from_agent posts above baseline_count within duration seconds."""
    want = from_agent.strip().lstrip("@")
    deadline = time.time() + duration
    while time.time() < deadline:
        msgs = list_messages(base, channel, 200)
        pool = chat_agent_messages(msgs)
        agent_msgs = [m for m in pool if (m.get("from") or {}).get("name") == want]
        new_count = len(agent_msgs) - baseline_count
        if new_count > 0:
            return False, f"@{want} posted {new_count} new message(s) while channel held"
        time.sleep(min(POLL_INTERVAL, 0.5))
    return True, f"no new replies from @{want} for {duration:.0f}s (baseline={baseline_count})"


def resolve_agent_id(base: str, agent_name: str) -> str | None:
    """Resolve display name to hub agent id."""
    want = agent_name.strip().lstrip("@")
    code, data = hub_request(base, "GET", "/api/agents")
    if code != 200 or not isinstance(data, list):
        return None
    for a in data:
        if not isinstance(a, dict):
            continue
        if (a.get("name") or "").strip() == want:
            return (a.get("id") or "").strip() or None
    return None


def ensure_dm_channel(base: str, user: str, agent_name: str) -> str | None:
    """Create or return DM channel name between user and agent (by display name)."""
    agent_id = None
    for attempt in range(3):
        agent_id = resolve_agent_id(base, agent_name)
        if agent_id:
            break
        if attempt + 1 < 3:
            time.sleep(1.0 * (attempt + 1))
    if not agent_id:
        return None
    body = {
        "type": "dm",
        "created_by": user,
        "members": [agent_id],
        "description": "Chat scenario DM",
    }
    code, data = None, None
    for attempt in range(3):
        code, data = hub_request(base, "POST", "/api/channels/create", body)
        if code in (200, 201):
            break
        if attempt + 1 < 3 and code in (429, 500, 502, 503, 504):
            time.sleep(2.0 * (attempt + 1))
            continue
        break
    if code not in (200, 201) or not isinstance(data, dict):
        return None
    name = (data.get("name") or "").strip()
    return name or None


def clear_channel_history(base: str, channel: str, *, max_retries: int = 3) -> bool:
    payload = {"name": channel}
    for attempt in range(max_retries):
        code, _ = hub_request(base, "POST", "/api/channels/clear-history", payload)
        if code == 200:
            return True
        if code == 429 and attempt + 1 < max_retries:
            time.sleep(2.0 * (attempt + 1))
            continue
        return False
    return False


def fetch_debug_context(
    base: str,
    channel: str,
    message: str,
    *,
    conversation_mode: str | None = None,
    context_scope: str | None = None,
) -> dict | None:
    q: dict[str, str] = {"channel": channel, "message": message}
    if conversation_mode:
        q["conversation_mode"] = conversation_mode
    if context_scope:
        q["context_scope"] = context_scope
    query = urllib.parse.urlencode(q)
    code, data = hub_request(base, "GET", f"/api/debug/channel-context?{query}")
    if code == 200 and isinstance(data, dict):
        return data
    return None


def resolve_agents(profile: str | None, override: str | None) -> str:
    if override and override.strip():
        return override.strip()
    key = (profile or os.environ.get("NJ_SCENARIO_PROFILE") or "fast").strip().lower()
    return AGENT_PROFILES.get(key, AGENT_PROFILES["fast"])


def parse_agent_mentions(text: str) -> list[str]:
    """Extract @AgentName tokens from collaborate command or goal text."""
    if not text:
        return []
    names: list[str] = []
    seen: set[str] = set()
    for match in re.finditer(r"@([A-Za-z][A-Za-z0-9_-]*)", text):
        name = match.group(1).strip()
        if name and name not in seen:
            seen.add(name)
            names.append(name)
    return names


def collaborate_agent_names(scenario: dict, agents: str) -> list[str]:
    """Agent roster from explicit scenario agents string and required_agents."""
    names = parse_agent_mentions(agents)
    required = scenario.get("required_agents") or []
    if isinstance(required, list):
        for raw in required:
            name = str(raw).strip().lstrip("@")
            if name and name not in names:
                names.append(name)
    return names


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
_TASK_STATUS_ANY = re.compile(r"TASK_STATUS:\s*\S+", re.I)


def sanitize_deliverable_body(text: str) -> str:
    """Remove machine-readable task lines from file deliverable content."""
    cleaned = _TASK_STATUS_LINE.sub("", text)
    cleaned = _TASK_STATUS_ANY.sub("", cleaned)
    return re.sub(r"\n{3,}", "\n\n", cleaned).strip()


def _collect_bullet_findings(messages: list[dict], limit: int = 8) -> str:
    lines: list[str] = []
    for msg in messages:
        who = (msg.get("from") or {}).get("name") or ""
        if who in ("System",):
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


def messages_for_collab(messages: list[dict], collab_id: str) -> list[dict]:
    """Filter messages whose metadata collaboration_id matches collab_id."""
    if not collab_id:
        return messages
    cid = collab_id.strip()
    out: list[dict] = []
    for m in messages:
        meta = m.get("metadata") or {}
        if not isinstance(meta, dict):
            continue
        if (meta.get("collaboration_id") or "").strip() == cid:
            out.append(m)
            continue
        # Agents sometimes omit collaboration_id on error-path discussion posts.
        if (m.get("type") or "") == "collaboration_discussion" and not (
            meta.get("collaboration_id") or ""
        ).strip():
            out.append(m)
    return out


def planning_discussion_ready(base: str, channel: str, collab_id: str) -> bool:
    """True when planning discussion finished or phase advanced past planning."""
    collab = fetch_collab(base, channel, collab_id)
    if not collab:
        return False
    phase = (collab.get("phase") or "").strip().lower()
    if phase == "reviewing":
        return True
    disc = collab.get("discussion") or {}
    status = (disc.get("status") or "").strip().lower()
    if status and status != "active":
        return True
    total = int(disc.get("total_message_count") or 0)
    max_msgs = int(disc.get("max_total_messages") or 0)
    if max_msgs > 0 and total >= max_msgs:
        return True
    return False


def discussion_diagnosis(
    base: str,
    channel: str,
    *,
    required_agents: list[str] | None = None,
    collab_id: str = "",
) -> str:
    """Human-readable summary of who spoke and common failure signals."""
    msgs = messages_for_collab(list_messages(base, channel, 200), collab_id)
    pool = agent_messages(msgs)
    counts = count_by_agent(pool)
    lines = [f"agent discussion: total={len(pool)} counts={counts}"]
    gen_errors = sum(
        1
        for m in msgs
        if isinstance(m, dict)
        and (m.get("type") or "") == "collaboration_discussion"
        and isinstance(m.get("metadata"), dict)
        and m["metadata"].get("generation_error")
    )
    if gen_errors:
        lines.append(f"  generation_error posts in channel: {gen_errors}")

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
    if collab_id:
        ready = planning_discussion_ready(base, channel, collab_id)
        collab = fetch_collab(base, channel, collab_id)
        disc = (collab or {}).get("discussion") or {}
        lines.append(
            f"  planning_discussion_ready={ready} "
            f"phase={(collab or {}).get('phase')!r} "
            f"discussion.status={disc.get('status')!r} "
            f"msgs={disc.get('total_message_count')}/{disc.get('max_total_messages')}"
        )
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
