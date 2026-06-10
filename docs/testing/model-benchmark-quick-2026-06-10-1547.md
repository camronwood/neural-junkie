# Model benchmark — quick

**Run:** 2026-06-10 15:47 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on 14B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 4/5 | 80% | 2/3 | 2/2 | 9m28s |  |
| 2 | `codegemma:7b` | 4/5 | 80% | 2/3 | 2/2 | 10m26s |  |

## Per-scenario matrix

| Scenario | codegemma:7b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 1m49s | ✓ 41s |
| implement/theme-toggle | ✗ 7m06s | ✗ 7m04s |
| implement/ask-mode-no-write | ✓ 17s | ✓ 25s |
| chat/dm-backend-workspace | ✓ 40s | ✓ 45s |
| chat/dm-backend-echo-followup | ✓ 33s | ✓ 33s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
