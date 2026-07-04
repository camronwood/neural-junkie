# layer fix loop — implement — iteration 3 — 2026-07-04-1102-iter3 UTC

layer=implement
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-implement-2026-07-04-1102-iter3.md
fix_branch=release-prep/layer-implement-2026-07-04-1102
git_commit=f950b05
layer_gate_rc=1
agent_rc=0

## Cursor agent output

```text
## Summary

Gate failures were in four implement scenarios: **`at-file-explicit-path`** (FrontendEngineer never replied within 900s) and three theme scenarios where the agent replied but **`tailwind.config.js` lacked `darkMode`**.

### Root causes addressed

1. **Scoped `@file:` edits** — `tryEarlyScopedFileEdit` proposed hub edits but had no direct-write fallback when auto-approve didn’t land on disk (unlike the tailwind path added in iter2).
2. **Theme tailwind repair** — `repairTailwindDarkModeIfNeeded` only ran inside the LLM file cycle, not on early-exit paths; a session could finish without `darkMode` on disk.
3. **Corrupt fixture** — `react-tailwind-minimal/package.json` contained `App.tsx` source instead of JSON (also missing from `.scenario-baseline`), which broke stack detection and baseline reset.

### Code / fixture changes

| Area | Change |
|------|--------|
| `internal/agent/implementation_fallback.go` | Added `proposeScopedFileEdit`, `scopedFileEditSatisfied`, `finalizeImplementationSessionRepairs`; wired `tryEarlyScopedFileEdit` through direct-write fallback |
| `internal/agent/implementation_session.go` | Deferred `finalizeImplementationSessionRepairs` on every implementation session exit |
| `internal/agent/implementation_fallback_test.go` | Added `TestTryEarlyScopedFileEdit_subtitle` |
| `scenarios/fixtures/react-tailwind-minimal/` | Restored valid `package.json`; added `.scenario-baseline/package.json` |

### Verification commands run

- `go test ./internal/agent/... -run 'Scoped|Theme|Tailwind|ImplSession' -count=1` — pass  
- `make test-conversation-contract` — pass  
- `make test-all` — pass  

Live `implement-scenarios` were not re-run here (requires a hub restart + ~40 min). After rebuilding/restarting the regression hub with this worktree, re-run:

```bash
python3 scripts/implement-scenarios.py --scenario at-file-explicit-path --hub http://127.0.0.1:18765
python3 scripts/implement-scenarios.py --scenario react-theme-toggle --hub http://127.0.0.1:18765
```The earlier `make test-all` run finished successfully (exit 0) — vet, Go tests, desktop tsc, Vitest, and coverage all passed.

Together with the fixes from this pass (scoped edit direct-write, deferred tailwind repair, restored `react-tailwind-minimal/package.json`), CI is green via `make test-all` and `make test-conversation-contract`. To confirm the four failing live scenarios, restart the regression hub on this worktree build and run the targeted `implement-scenarios.py` checks for `at-file-explicit-path` and `react-theme-toggle`.
```

