#!/usr/bin/env python3
"""Verify Model Arena pack is enabled and hub exposes arena capabilities."""

from __future__ import annotations

import json
import sys
import urllib.error
import urllib.request

HUB = os.environ.get("HUB", "http://127.0.0.1:18765")


def main() -> int:
    url = f"{HUB}/api/packs"
    try:
        with urllib.request.urlopen(url, timeout=5) as resp:
            data = json.load(resp)
    except urllib.error.URLError as exc:
        print(f"FAIL: hub not reachable at {url}: {exc}")
        return 1

    caps = data.get("capabilities") or []
    registry = data.get("capability_registry") or []
    packs = data.get("packs") or []

    arena = next((p for p in packs if p.get("id") == "model-arena"), None)
    print("model-arena pack:", json.dumps(arena, indent=2) if arena else "NOT INSTALLED")

    arena_caps = [c for c in caps if "arena" in c]
    arena_reg = [
        {
            "id": r.get("id"),
            "kind": r.get("kind"),
            "modal": (r.get("ui") or {}).get("modal"),
        }
        for r in registry
        if "arena" in (r.get("id") or "")
    ]
    print("\ncapabilities (arena):", arena_caps or "(none)")
    print("registry (arena):", json.dumps(arena_reg, indent=2) if arena_reg else "(none)")

    manifest_caps = (arena or {}).get("capabilities") or []
    workbench_ok = (
        "model-arena-workbench" in caps
        or "model-arena" in caps
        or "model-arena-workbench" in manifest_caps
        or any(r.get("id") == "model-arena-launcher" for r in registry)
    )
    chip_ok = any(r.get("id") == "model-arena-launcher" for r in registry)
    enabled = bool(arena and arena.get("enabled"))

    print("\nchecks:")
    print(f"  enabled: {enabled}")
    print(f"  toolbar chip registered: {chip_ok}")
    print(f"  workbench gate would pass: {workbench_ok}")

    try:
        with urllib.request.urlopen(f"{HUB}/api/arena/challenges", timeout=5) as resp:
            print(f"  /api/arena/challenges: HTTP {resp.status}")
    except urllib.error.HTTPError as exc:
        print(f"  /api/arena/challenges: HTTP {exc.code} ({exc.reason})")
        if exc.code == 403:
            print("  hint: rebuild/restart hub from neural-junkie main with arena routes")
    except urllib.error.URLError as exc:
        print(f"  /api/arena/challenges: {exc}")

    if not enabled:
        print("\nFAIL: enable Model Arena in Domain packs")
        return 1
    if not chip_ok:
        print("\nFAIL: model-arena-launcher missing from capability_registry (add to pack capabilities)")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
