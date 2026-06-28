# Model benchmark — release

**Run:** 2026-06-22 08:08 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Release gate — benchmark winners only (~10–20 min).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `qwen2.5-coder:14b` | 3/4 | 75% | 3/4 | — | 0s |  |
| 2 | `gemma3:12b` | 3/4 | 75% | 3/4 | — | 0s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 1m47s | ✓ 39s |
| implement/theme-toggle | ✓ 48s | ✓ 42s |
| implement/ask-mode-no-write | ✓ 37s | ✓ 39s |
| implement/go-test-failure-repair | ✗ 15m03s | ✗ 15m03s |
| implement/rules-constrained-implement | — | — |
| implement/plan-mode-no-write | — | — |
| chat/dm-backend-workspace | — | — |
| chat/dm-backend-echo-followup | — | — |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, rules-constrained-implement, plan-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini/qwen2.5-coder:14b`
