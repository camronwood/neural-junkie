# Model benchmark — release

**Run:** 2026-06-21 03:01 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Release gate — benchmark winners only (~10–20 min).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 5/5 | 100% | 3/3 | 2/2 | 3m20s | winner |
| 2 | `qwen2.5-coder:14b` | 5/5 | 100% | 3/3 | 2/2 | 4m02s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 37s | ✓ 36s |
| implement/theme-toggle | ✓ 39s | ✓ 43s |
| implement/ask-mode-no-write | ✓ 39s | ✓ 39s |
| chat/dm-backend-workspace | ✓ 1m01s | ✓ 37s |
| chat/dm-backend-echo-followup | ✓ 1m04s | ✓ 45s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
