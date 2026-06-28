# Model benchmark — quick

**Run:** 2026-06-24 19:17 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 5 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `deepseek-coder:6.7b` | 4/5 | 80% | 4/5 | — | 0s |  |
| 2 | `codegemma:7b` | 4/5 | 80% | 4/5 | — | 0s |  |
| 3 | `qwen3.5:9b` | 4/5 | 80% | 4/5 | — | 0s |  |
| 4 | `codestral:22b` | 4/5 | 80% | 4/5 | — | 0s |  |
| 5 | `devstral:24b` | 4/5 | 80% | 4/5 | — | 0s |  |
| 6 | `gemma3:12b` | 5/7 | 71% | 5/5 | 0/2 | 2m07s |  |
| 7 | `qwen2.5-coder:14b` | 0/1 | 0% | 0/1 | — | 0s |  |

## Per-scenario matrix

| Scenario | deepseek-coder:6.7b | codegemma:7b | qwen3.5:9b | gemma3:12b | qwen2.5-coder:14b | codestral:22b | devstral:24b |
|---|---|---|---|---|---|---|---|
| implement/go-handler | ✓ 1m13s | ✓ 33s | ✓ 33s | ✓ 27s | ✗ 23s | ✓ 33s | ✓ 31s |
| implement/theme-toggle | ✓ 33s | ✓ 35s | ✓ 31s | ✓ 31s | — | ✓ 31s | ✓ 33s |
| implement/ask-mode-no-write | ✓ 29s | ✓ 25s | ✓ 25s | ✓ 33s | — | ✓ 29s | ✓ 31s |
| implement/go-test-failure-repair | ✓ 5s | ✓ 5s | ✓ 5s | ✓ 5s | — | ✓ 5s | ✓ 5s |
| implement/typescript-compile-error-fix | ✗ 3m45s | ✗ 27s | ✗ 31s | ✓ 31s | — | ✗ 3m33s | ✗ 31s |
| chat/dm-backend-workspace | — | — | — | ✗ 0s | — | — | — |
| chat/dm-backend-echo-followup | — | — | — | ✗ 0s | — | — | — |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini/qwen2.5-coder:14b`
