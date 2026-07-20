# Model benchmark — release

**Run:** 2026-07-19 17:46 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Release gate — benchmark winners only (~10–20 min).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 8/9 | 89% | 6/6 | 2/2 | 3m50s |  |
| 2 | `qwen2.5-coder:14b` | 8/9 | 89% | 6/6 | 2/2 | 3m59s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 30s | ✓ 30s |
| implement/theme-toggle | ✓ 55s | ✓ 55s |
| implement/ask-mode-no-write | ✓ 29s | ✓ 29s |
| implement/go-test-failure-repair | ✓ 7s | ✓ 7s |
| implement/rules-constrained-implement | ✓ 7s | ✓ 7s |
| implement/plan-mode-no-write | ✓ 21s | ✓ 25s |
| chat/dm-backend-workspace | ✓ 38s | ✓ 28s |
| chat/dm-backend-echo-followup | ✓ 39s | ✓ 36s |
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

- **Provider:** `ollama/qwen3.5:9b`
