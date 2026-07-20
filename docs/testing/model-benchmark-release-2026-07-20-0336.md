# Model benchmark — release

**Run:** 2026-07-20 03:36 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Release gate — benchmark winners only (~10–20 min).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 9/9 | 100% | 6/6 | 2/2 | 3m26s | winner |
| 2 | `qwen2.5-coder:14b` | 9/9 | 100% | 6/6 | 2/2 | 4m16s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 23s | ✓ 21s |
| implement/theme-toggle | ✓ 21s | ✓ 17s |
| implement/ask-mode-no-write | ✓ 40s | ✓ 29s |
| implement/go-test-failure-repair | ✓ 7s | ✓ 7s |
| implement/rules-constrained-implement | ✓ 7s | ✓ 7s |
| implement/plan-mode-no-write | ✓ 27s | ✓ 23s |
| chat/dm-backend-workspace | ✓ 40s | ✓ 29s |
| chat/dm-backend-echo-followup | ✓ 42s | ✓ 35s |
| arena/logic-set | ✓ 35s | ✓ 24s |

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
