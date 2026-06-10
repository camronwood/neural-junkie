# Model benchmark — quick

**Run:** 2026-06-09 23:44 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on 14B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `devstral:24b` | 5/5 | 100% | 3/3 | 2/2 | 7m39s | winner |
| 2 | `codestral:22b` | 3/5 | 60% | 2/3 | 1/2 | 8m16s |  |

## Per-scenario matrix

| Scenario | codestral:22b | devstral:24b |
|---|---|---|
| implement/go-handler | ✓ 51s | ✓ 1m33s |
| implement/theme-toggle | ✓ 1m09s | ✓ 1m13s |
| implement/ask-mode-no-write | ✗ 1m07s | ✓ 1m25s |
| chat/dm-backend-workspace | ✓ 1m55s | ✓ 1m38s |
| chat/dm-backend-echo-followup | ✗ 3m13s | ✓ 1m48s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
