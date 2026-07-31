# Model benchmark — release

**Run:** 2026-07-31 07:04 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Release gate — benchmark winners only (~10–20 min).

## Summary

| Rank | Configured model | Pass@1 | Eventual | Implement | Chat | Total time | Notes |
|------|------------------|--------|----------|-----------|------|------------|-------|
| 1 | `gemma3:12b` | 100% | 9/9 (100%) | 6/6 | 2/2 | 9m13s | winner |
| 2 | `qwen2.5-coder:14b` | 100% | 9/9 (100%) | 6/6 | 2/2 | 9m33s |  |

## Per-scenario matrix

| Scenario | qwen2.5-coder:14b | gemma3:12b |
|---|---|---|
| implement/go-handler | ✓ 24s | ✓ 26s |
| implement/theme-toggle | ✓ 16s | ✓ 27s |
| implement/ask-mode-no-write | ✓ 1m48s | ✓ 1m47s |
| implement/go-test-failure-repair | ✓ 30s | ✓ 28s |
| implement/rules-constrained-implement | ✓ 14s | ✓ 14s |
| implement/plan-mode-no-write | ✓ 42s | ✓ 33s |
| chat/dm-backend-workspace | ✓ 2m35s | ✓ 1m44s |
| chat/dm-backend-echo-followup | ✓ 2m00s | ✓ 2m40s |
| arena/logic-set | ✓ 42s | ✓ 31s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, rules-constrained-implement, plan-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
- **Collab:** (none)
- **User flows:** (none)
- **Arena:** logic-set
- **CAD:** (none)
- **External:** (none)

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `ollama/qwen3.5:9b`
