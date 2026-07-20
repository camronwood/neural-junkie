# Model benchmark — release

**Run:** 2026-07-20 03:26 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Release gate — benchmark winners only (~10–20 min).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 8/9 | 89% | 6/6 | 2/2 | 3m36s |  |
| 2 | `qwen2.5-coder:14b` | 8/9 | 89% | 6/6 | 2/2 | 4m09s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 23s | ✓ 21s |
| implement/theme-toggle | ✓ 23s | ✓ 17s |
| implement/ask-mode-no-write | ✓ 29s | ✓ 31s |
| implement/go-test-failure-repair | ✓ 9s | ✓ 11s |
| implement/rules-constrained-implement | ✓ 7s | ✓ 7s |
| implement/plan-mode-no-write | ✓ 53s | ✓ 29s |
| chat/dm-backend-workspace | ✓ 48s | ✓ 51s |
| chat/dm-backend-echo-followup | ✓ 45s | ✓ 35s |
| arena/logic-set | ✗ 0s | ✗ 0s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, rules-constrained-implement, plan-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
- **Collab:** (none)
- **Arena:** logic-set
- **CAD:** (none)
- **External:** (none)

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `claude`
