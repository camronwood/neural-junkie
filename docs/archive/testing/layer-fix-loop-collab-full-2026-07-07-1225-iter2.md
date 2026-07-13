# layer fix loop — collab-full — iteration 2 — 2026-07-07-1225-iter2 UTC

layer=collab-full
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-07-1225-iter2.md
fix_branch=release-prep/layer-collab-full-2026-07-07-1225
git_commit=6bb9932
layer_gate_rc=2
agent_rc=0

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict (exit 1)

## Cursor agent output

```text
Connection lost, reconnecting to https://agentn.global.api5.cursor.sh (attempt 1)...
Retry attempt 1...
Those background runs were interrupted before they finished — they were killed mid-scenario, not by a clean script failure.

**704129 (4 collab scenarios):** `collab-participation-two-agent-strict` and `plan-distinct-deliverables-same-agent` both failed on `wait_discussion` (0 real agent messages). Killed during `collab-human-planning-interject` (~15m in).

**704130 (conversation contract + 5 scenarios):** `make test-conversation-contract` passed (90/90 Vitest). Same two participation scenarios failed; killed during `collab-human-planning-interject` (~15m in). `planning-two-agent` and `collab-participation-three-agent` never ran.

**704131 (`planning-two-agent`):** Killed after ~7m with no scenario output captured.

CI from the earlier pass is still green (`make test-all`, `make test-conversation-contract`). The code fixes are in place; live collab verification was incomplete because those long-running scenario jobs were stopped. Re-run the brief’s scenario list with a rebuilt/restarted hub when you want the full collab sweep.
```

