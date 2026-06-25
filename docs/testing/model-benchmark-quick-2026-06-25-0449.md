# Model benchmark — quick

**Run:** 2026-06-25 04:49 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 5 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `qwen2.5-coder:14b` | 6/7 | 86% | 5/5 | 1/2 | 6m54s |  |
| 2 | `devstral:24b` | 6/7 | 86% | 5/5 | 1/2 | 7m51s |  |
| 3 | `qwen3.5:9b` | 6/7 | 86% | 5/5 | 1/2 | 7m52s |  |
| 4 | `gemma3:12b` | 6/7 | 86% | 5/5 | 1/2 | 8m46s |  |
| 5 | `deepseek-coder:6.7b` | 6/7 | 86% | 5/5 | 1/2 | 9m48s |  |
| 6 | `codegemma:7b` | 5/7 | 71% | 5/5 | 0/2 | 8m25s |  |
| 7 | `codestral:22b` | 2/3 | 67% | 2/3 | — | 0s |  |

## Per-scenario matrix

| Scenario | deepseek-coder:6.7b | codegemma:7b | qwen3.5:9b | gemma3:12b | qwen2.5-coder:14b | codestral:22b | devstral:24b |
|---|---|---|---|---|---|---|---|
| implement/go-handler | ✓ 1m47s | ✓ 38s | ✓ 49s | ✓ 42s | ✓ 56s | ✓ 40s | ✓ 44s |
| implement/theme-toggle | ✓ 1m55s | ✓ 54s | ✓ 48s | ✓ 50s | ✓ 56s | ✓ 50s | ✓ 48s |
| implement/ask-mode-no-write | ✓ 2m45s | ✓ 2m49s | ✓ 1m41s | ✓ 1m27s | ✓ 1m07s | ✗ 5m04s | ✓ 1m37s |
| implement/go-test-failure-repair | ✓ 5s | ✓ 5s | ✓ 5s | ✓ 5s | ✓ 5s | — | ✓ 5s |
| implement/typescript-compile-error-fix | ✓ 5s | ✓ 5s | ✓ 5s | ✓ 5s | ✓ 5s | — | ✓ 5s |
| chat/dm-backend-workspace | ✓ 1m08s | ✗ 0s | ✓ 56s | ✓ 1m39s | ✓ 1m44s | — | ✓ 2m31s |
| chat/dm-backend-echo-followup | ✗ 2m00s | ✗ 3m54s | ✗ 3m27s | ✗ 3m57s | ✗ 2m00s | — | ✗ 2m00s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini/qwen2.5-coder:14b`
