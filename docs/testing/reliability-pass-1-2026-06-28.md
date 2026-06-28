# Agent Runtime Reliability — Pass 1 Report (2026-06-28)

## Scores vs targets

| Metric | Target | Result |
|--------|--------|--------|
| `go test ./internal/agent/...` | Green | **PASS** |
| `go test ./test/...` | Green | **PASS** |
| `make implement-scenarios` | ≥15/20 | **Not run** (requires live hub + Ollama agents) |
| Fixture mismatches | 0 | **Fixed** (`go-build-error-fix` aligned to Multiply bug) |

## Pass 1 RUN — shipped

- Wired `ShouldBlockRunCommand` + `RecordCommandRun` into LLM `run_command` path (`executeAgentTool`)
- Boot-fix grounding: `RecordReadPath` on successful `read_file` tool calls
- Completed `attemptTauriVitePortPlaybook` (align Vite port to Tauri `devPath`)
- Harness: `--all` preflight verifies required agents online before sweep
- Scenario outcome assertions (required): `go-build-error-fix`, `verify-failure-one-repair`, `tauri-make-start-all-missing`, `app-wont-boot-fix-like`, `deny-destructive-command`
- `test/` package compiles and passes

## Top failure categories (pre-fix baseline)

From `test-everything-2026-06-27-2204.md`:

1. **Timeout** — agents not responding on `implement-scenarios` channel (~95% of failures)
2. **Fixture mismatch** — `go-build-error-fix` expected `Add`, fixture had `Multiply` bug
3. **Policy not wired** — circuit breaker only on bootstrap/repro, not tool-loop `run_command`

## Diagnostic answer

> **What is the biggest area we still need to improve NJ in?**

**Agent runtime reliability** — specifically closing the loop from Agent mode send → grounded edit → verify/repair → visible outcome. Pass 1 fixed policy wiring and harness truthfulness; live scenario pass rate still depends on hub health and local model behavior.

## Pass 2 scope delta

- Reliable-tier routing (local heavy + opt-in cloud repair 2+)
- Outcome metadata on redirect/timeout paths
- Re-run `make implement-scenarios` after hub restart to measure ≥18/20
