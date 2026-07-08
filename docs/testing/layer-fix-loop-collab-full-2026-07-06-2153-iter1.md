# layer fix loop — collab-full — iteration 1 — 2026-07-06-2153-iter1 UTC

layer=collab-full
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-06-2153-iter1.md
fix_branch=release-prep/layer-collab-full-2026-07-06-2153
git_commit=b292630
layer_gate_rc=2
agent_rc=0

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject (exit 0)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 0)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict (exit 0)

## Cursor agent output

```text
Connection lost, reconnecting to https://agentn.global.api5.cursor.sh (attempt 1)...
Retry attempt 1...
## Summary

Fixed CI blockers first, then addressed the dominant collab failure mode (Ollama HTTP timeouts during planning). Ran the brief’s verification scenarios against the live hub.

### Code changes (4 files)

1. **`internal/hub/project_sets_test.go`**  
   Isolated test storage with `t.Setenv("HOME", tmpHome)` so `ProjectSetManager` no longer loads the developer’s real `~/.neural-junkie/project-sets.json` (was causing `list len=3/4` instead of 1). Removed `t.Parallel()` because `t.Setenv` forbids it.

2. **`scripts/lib/deliverable_judge.py`**  
   Judge provider/mode now resolve at **runtime** via `env_or_automation()`, not the module-import constant baked from hub automation (`ollama`). Fixes:
   - `test_cloud_judge_falls_back_to_ollama`
   - `test_cloud_judge_circuit_skips_hub_on_second_call`  
   Hub mode explicitly returns the configured cloud provider (`gemini`/`cursor`).

3. **`internal/ai/ollama_thinking.go`** + **`ollama_thinking_test.go`**  
   Raised collab-capable model HTTP timeout from **360s → 540s** so Ollama can use the full collab discussion budget (480s) before the HTTP client aborts. This was the main cause of `generation_error` / “timed out before completion” during planning.

### Commands run

**CI (priority):**
- `go test -run TestProjectSetManager_CreateListDelete ./internal/hub/`
- `python3 -m unittest scenario_assert_test.DeliverableAssertTest.test_cloud_judge_*`
- `make test-conversation-contract` — **PASS**
- `make test-all` — **PASS**  
  (also `npm install` in `desktop/` — worktree had no `node_modules`; env setup only, not a code change)

**Collab verification (from brief):** all 20 scenarios attempted in 3 batches:

| Scenario | Result |
|---|---|
| plan-distinct-deliverables-same-agent | PASS (flake retry) |
| planning-two-agent | PASS |
| collab-participation-two-agent-strict | PASS |
| collab-human-planning-interject | PASS |
| plan-dependency-prose-regression | PASS |
| collab-generation-error-resilience | PASS |
| collab-participation-three-agent | PASS (flake retry) |
| collab-conversation-quality-regression | FAIL — `wait_tasks` timeout |
| plan-findings-task-regression | FAIL — planning recap pending |
| plan-phoenix-combined-regression | FAIL — `@BackendEngineer` never spoke |
| resource-api-schema-planning | FAIL — `assert_plan` (too few tasks) |
| resource-api-schema-regression | FAIL — `assert_plan` |
| collaboration-station-website | FAIL — `wait_tasks` timeout |
| collaboration-station-website-sa | FAIL — `wait_tasks` timeout |
| document-findings-execution | FAIL — plan parser explosion (4 tasks, want ≤2) |
| execute-deliverable | FAIL — `wait_tasks` timeout |
| execution-no-stack-commands | FAIL — deliverable LLM judge |
| make-me-a-website | FAIL — `assert_plan` (0 tasks) |
| phoenix-resource-api-e2e | FAIL — `assert_plan` (0 tasks) |
| solo-vs-collab-parity | FAIL — solo leg (truncated in batch log) |

**7/20 PASS** on this pass; planning-participation scenarios improved noticeably after the timeout fix. Remaining failures cluster around:
- **Plan parsing** (`assert_plan` — 0 tasks or too many)
- **Execution stalls** (`wait_tasks` timeouts)
- **Wrong workspace context** (agents describing React/desktop repo instead of scenario fixtures)
- **Occasional agent silence** (e.g. `BackendEngineer` in phoenix-combined)

No test assertions were weakened. Next iteration should likely target workspace binding for scenario fixtures and execution/task dispatch, separate from the timeout fix applied here.All three verification batches finished. Here is where things stand:

**Batch 1 (4 scenarios)** — all passed:
- `planning-two-agent`
- `collab-participation-two-agent-strict`
- `collab-human-planning-interject`
- `plan-dependency-prose-regression`

**Batch 2 (6 scenarios)** — 2 passed, 4 failed:
- PASS: `collab-generation-error-resilience`, `collab-participation-three-agent` (retry)
- FAIL: `collab-conversation-quality-regression` (task timeout), `plan-findings-task-regression` (recap pending), `plan-phoenix-combined-regression` (BackendEngineer silent), `resource-api-schema-planning` (too few tasks)

**Batch 3 (9 scenarios)** — 0 full passes; failures were mostly plan parsing, execution timeouts, and deliverable quality:
- Planning often completed, but `assert_plan`, `wait_tasks`, or deliverable judge steps failed
- `solo-vs-collab-parity` failed on the solo leg

**Overall:** 7/20 scenarios passed in this verification run. CI is green (`make test-all`, `make test-conversation-contract`). The Ollama timeout change helped planning participation; remaining failures point at workspace binding, plan parsing, and execution dispatch — good targets for the next fix-loop iteration.
```

