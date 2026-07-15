# Model benchmark — standard

**Run:** 2026-07-15 14:48 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Balanced gate — quick implement + regression chat tag filter.

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 32/34 | 94% | 12/12 | 20/20 | 20m21s |  |
| 2 | `codegemma:7b` | 31/34 | 91% | 12/12 | 19/20 | 10m13s |  |
| 3 | `deepseek-coder:6.7b` | 31/34 | 91% | 12/12 | 19/20 | 11m03s |  |
| 4 | `qwen3.5:9b` | 31/34 | 91% | 12/12 | 19/20 | 13m30s |  |
| 5 | `qwen2.5-coder:14b` | 31/34 | 91% | 11/12 | 20/20 | 27m27s |  |
| 6 | `nj-ornith:9b` | 30/34 | 88% | 11/12 | 19/20 | 26m09s |  |
| 7 | `nj-bonsai:27b` | 24/34 | 71% | 11/12 | 13/20 | 16m56s |  |
| 8 | `nj-ternary-bonsai:27b` | 23/34 | 68% | 10/12 | 13/20 | 30m04s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | qwen3.5:9b | nj-ornith:9b | nj-bonsai:27b | nj-ternary-bonsai:27b | deepseek-coder:6.7b | codegemma:7b | gemma3:12b |
|---|---|---|---|---|---|---|---|---|
| implement/go-handler | ✓ 20s | ✓ 15s | ✓ 14s | ✓ 18s | ✓ 10s | ✓ 10s | ✓ 21s | ✓ 19s |
| implement/theme-toggle | ✓ 11s | ✓ 9s | ✓ 9s | ✓ 9s | ✓ 9s | ✓ 9s | ✓ 9s | ✓ 9s |
| implement/react-theme-toggle | ✓ 13s | ✓ 13s | ✓ 13s | ✓ 13s | ✓ 13s | ✓ 13s | ✓ 13s | ✓ 13s |
| implement/ask-mode-no-write | ✓ 33s | ✓ 23s | ✓ 35s | ✗ 10m13s | ✗ 10m13s | ✓ 29s | ✓ 21s | ✓ 29s |
| implement/go-test-failure-repair | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s |
| implement/typescript-compile-error-fix | ✓ 8s | ✓ 8s | ✓ 8s | ✓ 8s | ✓ 8s | ✓ 8s | ✓ 8s | ✓ 8s |
| implement/rules-constrained-implement | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s |
| implement/selection-scoped-edit | ✗ 14m14s | ✓ 43s | ✗ 14m14s | ✓ 45s | ✗ 14m14s | ✓ 47s | ✓ 49s | ✓ 7m52s |
| implement/at-file-explicit-path | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s |
| implement/verify-failure-one-repair | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s |
| implement/deny-destructive-command | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s | ✓ 7s |
| implement/plan-mode-no-write | ✓ 35s | ✓ 19s | ✓ 41s | ✓ 23s | ✓ 21s | ✓ 21s | ✓ 19s | ✓ 27s |
| chat/already-said-closure | ✓ 19s | ✓ 15s | ✓ 15s | ✓ 14s | ✓ 15s | ✓ 15s | ✓ 15s | ✓ 17s |
| chat/cross-repo-ambient-scope | ✓ 12s | ✓ 10s | ✗ 6s | ✗ 3s | ✗ 3s | ✓ 14s | ✓ 11s | ✓ 13s |
| chat/dm-architect-workspace | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s |
| chat/dm-assistant-continue-after-closure | ✓ 30s | ✓ 34s | ✓ 33s | ✓ 35s | ✓ 32s | ✓ 32s | ✓ 32s | ✓ 36s |
| chat/dm-backend-codebase-semantic | ✓ 24s | ✓ 35s | ✓ 49s | ✗ 4s | ✗ 4s | ✗ 21s | ✗ 28s | ✓ 34s |
| chat/dm-backend-deep-continuation | ✓ 43s | ✗ 1m05s | ✓ 40s | ✗ 3s | ✗ 3s | ✓ 41s | ✓ 41s | ✓ 43s |
| chat/dm-backend-echo-followup | ✓ 43s | ✓ 46s | ✓ 59s | ✗ 3s | ✗ 3s | ✓ 35s | ✓ 34s | ✓ 38s |
| chat/dm-backend-interject-resume | ✓ 31s | ✓ 31s | ✓ 32s | ✓ 33s | ✓ 34s | ✓ 36s | ✓ 34s | ✓ 35s |
| chat/dm-backend-workspace | ✓ 35s | ✓ 34s | ✓ 32s | ✗ 4s | ✗ 3s | ✓ 24s | ✓ 16s | ✓ 27s |
| chat/dm-code-reviewer-workspace | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s |
| chat/dm-database-workspace | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 4s | ✓ 3s | ✓ 3s | ✓ 3s |
| chat/dm-frontend-workspace | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s |
| chat/dm-ide-route-backend | ✓ 12s | ✓ 13s | ✓ 13s | ✓ 14s | ✓ 14s | ✓ 14s | ✓ 14s | ✓ 15s |
| chat/dm-platform-workspace | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s |
| chat/dm-safe-readonly-command | ✓ 27s | ✓ 24s | ✓ 14s | ✓ 17s | ✓ 18s | ✓ 17s | ✓ 15s | ✓ 19s |
| chat/dm-security-workspace | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 4s | ✓ 3s | ✓ 3s | ✓ 3s | ✓ 3s |
| chat/dm-topic-switch | ✓ 1m20s | ✓ 1m12s | ✓ 1m06s | ✓ 50s | ✓ 42s | ✓ 1m04s | ✓ 56s | ✓ 1m31s |
| chat/public-backend-theme-workspace | ✓ 53s | ✓ 46s | ✓ 35s | ✗ 4s | ✗ 3s | ✓ 29s | ✓ 20s | ✓ 41s |
| chat/public-frontend-theme-continuation | ✓ 2m27s | ✓ 2m15s | ✓ 1m37s | ✗ 4s | ✗ 3s | ✓ 1m18s | ✓ 1m02s | ✓ 1m40s |
| chat/thanks-closure | ✓ 33s | ✓ 37s | ✓ 21s | ✓ 16s | ✓ 17s | ✓ 23s | ✓ 12s | ✓ 50s |
| arena/logic-set | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s |
| external/humaneval-25 | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s | ✗ 0s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, react-theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix, rules-constrained-implement, selection-scoped-edit, at-file-explicit-path, verify-failure-one-repair, deny-destructive-command, plan-mode-no-write
- **Chat:** already-said-closure, cross-repo-ambient-scope, dm-architect-workspace, dm-assistant-continue-after-closure, dm-backend-codebase-semantic, dm-backend-deep-continuation, dm-backend-echo-followup, dm-backend-interject-resume, dm-backend-workspace, dm-code-reviewer-workspace, dm-database-workspace, dm-frontend-workspace, dm-ide-route-backend, dm-platform-workspace, dm-safe-readonly-command, dm-security-workspace, dm-topic-switch, public-backend-theme-workspace, public-frontend-theme-continuation, thanks-closure
- **Collab:** (none)
- **Arena:** logic-set
- **CAD:** (none)
- **External:** humaneval-25

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `ollama/qwen2.5-coder:14b`
