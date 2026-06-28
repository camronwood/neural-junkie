# Model benchmark — quick

**Run:** 2026-06-18 05:23 UTC  
**Hub:** `http://127.0.0.1:18765`  
**Suite:** Smoke benchmark — 3 implement + 2 chat scenarios (~15–45 min per model on 14B class).

## Summary

| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |
|------|-------|------|------|-----------|------|------------|-------|

## Skipped models

- `qwen2.5-coder:14b` — model not installed (use --pull or ollama pull)
- `codestral:22b` — model not installed (use --pull or ollama pull)
- `devstral:24b` — model not installed (use --pull or ollama pull)
- `qwen3.5:9b` — model not installed (use --pull or ollama pull)
- `deepseek-coder:6.7b` — model not installed (use --pull or ollama pull)
- `codegemma:7b` — model not installed (use --pull or ollama pull)
- `gemma3:12b` — model not installed (use --pull or ollama pull)

## Per-scenario matrix

| Scenario |  |
|---||
| implement/go-handler |  |
| implement/theme-toggle |  |
| implement/ask-mode-no-write |  |
| chat/dm-backend-workspace |  |
| chat/dm-backend-echo-followup |  |

## Scenario lists

- **Implement:** go-handler, theme-toggle, ask-mode-no-write
- **Chat:** dm-backend-workspace, dm-backend-echo-followup
