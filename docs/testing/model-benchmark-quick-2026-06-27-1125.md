# Model benchmark — quick

**Run:** 2026-06-27 11:25 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 5 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `deepseek-coder:6.7b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 2 | `codegemma:7b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 3 | `qwen3.5:9b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 4 | `gemma3:12b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 5 | `qwen2.5-coder:14b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 6 | `codestral:22b` | 0/1 | 0% | 0/1 | — | 0s |  |
| 7 | `devstral:24b` | 0/1 | 0% | 0/1 | — | 0s |  |

## Per-scenario matrix

| Scenario | deepseek-coder:6.7b | codegemma:7b | qwen3.5:9b | gemma3:12b | qwen2.5-coder:14b | codestral:22b | devstral:24b |
|---|---|---|---|---|---|---|---|
| implement/go-handler | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s |
| implement/theme-toggle | — | — | — | — | — | — | — |
| implement/ask-mode-no-write | — | — | — | — | — | — | — |
| implement/go-test-failure-repair | — | — | — | — | — | — | — |
| implement/typescript-compile-error-fix | — | — | — | — | — | — | — |
| chat/dm-backend-workspace | — | — | — | — | — | — | — |
| chat/dm-backend-echo-followup | — | — | — | — | — | — | — |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini/qwen2.5-coder:14b`
