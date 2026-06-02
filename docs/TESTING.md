# Testing

## Test isolation

Go tests that touch hub storage, repo agents, or collaboration sandboxes should not write to your real `~/.neural-junkie` tree.

- **`internal/testutil/isolate.go`** — `IsolateNeuralJunkieHome(t)` sets `HOME`, `USERPROFILE`, `NEURAL_JUNKIE_REPO_DIR`, and `NEURAL_JUNKIE_COLLAB_ASSETS_DIR` under a temp directory for the duration of a test.
- **`test/test_main.go`**, **`internal/hub/test_main.go`**, and **`cmd/server/test_main.go`** — package-level `TestMain` hooks apply the same temp-home pattern before any test in those packages runs.
- **Hub collab tests** — use `newTestHub(t)` (in `internal/hub/collab_test_helpers_test.go`) when a test creates collaborations and runs `approveAndExecuteCollabForTest`; it combines home isolation with `SetCollaborationAssetsRootResolver`.

Integration tests in the top-level `test/` package call `useIsolatedRepoStorage(t)`, which delegates to `testutil.IsolateNeuralJunkieHome`.

## Scenario runners and `KEEP=1`

Live scenario scripts against a running hub clean up after themselves by default:

| Script | Default cleanup | Keep artifacts |
|--------|-----------------|----------------|
| `scripts/collab-scenarios.py` | Cancels collaborations, removes `collabs/<id>/` from the scenario workspace | `--keep` |
| `scripts/learning-scenarios.py` | Deletes learnings created during the run, restores hub settings | `--keep` or `KEEP=1` |

Use `--keep` (or `KEEP=1` for learning scenarios) when you want to inspect hub state, workspace files, or learnings after a failed or successful run.
