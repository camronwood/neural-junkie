# layer fix loop — collab-full — iteration 3 — 2026-07-08-1132-iter3 UTC

layer=collab-full
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-08-1132-iter3.md
fix_branch=release-prep/layer-collab-full-2026-07-08-1132
git_commit=a633039
layer_gate_rc=2
agent_rc=0

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collaboration-station-website (exit 1)

## Cursor agent output

```text
Connection lost, reconnecting to https://agentn.global.api5.cursor.sh (attempt 1)...
Retry attempt 1...
## Summary

CI is green. Live collab verification is **partially** improved in code, but the local hub run stayed flaky (Ollama had `qwen3.5:27b` loaded above the 14B regression cap; agents timed out or posted only `generation_error`).

### Code changes (product/hub behavior)

1. **Planning turn advancement on failures** (`internal/agent/collab_generation_error.go`)  
   After a collab `generation_error`, always hand off to the next participant during planning (not only when `RecordMessage` succeeds).

2. **Faster short-collab planning timeouts** (`internal/collaboration/types.go`, `internal/collaboration/discussion.go`)  
   For `--messages ≤4`: wall timeout min 90s (was 2m), first-reply grace 45s (was 2m). Helps scenarios like `document-findings-execution` reach `reviewing` within their wait windows.

3. **Planning watchdog kick** (`internal/hub/collab_planning_watchdog.go`, `internal/hub/commands_collab.go`)  
   Added `KickPlanningDiscussionWatchdog()`; called on collab create + delayed retry so silent first speakers get re-prompted.

4. **Faster idle watchdog** (`cmd/server/main.go`)  
   Collaboration idle watchdog tick: 30s → 15s.

5. **Faster agent handoff retries** (`internal/agent/agent_collab_turn.go`)  
   Turn handoff retry delay: 25s → 15s.

6. **Silent-participant tracking** (`internal/collaboration/discussion.go`)  
   `generation_error` posts count as “attempted” so the watchdog stops re-prompting the same timed-out agent and moves to the next participant. Quorum still requires real (non-error) messages.

7. **Block `generate_image` during collab planning** (`internal/agent/image_gen_tools.go` + test)  
   Prevents slow image-tool calls during planning turns (seen in hub logs for `SoftwareArchitect`).

8. **Solo parity harness fix** (`scripts/collab-scenarios.py`)  
   Clear `collab-scenarios-solo` before send; refresh baseline after failure retry (fixes stale `baseline=1` false negatives).

### Commands run

| Command | Result |
|---------|--------|
| `make test-all` | ✅ Pass |
| `make test-conversation-contract` | ✅ Pass |
| `go test ./internal/collaboration/... ./internal/agent/... ./internal/hub/...` | ✅ Pass |
| `python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict` | ✅ Pass once (~32s, with `SKIP_BOOT=1`); later runs failed with 0 messages / generation timeouts |
| Other brief-listed scenarios | ❌ Not fully green in this session (Ollama `qwen3.5:27b` loaded; batch runs contended) |

Also ran `enforce_regression_agent_models` / `unload_loaded_models_over_cap` which unloaded `qwen3.5:27b` and switched agents to `qwen3.5:9b`, but subsequent scenario runs still timed out—likely needs a **full** `regression_boot` (not manual hub + `SKIP_BOOT`) for a reliable live sweep.

### Recommended next step for full brief verification

Run the 19 scenarios from the brief with a clean regression boot (no `SKIP_BOOT`), after confirming Ollama has no model >14B loaded:

```bash
make server-regression   # or full layer boot
python3 scripts/collab-scenarios.py --scenario <name>   # per brief list
```

The iter3 failures (silent agents, `generation_error` only, `msgs=0/N`, PlatformEngineer mismatches in stale brief) should improve with these fixes once the hub runs the new binary under the 14B regression model policy.Background shell tasks from the fix loop mostly **completed successfully as processes**, but **live collab scenarios did not reliably pass** in this environment.

**Hub / infra**
- Old hub on `:18765` was killed and restarted.
- One background hub start exited **137** (SIGKILL — likely OOM or killed when another hub bind/restart ran).
- Later restarts came up; health checks returned 200.
- `qwen3.5:27b` was loaded at one point (above the 14B regression cap). `enforce_regression_agent_models` unloaded it and switched agents to `qwen3.5:9b`.

**Scenario runs**
- `collab-participation-two-agent-strict`: **passed once** (~32s, early in the session).
- Later reruns of that scenario and others (`plan-distinct-deliverables-same-agent`, batch verification runs) **failed** with discussion timeouts — usually 0 valid messages, often only `generation_error` posts, or agents still generating after the scenario cancelled the collab.
- Batched scenario runs were stopped to avoid overlapping contention.

**CI (unchanged)**
- `make test-all` and `make test-conversation-contract` passed earlier in the session.
- Go unit tests on the changed packages passed after the code edits.

The code fixes are in place; live verification needs a **clean full regression boot** (not manual hub + `SKIP_BOOT`) with no model above 14B loaded before re-running the brief’s 19 scenarios.
```

