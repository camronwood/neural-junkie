# Model benchmark — quick

**Run:** 2026-06-29 16:44 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 5 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `codegemma:7b` | 4/7 | 57% | 3/5 | 1/2 | 41m42s |  |
| 2 | `qwen2.5-coder:14b` | 3/7 | 43% | 2/5 | 1/2 | 25m18s |  |
| 3 | `devstral:24b` | 3/7 | 43% | 2/5 | 1/2 | 29m55s |  |
| 4 | `codestral:22b` | 3/7 | 43% | 2/5 | 1/2 | 37m03s |  |
| 5 | `deepseek-coder:6.7b` | 2/7 | 29% | 2/5 | 0/2 | 56m01s |  |
| 6 | `qwen3.5:9b` | 2/7 | 29% | 1/5 | 1/2 | 58m51s |  |
| 7 | `gemma3:12b` | 0/7 | 0% | 0/5 | 0/2 | 24m43s |  |

## Per-scenario matrix

| Scenario | deepseek-coder:6.7b | codegemma:7b | qwen3.5:9b | gemma3:12b | qwen2.5-coder:14b | codestral:22b | devstral:24b |
|---|---|---|---|---|---|---|---|
| implement/go-handler | ✓ 1m41s | ✓ 51s | ✓ 16m47s | ✗ 1m05s | ✓ 1m13s | ✓ 6m08s | ✓ 2m21s |
| implement/theme-toggle | ✗ 20m16s | ✗ 20m15s | ✗ 10m03s | ✗ 10m05s | ✗ 10m03s | ✗ 10m05s | ✗ 10m03s |
| implement/ask-mode-no-write | ✓ 1m21s | ✓ 1m11s | ✗ 5m03s | ✗ 2m54s | ✗ 3m31s | ✓ 2m41s | ✓ 2m17s |
| implement/go-test-failure-repair | ✗ 13m13s | ✓ 5s | ✗ 15m04s | ✗ 1m21s | ✓ 2m22s | ✗ 6m06s | ✗ 4m39s |
| implement/typescript-compile-error-fix | ✗ 14m18s | ✗ 14m17s | ✗ 7m04s | ✗ 7m05s | ✗ 7m04s | ✗ 7m04s | ✗ 7m04s |
| chat/dm-backend-workspace | ✗ 3m00s | ✗ 3m01s | ✓ 2m37s | ✗ 0s | ✗ 0s | ✓ 2m46s | ✓ 1m17s |
| chat/dm-backend-echo-followup | ✗ 2m00s | ✓ 1m50s | ✗ 2m00s | ✗ 2m01s | ✓ 52s | ✗ 2m00s | ✗ 2m00s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini`
