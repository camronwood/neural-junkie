# Model benchmark — quick

**Run:** 2026-06-29 17:55 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 5 implement + 2 chat scenarios (~15–45 min per model on ≤24B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|
| 1 | `nj-ornith:9b` | 3/7 | 43% | 2/5 | 1/2 | 48m39s |  |

## Per-scenario matrix

| Scenario | nj-ornith:9b |
|---|---|
| implement/go-handler | ✓ 6m03s |
| implement/theme-toggle | ✗ 20m17s |
| implement/ask-mode-no-write | ✓ 1m45s |
| implement/go-test-failure-repair | ✗ 2m33s |
| implement/typescript-compile-error-fix | ✗ 14m18s |
| chat/dm-backend-workspace | ✓ 1m29s |
| chat/dm-backend-echo-followup | ✗ 2m00s |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write, go-test-failure-repair, typescript-compile-error-fix
- **Chat:** dm-backend-workspace, dm-backend-echo-followup

## Hardware

- **RAM:** 24 GB (recommended tier)

## Deliverable judge

- **Provider:** `gemini`
