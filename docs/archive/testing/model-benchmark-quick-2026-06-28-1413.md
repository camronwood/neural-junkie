# Model benchmark — quick

**Run:** 2026-06-28 14:13 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 5 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 5/7 | 71% | 4/5 | 1/2 | 13m34s |  |
| 2 | `qwen2.5-coder:14b` | 5/7 | 71% | 4/5 | 1/2 | 14m33s |  |
| 3 | `qwen3.5:9b` | 4/7 | 57% | 2/5 | 2/2 | 23m27s |  |
| 4 | `codegemma:7b` | 4/7 | 57% | 2/5 | 2/2 | 24m43s |  |
| 5 | `devstral:24b` | 3/7 | 43% | 2/5 | 1/2 | 21m50s |  |
| 6 | `codestral:22b` | 3/7 | 43% | 2/5 | 1/2 | 31m09s |  |
| 7 | `deepseek-coder:6.7b` | 1/7 | 14% | 0/5 | 1/2 | 41m28s |  |

## Per-scenario matrix

| Scenario | deepseek-coder:6.7b | codegemma:7b | qwen3.5:9b | gemma3:12b | qwen2.5-coder:14b | codestral:22b | devstral:24b |
|---|---|---|---|---|---|---|---|
| implement/go-handler | ✗ 7m15s | ✗ 7m07s | ✗ 7m04s | ✓ 1m55s | ✓ 6m48s | ✓ 6m42s | ✗ 9m06s |
| implement/theme-toggle | ✗ 10m06s | ✗ 10m22s | ✗ 10m06s | ✓ 5m32s | ✓ 1m55s | ✗ 4m39s | ✓ 1m39s |
| implement/ask-mode-no-write | ✗ 5m03s | ✓ 2m12s | ✓ 1m27s | ✓ 2m01s | ✓ 1m29s | ✓ 25s | ✗ 5m05s |
| implement/go-test-failure-repair | ✗ 15m04s | ✓ 7s | ✓ 5s | ✓ 5s | ✓ 5s | ✗ 15m03s | ✓ 7s |
| implement/typescript-compile-error-fix | ✗ 8s | ✗ 8s | ✗ 8s | ✗ 6s | ✗ 6s | ✗ 8s | ✗ 8s |
| chat/dm-backend-workspace | ✓ 1m38s | ✓ 1m54s | ✓ 1m48s | ✓ 1m43s | ✓ 1m58s | ✓ 1m59s | ✓ 1m57s |
| chat/dm-backend-echo-followup | ✗ 2m00s | ✓ 2m22s | ✓ 2m37s | ✗ 2m00s | ✗ 2m00s | ✗ 2m00s | ✗ 3m36s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini/qwen2.5-coder:14b`
