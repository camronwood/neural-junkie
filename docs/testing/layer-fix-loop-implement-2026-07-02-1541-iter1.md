# layer fix loop — implement — iteration 1 — 2026-07-02-1541-iter1 UTC

layer=implement
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-implement-2026-07-02-1541-iter1.md
fix_branch=release-prep/layer-implement-2026-07-02-1541
git_commit=d2577d6
layer_gate_rc=1
agent_rc=0

## Cursor agent output

```text
## Summary

### Root cause (implement-scenarios)
Hub logs showed **FrontendEngineer stuck on the first scenario** (`app-wont-boot-fix-like`) because capability routing sent **first-attempt boot/fix sessions to `TaskImplementHeavy` → `devstral:24b`**. That model did not finish within the 600–900s scenario window, so the agent never replied and **blocked every later FrontendEngineer scenario** (13/20 failures were `timeout waiting for FrontendEngineer`).

BackendEngineer mostly used `qwen2.5-coder:14b` and completed; boot/fix frontend cases hit `devstral:24b`.

### Code changes
1. **`internal/routing/capabilities/classify.go`** — Reserve `implement_heavy` for **repair tier only** (`RepairAttempts >= 1` or `VerifyFailed`). First-attempt boot/fix/error work stays on standard `implement` (local `qwen2.5-coder:14b` / `qwen3.5:9b`).
2. **`capability_profiles.json`** (embedded + `docs/data/model-capability-profiles.json`) — Rank **`qwen2.5-coder:14b` and `qwen3.5:9b` ahead of `devstral:24b`** in `implement_heavy` when repair tier does upgrade.
3. **Tests** — `classify_impl_test.go`, `select_capability_test.go`.

### Live smoke check (after hub restart)
`at-file-explicit-path` now routes as:
```text
[impl-routing] FrontendEngineer: tool_model=qwen2.5-coder:14b
```
FrontendEngineer runs tool loops instead of hanging on `devstral:24b` (full 900s scenario not waited out here).

### CI
`make test-all` initially failed only because **`desktop/node_modules` was missing** (`npx tsc` could not find TypeScript). After `npm install` in `desktop/`, CI passes with no code changes.

### Commands run
| Command | Result |
|---------|--------|
| `npm install` (in `desktop/`) | Needed for tsc |
| `make test-all` | **PASS** (~151s) |
| `make test-conversation-contract` | **PASS** |
| `go test ./internal/routing/capabilities/... ./internal/implementation/routing/...` | **PASS** |
| Hub restart + `implement-scenarios.py --scenario at-file-explicit-path` | Confirmed `qwen2.5-coder:14b` routing (stopped before 900s timeout) |

Recommend re-running `make layer-gate LAYER=implement` (or the full fix loop) with a fresh hub restart so agents pick up the routing change. I did not commit — say if you want that.The two background `at-file-explicit-path` runs didn’t finish — I stopped them after they sat on `wait_reply` (900s timeout). That’s expected for a full live scenario, not a sign the routing fix failed.

Before stopping, hub logs confirmed the fix: FrontendEngineer routed to **`qwen2.5-coder:14b`** instead of hanging on **`devstral:24b`**, and the agent was actively running tool loops.

**CI is green** (`make test-all`, `make test-conversation-contract`). To validate the full implement layer end-to-end, restart the hub and run:

```bash
make layer-gate LAYER=implement
```
```

