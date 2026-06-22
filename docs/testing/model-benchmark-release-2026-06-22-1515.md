# Model benchmark — release

**Run:** 2026-06-22 15:15 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Release gate — benchmark winners only (~10–20 min).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 7/8 | 88% | 5/6 | 2/2 | 20m37s |  |
| 2 | `qwen2.5-coder:14b` | 7/8 | 88% | 5/6 | 2/2 | 22m44s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 2m29s | ✓ 32s |
| implement/theme-toggle | ✓ 52s | ✓ 38s |
| implement/ask-mode-no-write | ✓ 42s | ✓ 33s |
| implement/go-test-failure-repair | ✗ 15m03s | ✗ 15m04s |
| implement/rules-constrained-implement | ✓ 1m11s | ✓ 1m39s |
| implement/plan-mode-no-write | ✓ 31s | ✓ 29s |
| chat/dm-backend-workspace | ✓ 37s | ✓ 50s |
| chat/dm-backend-echo-followup | ✓ 1m17s | ✓ 52s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, rules-constrained-implement, plan-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini`
