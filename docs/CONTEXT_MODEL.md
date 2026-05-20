# Context model (intent router + session summary)

Neural Junkie routes each user turn through two layers before calling the agent's chat model.

## Phase A — Turn intent router

Each message is classified before building the prompt:

| Intent | Behavior |
|--------|----------|
| `closure` | Canned reply (thanks, already-said, brief ack after agent reply); no LLM |
| `low_signal` | Minimal prompt + 2 history rows |
| `meta` | Minimal prompt when user asks about prompt/context |
| `substantive` | Full agent prompt; 8 history rows (4 when session summary present) |

Collaboration channels use closure only; they always keep the full collab prompt.

Implementation: `internal/agent/turn_intent.go`, wired in `generateResponse` and `generateResponseStreaming`.

## Phase B — Hub session summary

For `dm` and `custom` channels the hub maintains a rolling **session summary**:

- Updated asynchronously with `qwen2.5:7b` (`config.UtilityOllamaModel`) after every 3 user turns (or sooner when empty and enough transcript exists).
- Persisted in `last-session.json` as `session_summary` / `session_summary_at` per channel.
- Cleared with **Clear message history**.
- Injected into agent prompts as `=== SESSION SUMMARY ===`.

Implementation: `internal/hub/channel_summary.go`, `internal/chatcontext` for transcript filtering.

## Debug

When `NEURAL_JUNKIE_DEBUG=1`:

```bash
curl 'http://localhost:18765/api/debug/channel-context?channel=dm-camron-assistant'
```

## Preconditions

- Ollama running with `qwen2.5:7b` for summaries.
- Rebuild hub: `make stop && make start-all`.
