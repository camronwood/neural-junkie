# layer fix loop — implement — iteration 2 — 2026-07-03-1109-iter2 UTC

layer=implement
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-implement-2026-07-03-1109-iter2.md
fix_branch=release-prep/layer-implement-2026-07-03-1109
git_commit=ba11fd4
layer_gate_rc=1
agent_rc=0

## Cursor agent output

```text
## Summary

Addressed CI failures and the main implement-scenario timeout pattern from the iter2 gate log (13/20 failures were `timeout waiting for FrontendEngineer`, often because the first boot-fix scenario never finished and blocked later frontend work).

### Root causes

1. **CI** — `TestAugmentPromptWithCLIImages_AppendsPaths` failed when `CURSOR_WORK_DIR=/path/to/your/project` (placeholder, not writable) overrode the provider temp dir. `desktop/node_modules` was also missing for `tsc`/Vitest.
2. **Boot-fix scenarios** — After the early `missing_start_all_target` playbook, `runReproVerify` re-ran `make start-all`, which starts dev servers and could block until session timeout.
3. **Session vs scenario timing** — Implement-scenarios used 480s/600s session caps while several scenarios only wait 420s, so the agent could still be running when `wait_reply` timed out.
4. **`runReproVerify`** — Unlike `runReproCommand`, it had no command timeout (dev-server commands could hang).

### Code changes

| File | Change |
|------|--------|
| `internal/agent/cli_image_bridge.go` | `cliWorkDirUsable()` — ignore unusable `*_WORK_DIR` env paths |
| `internal/agent/cli_image_bridge_test.go` | Test sets placeholder env to verify fallback |
| `internal/agent/cli_workdir_test.go` | Env override test uses a real temp dir |
| `internal/agent/implementation_repro.go` | Add boot-fix/dev-server timeouts to `runReproVerify` |
| `internal/agent/implementation_session.go` | Skip repro verify after `missing_start_all_target` playbook; treat playbook proposals as proposed |
| `internal/agent/agent_runtime_config.go` | Implement-scenarios caps: **360s** backend / **540s** frontend |
| `internal/agent/agent_runtime_config_test.go` | Updated timeout expectations |

### Verification commands run

```bash
cd desktop && npm ci
go test ./internal/agent/ -count=1 -run 'TestAugmentPromptWithCLIImages|TestImplSessionTimeout|TryEarly|Playbook'
make test-all          # exit 0
make test-conversation-contract  # exit 0
```

Both CI targets pass. Live `make layer-gate LAYER=implement` still needs a hub restart so agents pick up these changes before expecting 20/20 implement scenarios.The release-prep fix loop is done. CI is green again after:

- Ignoring unusable CLI work-dir env overrides (fixes the `TestAugmentPromptWithCLIImages_AppendsPaths` failure)
- Skipping long `make start-all` repro verify after the boot-fix playbook
- Tightening implement-scenario session caps (360s/540s) so agents reply before 420s `wait_reply` windows
- Installing `desktop/node_modules` for `tsc`/Vitest

`make test-all` and `make test-conversation-contract` both pass. To validate the full implement layer (20/20), restart the hub and run `make layer-gate LAYER=implement`.
```

