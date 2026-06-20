# Model benchmark — quick

**Run:** 2026-06-20 17:48 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `qwen2.5-coder:14b` | 5/5 | 100% | 3/3 | 2/2 | 3m06s | winner |
| 2 | `gemma3:12b` | 5/5 | 100% | 3/3 | 2/2 | 3m35s |  |
| 3 | `qwen3.5:9b` | 1/2 | 50% | 1/2 | — | 0s |  |
| 4 | `devstral:24b` | 1/2 | 50% | 1/2 | — | 0s |  |
| 5 | `deepseek-coder:6.7b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 6 | `codegemma:7b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 7 | `codestral:22b` | 0/1 | 0% | 0/1 | — | 0s |  |

## Per-scenario matrix

| Scenario | deepseek-coder:6.7b | codegemma:7b | qwen3.5:9b | gemma3:12b | qwen2.5-coder:14b | codestral:22b | devstral:24b |
|---|---|---|---|---|---|---|---|
| implement/go-handler | ✗ 31s | ✗ 33s | ✓ 35s | ✓ 35s | ✓ 35s | ✗ 39s | ✓ 39s |
| implement/theme-toggle | — | — | ✗ 35s | ✓ 37s | ✓ 37s | — | ✗ 31s |
| implement/ask-mode-no-write | — | — | — | ✓ 29s | ✓ 35s | — | — |
| chat/dm-backend-workspace | — | — | — | ✓ 45s | ✓ 34s | — | — |
| chat/dm-backend-echo-followup | — | — | — | ✓ 1m08s | ✓ 44s | — | — |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
