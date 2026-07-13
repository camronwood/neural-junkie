# layer fix loop — implement — iteration 3 — 2026-07-03-1109-iter3 UTC

layer=implement
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-implement-2026-07-03-1109-iter3.md
fix_branch=release-prep/layer-implement-2026-07-03-1109
git_commit=a426650
layer_gate_rc=1
agent_rc=0

## Cursor agent output

```text
## Summary

Live failures were caused by **FrontendEngineer blocking on `implement-scenarios`**: an implementation session started at hub boot (17:42) never finished, so the agent never replied to later frontend scenarios. BackendEngineer uses a separate listener and kept passing.

### Root causes
1. **Agent Runtime v2** on `implement-scenarios` allowed up to 100 tool-loop iterations with no outer generation deadline.
2. **LLM routing (`Plan`)** ran before deterministic early fixes (playbooks, theme.css shortcuts).
3. **`missing_start_all_target` playbook** could fail grounding validation and fall through to a long LLM session.
4. **`/api/channels/join`** only updated membership; agents could wait on `discoverChannels` before subscribing.

### Code changes
| Area | Change |
|------|--------|
| `internal/agent/agent_runtime_config.go` | Use legacy impl caps (20 tool iters / 3 edit rounds / 3 files) on `implement-scenarios`, even with runtime v2 |
| `internal/agent/collab_execution_timeout.go` | Apply 540s generation deadline on `implement-scenarios` |
| `internal/agent/implementation_session.go` | Run deterministic early fixes **before** `globalImplementationRouting.Plan()` |
| `internal/agent/implementation_playbooks.go` | Skip grounding gate for pasted `missing_start_all_target` evidence; validate Makefile directly |
| `cmd/server/channels_handlers.go` + `internal/hub/hub_agent_registry.go` | Eagerly subscribe agents on channel join via `EnsureAgentSubscribedToChannel` |

### Verification commands run
- `go test ./internal/agent/... ./cmd/server/... -run 'ImplSession|CollabGeneration|Playbook|Theme|MissingStart'` — pass
- `make test-all` — pass
- `make test-conversation-contract` — pass
- `SKIP_BOOT=1 python3 scripts/implement-scenarios.py --scenario theme-toggle --hub http://127.0.0.1:18765` — **PASS** (after hub recovery restart)

CI was already green before these edits; the live scenario pass confirms FrontendEngineer responds again with the new behavior. For a full implement layer gate, restart the regression hub so it loads this worktree build (the previously running hub was on older code with FE stuck since ~17:42).`make test-all` completed successfully — CI is green.

The other background runs were live scenario probes against the regression hub:

- **First `theme-toggle` attempts failed** — FrontendEngineer timed out on the old hub, which had been stuck in an implementation session since ~17:42 and never picked up new messages.
- **Channel-join tests** confirmed agents were on `implement-scenarios` but FE still didn’t reply while that session was hung.
- **After the code fixes and a hub recovery restart** (`theme-toggle` on the worktree build), the scenario **passed** — FrontendEngineer replied and deliverable assertions cleared.

No further code changes from those runs. To re-run the full implement layer, restart the regression hub so it loads this worktree build, then run `make layer-gate LAYER=implement` or `python3 scripts/implement-scenarios.py --all`.
```

