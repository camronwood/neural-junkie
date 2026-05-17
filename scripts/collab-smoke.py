#!/usr/bin/env python3
"""
Collaboration smoke test — CI (in-process Go test) or live hub (optional).

Default (no flags): runs TestCollabSmokePhaseTransitions via go test (no running hub).

Live mode (--live): POST /collaborate to a running hub, poll GET /api/collaborations
for phase transitions. Requires agents to finish discussion for reviewing→executing;
otherwise only planning (+ optional cancel cleanup) is verified.

Examples:
  ./scripts/collab-smoke.py
  ./scripts/collab-smoke.py --live
  NEURAL_JUNKIE_HUB_URL=http://127.0.0.1:18765 ./scripts/collab-smoke.py --live
"""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DEFAULT_HUB = os.environ.get("NEURAL_JUNKIE_HUB_URL", "http://127.0.0.1:18765").rstrip("/")
SMOKE_CHANNEL = "collab-smoke"
POLL_SECS = 90
POLL_INTERVAL = 1.0
MAX_CONCURRENT_COLLABS = 3


def hub_request(base: str, method: str, path: str, body: dict | None = None) -> tuple[int, Any]:
    url = f"{base}{path}"
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
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


def run_go_smoke() -> int:
    print("collab-smoke: running in-process API test (go test -run TestCollabSmokePhaseTransitions)")
    cmd = [
        "go",
        "test",
        "./test",
        "-run",
        "TestCollabSmokePhaseTransitions",
        "-count=1",
        "-v",
    ]
    proc = subprocess.run(cmd, cwd=ROOT)
    return proc.returncode


def find_collab(collabs: list, collab_id: str) -> dict | None:
    for c in collabs:
        if isinstance(c, dict) and c.get("id") == collab_id:
            return c
    return None


def list_active_collaborations(base: str) -> list[dict]:
    code, data = hub_request(base, "GET", "/api/collaborations?include_terminal=false")
    if code != 200 or not isinstance(data, list):
        return []
    return [c for c in data if isinstance(c, dict)]


def cancel_collab(base: str, collab: dict) -> bool:
    cid = collab.get("id") or ""
    ch = collab.get("channel") or ""
    if not cid or not ch:
        return False
    code, _ = hub_request(
        base,
        "POST",
        "/api/send",
        {
            "channel": ch,
            "content": f"/cancel-plan {cid[:8]}",
            "type": "question",
            "from": {"name": "CollabSmoke", "type": "human"},
        },
    )
    return code == 200


def free_collab_capacity(base: str) -> bool:
    """Cancel prior smoke collabs; fail with a hint if the hub is still at capacity."""
    active = list_active_collaborations(base)
    smoke_markers = ("collab smoke", "collab-smoke", "nj collab smoke")
    cancelled = 0
    for c in active:
        title = (c.get("title") or "").lower()
        desc = (c.get("description") or "").lower()
        ch = (c.get("channel") or "").lower()
        if ch == SMOKE_CHANNEL or any(m in title or m in desc for m in smoke_markers):
            if cancel_collab(base, c):
                cancelled += 1
                print(f"  cancelled prior smoke collab {c.get('id', '')[:8]} ({c.get('phase')})")
    if cancelled:
        time.sleep(0.5)

    active = list_active_collaborations(base)
    if len(active) < MAX_CONCURRENT_COLLABS:
        return True

    print(
        f"FAIL: hub has {len(active)}/{MAX_CONCURRENT_COLLABS} active collaborations — "
        "free a slot with /cancel-plan <id> on its channel:",
        file=sys.stderr,
    )
    for c in active:
        print(
            f"    {c.get('id', '')[:8]}  phase={c.get('phase')}  channel={c.get('channel')}",
            file=sys.stderr,
        )
    return False


def last_system_error(base: str, channel: str) -> str:
    q = urllib.parse.urlencode({"channel": channel, "limit": "5"})
    code, data = hub_request(base, "GET", f"/api/messages?{q}")
    if code != 200 or not isinstance(data, list):
        return ""
    for m in reversed(data):
        if not isinstance(m, dict):
            continue
        if m.get("type") != "system_info":
            continue
        body = (m.get("content") or "").strip()
        if body.startswith("❌"):
            return body
    return ""


def ensure_channel(base: str, name: str) -> bool:
    """Create the smoke channel on a live hub if it does not exist yet."""
    code, chs = hub_request(base, "GET", "/api/channels")
    if code == 200 and isinstance(chs, list):
        if any(isinstance(c, dict) and c.get("name") == name for c in chs):
            return True
    code, out = hub_request(
        base,
        "POST",
        "/api/channels/create",
        {
            "name": name,
            "description": "Automated collab-smoke tests",
            "type": "public",
            "created_by": "CollabSmoke",
        },
    )
    if code in (200, 201):
        return True
    print(f"FAIL: could not create channel {name!r} ({code}): {out}", file=sys.stderr)
    return False


