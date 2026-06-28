# Model benchmark — quick

**Run:** 2026-06-19 06:39 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on ≤14B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `qwen2.5-coder:14b` | 4/5 | 80% | 2/3 | 2/2 | 3m26s |  |
| 2 | `deepseek-coder:6.7b` | 2/5 | 40% | 2/3 | 0/2 | 10m31s |  |
| 3 | `qwen3.5:9b` | 1/5 | 20% | 0/3 | 1/2 | 18m55s |  |

## Per-scenario matrix

| Scenario | qwen3.5:9b | deepseek-coder:6.7b | qwen2.5-coder:14b |
|---|---|---|---|
| implement/go-handler | ✗ 5m25s | ✓ 3m37s | ✓ 43s |
| implement/theme-toggle | ✗ 5m39s | ✗ 57s | ✗ 27s |
| implement/ask-mode-no-write | ✗ 5m03s | ✓ 55s | ✓ 49s |
| chat/dm-backend-workspace | ✓ 45s | ✗ 3m01s | ✓ 47s |
| chat/dm-backend-echo-followup | ✗ 2m00s | ✗ 2m00s | ✓ 39s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
