# Model benchmark — quick

**Run:** 2026-06-29 06:15 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 5 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `qwen3.5:9b` | 4/7 | 57% | 2/5 | 2/2 | 34m25s |  |
| 2 | `gemma3:12b` | 3/7 | 43% | 2/5 | 1/2 | 25m18s |  |
| 3 | `deepseek-coder:6.7b` | 3/7 | 43% | 2/5 | 1/2 | 29m15s |  |
| 4 | `codegemma:7b` | 2/7 | 29% | 2/5 | 0/2 | 25m30s |  |
| 5 | `devstral:24b` | 2/7 | 29% | 2/5 | 0/2 | 27m45s |  |
| 6 | `codestral:22b` | 2/7 | 29% | 2/5 | 0/2 | 28m59s |  |
| 7 | `qwen2.5-coder:14b` | 2/7 | 29% | 1/5 | 1/2 | 29m35s |  |

## Per-scenario matrix

| Scenario | deepseek-coder:6.7b | codegemma:7b | qwen3.5:9b | gemma3:12b | qwen2.5-coder:14b | codestral:22b | devstral:24b |
|---|---|---|---|---|---|---|---|
| implement/go-handler | ✓ 3m13s | ✓ 1m53s | ✓ 5m01s | ✓ 1m29s | ✗ 9m04s | ✓ 2m45s | ✓ 2m41s |
| implement/theme-toggle | ✗ 10m04s | ✗ 10m04s | ✗ 10m04s | ✗ 10m04s | ✗ 10m04s | ✗ 10m04s | ✗ 10m04s |
| implement/ask-mode-no-write | ✓ 2m15s | ✓ 15s | ✓ 2m13s | ✓ 1m21s | ✓ 13s | ✓ 1m49s | ✓ 27s |
| implement/go-test-failure-repair | ✗ 1m45s | ✗ 1m01s | ✗ 4m55s | ✗ 1m13s | ✗ 2m05s | ✗ 2m03s | ✗ 2m15s |
| implement/typescript-compile-error-fix | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s |
| chat/dm-backend-workspace | ✓ 2m40s | ✗ 3m01s | ✓ 2m35s | ✓ 1m54s | ✗ 0s | ✗ 3m01s | ✗ 3m00s |
| chat/dm-backend-echo-followup | ✗ 2m00s | ✗ 2m00s | ✓ 2m19s | ✗ 2m00s | ✓ 53s | ✗ 2m00s | ✗ 2m00s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini/qwen2.5-coder:14b`
