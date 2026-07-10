# layer fix loop — collab-full — iteration 1 — 2026-07-10-0103-iter1 UTC

layer=collab-full
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-10-0103-iter1.md
fix_branch=release-prep/layer-collab-full-2026-07-10-0103
git_commit=db9b72a
layer_gate_rc=2
agent_rc=0

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject (exit 0)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collaboration-station-website (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collaboration-station-website-sa (exit 1)

## Cursor agent output

```text
Connection lost, reconnecting to https://agentn.global.api5.cursor.sh (attempt 1)...
Retry attempt 1...
Background task results from the verification runs:

**Regression hub (51721)** — Started successfully; hub listening with agents booted (logs in `/tmp/nj-hub.log`).

**Live scenarios — all failed** (same participation pattern as the gate brief):

| Scenario | Outcome |
|----------|---------|
| `collab-human-planning-interject` | **FAIL** — connection reset mid-`wait_discussion` (likely hub overload from parallel runs) |
| `collab-participation-three-agent` | **FAIL** — BackendEngineer posted once; SoftwareArchitect and Claude silent (12 handoffs) |
| `planning-two-agent` (first run) | **FAIL** — SoftwareArchitect posted once; BackendEngineer silent (11 handoffs) |
| `collab-conversation-quality-regression` | **FAIL** — no agent posts; hub likely unavailable during parallel execution |
| `planning-two-agent` (rebuilt hub) | **FAIL** — zero discussion messages; SA hung ~6 min on `qwen3.5:27b` before cancel |

**CI is green** (`make test-go`, `make test-all`, `make test-conversation-contract`). The participation routing fixes are in the tree; live reruns need a clean sequential boot with models pinned to ≤14B (no `SKIP_BOOT=1`, no parallel scenarios) before the brief’s 18 scenario commands will be trustworthy.
```

