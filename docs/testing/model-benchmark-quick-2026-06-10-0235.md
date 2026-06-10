# Model benchmark — quick

**Run:** 2026-06-10 02:35 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on 14B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `qwen2.5-coder:14b` | 5/5 | 100% | 3/3 | 2/2 | 3m19s | winner |
| 2 | `devstral:24b` | 4/5 | 80% | 3/3 | 1/2 | 6m38s |  |
| 3 | `qwen3.5:9b` | 4/5 | 80% | 3/3 | 1/2 | 6m52s |  |
| 4 | `codestral:22b` | 3/5 | 60% | 2/3 | 1/2 | 6m44s |  |
| 5 | `deepseek-coder:6.7b` | 3/5 | 60% | 2/3 | 1/2 | 7m21s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | codestral:22b | devstral:24b | qwen3.5:9b | deepseek-coder:6.7b |
|---|---|---|---|---|---|
| implement/go-handler | ✓ 25s | ✓ 49s | ✓ 49s | ✓ 23s | ✓ 45s |
| implement/theme-toggle | ✓ 33s | ✓ 1m13s | ✓ 1m25s | ✓ 1m15s | ✓ 1m47s |
| implement/ask-mode-no-write | ✓ 39s | ✗ 1m15s | ✓ 1m17s | ✓ 1m15s | ✗ 1m13s |
| chat/dm-backend-workspace | ✓ 52s | ✓ 1m26s | ✗ 2m00s | ✓ 1m10s | ✓ 1m35s |
| chat/dm-backend-echo-followup | ✓ 49s | ✗ 2m00s | ✓ 1m05s | ✗ 2m48s | ✗ 2m00s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
