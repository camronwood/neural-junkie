#!/usr/bin/env python3
"""Create GitHub issues from KNOWN_ISSUES + roadmap blockers + test failures."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

REPO = "camronwood/neural-junkie"
ROOT = Path(__file__).resolve().parents[1]

ISSUES = [
    # Batch A — limitations
    {
        "id": "collab-chat-not-disk",
        "title": "[limitation] Collab chat does not write files to disk",
        "labels": ["limitation", "collab"],
        "body": """## ID
`collab-chat-not-disk`

## Status
**Limitation** — intentional behavior

## Summary
Collab chat markdown does **not** write to disk. Execution needs `[FILE_CHANGE]` proposals and your approval in **Pending changes**. `TASK_STATUS: completed` alone does not create files.

## Workaround
Use approved file-change proposals in Pending changes for disk writes.

## Docs
- [KNOWN_ISSUES.md](https://github.com/camronwood/neural-junkie/blob/main/docs/KNOWN_ISSUES.md#collaboration)
- [COLLABORATION.md](https://github.com/camronwood/neural-junkie/blob/main/docs/COLLABORATION.md)
""",
    },
    {
        "id": "collab-model-variance",
        "title": "[limitation] Local model variance in collaboration",
        "labels": ["limitation", "collab", "reliability"],
        "body": """## ID
`collab-model-variance`

## Status
**Limitation** — environment / model dependent

## Summary
Local models (Ollama) vary in discussion quality, silence, and timeouts. Hub enforces phase caps and fallbacks but cannot guarantee plan shape from every model.

## Mitigation
- Settings → **Collaboration planning provider** (optional cloud/larger model for planning turns)
- Implementation sessions: **reliable tool model** / optional **reliable provider** (repair round 2+)

## Docs
- [KNOWN_ISSUES.md](https://github.com/camronwood/neural-junkie/blob/main/docs/KNOWN_ISSUES.md#collaboration)
""",
    },
    {
        "id": "collab-smart-routing-scope",
        "title": "[limitation] Smart routing applies to execution tasks only",
        "labels": ["limitation", "collab", "hub"],
        "body": """## ID
`collab-smart-routing-scope`

## Status
**Limitation** — intentional scope

## Summary
Collaboration smart routing applies to **execution tasks only**, not normal channel chat. Planning can use optional `planning_provider_id` in Settings (separate from smart routing).
""",
    },
    {
        "id": "hub-history-bounded",
        "title": "[limitation] Bounded per-channel message history",
        "labels": ["limitation", "hub"],
        "body": """## ID
`hub-history-bounded`

## Status
**Limitation** — intentional scope

## Summary
Per-channel history is capped (5000 messages) and age-pruned after 24h unless marked **durable**. SQLite sidecar + **Export history** in channel info; not a full in-app search archive.
""",
    },
    {
        "id": "slack-bridge-local",
        "title": "[limitation] Slack bridge requires local hub",
        "labels": ["limitation", "integrations"],
        "body": """## ID
`slack-bridge-local`

## Status
**Limitation** — architectural

## Summary
Slack bridge runs **in-process** on the local hub (Socket Mode). Hub must be running locally — no public URL required.

## Diagnostics
Settings → Integrations → Slack
""",
    },
    {
        "id": "confluence-setup",
        "title": "[limitation] Confluence agents require setup and indexing time",
        "labels": ["limitation", "integrations"],
        "body": """## ID
`confluence-setup`

## Status
**Limitation** — operational

## Summary
Confluence agents need Cloud credentials and indexing time; search quality depends on space size and token limits.
""",
    },
    {
        "id": "room-chat-lan",
        "title": "[limitation] Room chat requires same-LAN guests",
        "labels": ["limitation", "integrations"],
        "body": """## ID
`room-chat-lan`

## Status
**Limitation** — network topology

## Summary
Room chat pack requires guests on the **same LAN** as the host hub. Corporate Wi‑Fi client isolation may block joins. Host must enable **listen on LAN** and configure a hub token.
""",
    },
    {
        "id": "web-ui-thin",
        "title": "[limitation] Browser hub UI is a thin chat client",
        "labels": ["limitation", "desktop"],
        "body": """## ID
`web-ui-thin`

## Status
**Limitation** — intentional scope

## Summary
Browser hub UI at `/` is a lightweight chat client — no full workspace, palette, or file-approval UX. Use the **Tauri desktop** for production work.
""",
    },
    {
        "id": "git-dev-pack",
        "title": "[limitation] In-app Git requires Software development pack",
        "labels": ["limitation", "desktop"],
        "body": """## ID
`git-dev-pack`

## Status
**Limitation** — pack dependency

## Summary
In-app Git operations require the **Software development** pack, `git` on PATH, and a git workspace.
""",
    },
    {
        "id": "macos-adhoc-sign",
        "title": "[limitation] macOS GitHub Release builds are ad-hoc signed",
        "labels": ["limitation", "desktop", "infrastructure"],
        "body": """## ID
`macos-adhoc-sign`

## Status
**Limitation** — pending Apple Developer credentials

## Summary
GitHub Release macOS builds are **ad-hoc signed** until Apple Developer credentials are available. First launch may require **Right-click → Open**.

## Planned fix
**v1.2.1** notarized builds
""",
    },
    {
        "id": "single-hub",
        "title": "[limitation] Single-server hub deployment",
        "labels": ["limitation", "hub"],
        "body": """## ID
`single-hub`

## Status
**Limitation** — architectural (D4 deferred)

## Summary
**Single-server** deployment — no horizontal scale or multi-region hub yet.
""",
    },
    {
        "id": "react-tools-allowlist",
        "title": "[limitation] ReAct MCP tools are allowlist-based",
        "labels": ["limitation", "hub"],
        "body": """## ID
`react-tools-allowlist`

## Status
**Limitation** — intentional scope

## Summary
**ReAct MCP** on non-native tool models is allowlist-based (e.g. `gemma3:12b` first). Other tags fall back to native tools or Qwen swap.
""",
    },
    # Batch B — release blockers
    {
        "id": "blocker-collab-soak",
        "title": "[release-blocker] Collab-full layer gate failing",
        "labels": ["release-blocker", "collab", "reliability", "bug"],
        "body": """## Status
**Release blocker** — gate not green

## Gate
`make layer-gate LAYER=collab-full`

## Evidence
- [layer-gate-collab-full-2026-07-05-2233-iter2.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/layer-gate-collab-full-2026-07-05-2233-iter2.md)
- Multiple scenarios failing: `make-me-a-website`, `phoenix-resource-api-e2e`, `plan-dependency-prose-regression`, and others

## Symptoms
- Agent silence / `shouldRespond blocked`
- Discussion timeouts (`timed_out`)
- `generation_error` loops from cloud providers
- Response truncation due to timeout

## Done when
`make layer-gate LAYER=collab-full` PASS with no weakened scenario assertions
""",
    },
    {
        "id": "blocker-parity-soak",
        "title": "[release-blocker] Parity soak not confirmed (test-parity-stable)",
        "labels": ["release-blocker", "reliability", "bug"],
        "body": """## Status
**Release blocker** — gate not measured

## Gate
`make test-parity-stable` — 3× implement scenarios @ 20/20 with hub restart

## Evidence
- [reliability-pass-3-2026-06-28.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/reliability-pass-3-2026-06-28.md) — **Not run**

## Done when
`test-parity-stable` 3/3 @ 20/20 logged to `docs/testing/reliability-parity-soak-*.log`
""",
    },
    {
        "id": "blocker-platform-smoke",
        "title": "[release-blocker] Platform smoke pending operator sign-off",
        "labels": ["release-blocker", "desktop", "infrastructure"],
        "body": """## Status
**Release blocker** — Gate 5 pending

## Gate
[STABLE_RELEASE_CHECKLIST.md](https://github.com/camronwood/neural-junkie/blob/main/docs/STABLE_RELEASE_CHECKLIST.md) Gate 5

## Checklist
[stable-platform-smoke.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/stable-platform-smoke.md)

## Minimum
macOS arm64 + one of Windows x64 or Linux x64 on beta.5+ installer

## Done when
Gate 5 matrix signed off in STABLE_RELEASE_CHECKLIST.md
""",
    },
    {
        "id": "blocker-d5-deferred",
        "title": "[release-blocker] D5 specialist simplification deferred until parity green",
        "labels": ["release-blocker", "hub"],
        "body": """## Status
**Release blocker dependency** — epic deferred

## Epic
D5 — Specialist simplification ([PHASE_D_BACKLOG.md](https://github.com/camronwood/neural-junkie/blob/main/docs/PHASE_D_BACKLOG.md))

## Scope
- MCP dedup
- QualityReviewer roll-up
- Fewer redundant specialists

## Blocked on
Parity gate green (`make test-parity-stable` 3/3 @ 20/20)

## Done when
D5 work can start after parity soak passes; not a stable-cut blocker itself but tracked as roadmap dependency
""",
    },
    # Batch C — active/investigating from test failures
    {
        "id": "collab-agent-silence",
        "title": "[bug] Collab agents silent or blocked during planning discussions",
        "labels": ["bug", "collab", "reliability"],
        "body": """## Status
**Investigating** — reproduces in layer-gate collab-core

## Symptoms
- `wait_discussion: discussion timeout`
- `no collaboration_discussion (silent or shouldRespond blocked)`
- `planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/3`
- Agents like `@SoftwareArchitect`, `@PlatformEngineer` never post

## Failing scenarios (sample)
- `document-findings-execution`
- `collab-participation-three-agent`
- `planning-two-agent`
- `phoenix-resource-api-e2e`
- `plan-dependency-prose-regression`

## Evidence
- [layer-gate-collab-core-2026-07-10-1406.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/layer-gate-collab-core-2026-07-10-1406.md)
- [layer-gate-collab-full-2026-07-05-2233-iter2.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/layer-gate-collab-full-2026-07-05-2233-iter2.md)

## Related limitation
`collab-model-variance` — but this is an active reliability gap beyond normal variance

## Repro
```bash
make collab-preflight
make layer-gate LAYER=collab-core
```
""",
    },
    {
        "id": "collab-generation-error",
        "title": "[bug] Cloud provider generation_error loops during collab planning",
        "labels": ["bug", "collab", "reliability", "hub"],
        "body": """## Status
**Investigating** — reproduces in layer-gate collab-core/full

## Symptoms
- Repeated `**Claude** could not complete this turn: Sorry, I encountered an error while generating a response`
- Similar errors from **Gemini** and **Assistant** (timeout)
- High `generation_error` post counts (e.g. 13 errors in one scenario)
- Planning phase stuck with partial agent participation

## Failing scenarios (sample)
- `document-findings-execution`
- `collab-participation-three-agent`
- `collab-generation-error-resilience`
- `make-me-a-website`

## Evidence
- [layer-gate-collab-core-2026-07-10-1406.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/layer-gate-collab-core-2026-07-10-1406.md)
- [layer-gate-collab-full-2026-07-05-2233-iter2.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/layer-gate-collab-full-2026-07-05-2233-iter2.md)

## Repro
```bash
make collab-scenario SCENARIO=collab-participation-three-agent PROFILE=realistic
```
""",
    },
    {
        "id": "layer-gate-make-verbose-macos",
        "title": "[bug] layer-gate passes unsupported --verbose to make on macOS",
        "labels": ["bug", "infrastructure"],
        "body": """## Status
**Active** — harness bug blocks collab-core gate on macOS

## Symptoms
```
make collab-scenarios-core --verbose
/Library/Developer/CommandLineTools/usr/bin/make: unrecognized option `--verbose'
```

## Impact
`make layer-gate LAYER=collab-core` fails immediately (0s) on macOS with Xcode/CLT make — gate never runs scenarios.

## Evidence
- [layer-gate-collab-core-2026-07-12-1604.md](https://github.com/camronwood/neural-junkie/blob/main/docs/testing/layer-gate-collab-core-2026-07-12-1604.md)

## Fix direction
Use `VERBOSE=1` env var or Makefile target variable instead of `make --verbose` (BSD make incompatible)
""",
    },
]


def create_issue(item: dict) -> int:
    cmd = [
        "gh",
        "issue",
        "create",
        "--repo",
        REPO,
        "--title",
        item["title"],
        "--body",
        item["body"],
    ]
    for label in item["labels"]:
        cmd.extend(["--label", label])
    result = subprocess.run(cmd, capture_output=True, text=True, check=True)
    url = result.stdout.strip()
    number = int(url.rstrip("/").split("/")[-1])
    return number


def main() -> int:
    mapping: dict[str, int] = {}
    for item in ISSUES:
        number = create_issue(item)
        mapping[item["id"]] = number
        print(f"{item['id']}: #{number}")

    out = ROOT / "docs" / "testing" / "github-issues-map.json"
    out.write_text(json.dumps(mapping, indent=2) + "\n")
    print(f"\nWrote {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