def collab_phase(base: str, channel: str, collab_id: str) -> str | None:
    q = urllib.parse.urlencode({"channel": channel, "include_terminal": "true"})
    code, data = hub_request(base, "GET", f"/api/collaborations?{q}")
    if code != 200 or not isinstance(data, list):
        return None
    c = find_collab(data, collab_id)
    return c.get("phase") if c else None


def wait_phase(base: str, channel: str, collab_id: str, want: str, timeout: float) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        phase = collab_phase(base, channel, collab_id)
        if phase == want:
            return True
        time.sleep(POLL_INTERVAL)
    return False


def run_live(base: str, agents: str, channel: str) -> int:
    print(f"collab-smoke (live): hub={base} channel={channel}")
    code, health = hub_request(base, "GET", "/api/health")
    if code != 200:
        print(f"FAIL: hub not healthy at {base} (status {code})", file=sys.stderr)
        print("Start with: make server", file=sys.stderr)
        return 1
    print(f"  health: {health.get('status')} agents={health.get('agent_count')}")

    if not ensure_channel(base, channel):
        return 1

    if not free_collab_capacity(base):
        return 1

    content = (
        f"/collaborate --rounds 1 --messages 2 {agents} "
        "nj collab smoke live probe (auto cleanup)"
    )
    code, send = hub_request(
        base,
        "POST",
        "/api/send",
        {
            "channel": channel,
            "content": content,
            "type": "question",
            "from": {"name": "CollabSmoke", "type": "human"},
        },
    )
    if code != 200 or not isinstance(send, dict):
        print(f"FAIL: POST /api/send ({code}): {send}", file=sys.stderr)
        return 1

    collab_id = send.get("collaboration_id") or ""
    collab_ch = send.get("collaboration_channel") or ""
    if not collab_id or not collab_ch:
        err = last_system_error(base, channel)
        print(f"FAIL: missing collaboration redirect in response: {send}", file=sys.stderr)
        if err:
            print(f"  hub: {err}", file=sys.stderr)
        return 1
    print(f"  created: id={collab_id[:8]} channel={collab_ch}")

    steps_ok = True

    if wait_phase(base, collab_ch, collab_id, "planning", 5):
        print("  ✓ phase planning")
    else:
        print("  ✗ expected phase planning", file=sys.stderr)
        steps_ok = False

    print(f"  polling for reviewing (up to {POLL_SECS}s; agents must discuss)...")
    if wait_phase(base, collab_ch, collab_id, "reviewing", POLL_SECS):
        print("  ✓ phase reviewing")
        code, _ = hub_request(
            base,
            "POST",
            "/api/send",
            {
                "channel": collab_ch,
                "content": f"/approve-plan {collab_id[:8]}",
                "type": "question",
                "from": {"name": "CollabSmoke", "type": "human"},
            },
        )
        if code == 200 and wait_phase(base, collab_ch, collab_id, "executing", 10):
            print("  ✓ phase executing")
            ack_code, _ = hub_request(
                base,
                "POST",
                "/api/collaboration-workspace-ack",
                {"collaboration_id": collab_id},
            )
            if ack_code == 204:
                print("  ✓ workspace ack")
            else:
                print(f"  ✗ workspace ack failed ({ack_code})", file=sys.stderr)
                steps_ok = False
        else:
            print("  ✗ approve-plan or executing phase", file=sys.stderr)
            steps_ok = False
    else:
        print(
            "  ⚠ still not reviewing — agents may be offline; run in-process: make collab-smoke",
            file=sys.stderr,
        )
        steps_ok = False

    # Cleanup
    hub_request(
        base,
        "POST",
        "/api/send",
        {
            "channel": collab_ch,
            "content": f"/cancel-plan {collab_id[:8]}",
            "type": "question",
            "from": {"name": "CollabSmoke", "type": "human"},
        },
    )
    if wait_phase(base, collab_ch, collab_id, "cancelled", 5):
        print("  ✓ cancelled (cleanup)")
    else:
        print("  ⚠ cancel-plan did not reach cancelled (may already be terminal)")

    if steps_ok:
        print("\ncollab-smoke (live): PASS")
        return 0
    print("\ncollab-smoke (live): FAIL (use `make collab-smoke` for full API path without agents)", file=sys.stderr)
    return 1


def main() -> int:
    p = argparse.ArgumentParser(description="Neural Junkie collaboration smoke test")
    p.add_argument(
        "--live",
        action="store_true",
        help="Run against a live hub (needs agents for full phase path)",
    )
    p.add_argument("--hub", default=DEFAULT_HUB, help="Hub base URL for --live")
    p.add_argument(
        "--agents",
        default="@RustExpert @SecurityExpert",
        help="Agent mentions for /collaborate (live mode)",
    )
    p.add_argument(
        "--channel",
        default=SMOKE_CHANNEL,
        help=f"Channel for /collaborate in live mode (default: {SMOKE_CHANNEL})",
    )
    args = p.parse_args()
    if args.live:
        return run_live(args.hub.rstrip("/"), args.agents.strip(), args.channel.strip())
    return run_go_smoke()


if __name__ == "__main__":
    sys.exit(main())
