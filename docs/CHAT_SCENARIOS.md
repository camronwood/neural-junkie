# Chat quality scenarios (live hub)

General **1:1 / channel chat** regression harness — separate from collaboration scenarios in [COLLABORATION.md](COLLABORATION.md). Tests multi-turn conversation quality with real agents (not mocks).

**CI (Layer A):** `internal/agent/chat_quality_router_test.go` — table-driven router tests (intent, mode, closure, tooling, history caps). Runs via `make test-go`.

**Local (Layer B):** JSON scenarios under `scenarios/chat/` driven by `scripts/chat-scenarios.py`.

## Prerequisites

- Hub running: `make server` or `make gui`
- **Assistant** (or agents named in scenario `required_agents`) online and not paused
- Models configured (e.g. Ollama) for agents that use local LLMs
- Optional: `NEURAL_JUNKIE_DEBUG=1` on the hub for `assert_debug_context` steps (`task-flip-review`)
- For sweeps: **start the hub** with `NEURAL_JUNKIE_RATE_LIMIT=0` (e.g. `NEURAL_JUNKIE_RATE_LIMIT=0 make server`). The Makefile only sets this on the scenario client, not on a already-running hub.

## Commands

```bash
# List scenarios
python3 scripts/chat-scenarios.py --list

# One scenario
make chat-scenario SCENARIO=greeting-chat-mode

# All chat scenarios
make chat-scenarios

# Keep history for inspection
make chat-scenario SCENARIO=thanks-closure KEEP=1 VERBOSE=1
```

Environment:

- `NEURAL_JUNKIE_HUB_URL` — default `http://127.0.0.1:18765`

## Scenarios

| Name | Channel | Purpose |
|------|---------|---------|
| `greeting-chat-mode` | public `chat-scenarios` | Chat mode hello; no MCP/tool dumps; reply length cap |
| `thanks-closure` | public | Substantive Q → `ok thanks` → canned closure |
| `already-said-closure` | public | “I know you said that already” → won't-repeat closure |
| `casual-opinion-chat` | public | Chat mode opinion; no grounding/tool spam |
| `task-flip-review` | public | Hello then `review …` with code mode; debug intent `task` |
| `dm-greeting` | DM via API | Real DM channel; same structural checks as greeting |

## Scenario JSON

Steps: `send`, `wait_reply`, `assert_messages`, `assert_reply_count`, `assert_debug_context`.

```json
{
  "channel": "chat-scenarios",
  "channel_type": "public",
  "mention": "@Assistant",
  "required_agents": ["Assistant"],
  "steps": [
    { "action": "send", "content": "hello", "metadata": { "conversation_mode": "chat" } },
    { "action": "wait_reply", "from": "Assistant", "timeout": "90s" },
    { "action": "assert_messages", "last_reply_only": true, "none_match": ["MCP"] }
  ],
  "cleanup": "clear"
}
```

DM scenarios use `"channel_type": "dm"`, `"dm_user": "ChatScenario"`, `"target_agent": "Assistant"` (no `channel` name — created via `/api/channels/create`).

Public scenarios call `ensure_channel_with_agents`: the runner creates `chat-scenarios` (if needed) and **joins** each `required_agents` entry via `/api/channels/join` so in-process agents subscribe within ~2s.

## Interpreting failures

- The runner prints per-step pass/fail.
- On failure, the last ~12 transcript lines are dumped to stderr.
- Common issues:
  - **Agent offline:** start hub with Assistant enabled; check `GET /api/agents`.
  - **Timeout on wait_reply:** model slow or agent did not respond to mention; increase `timeout` or check logs.
  - **assert_debug_context:** hub needs `NEURAL_JUNKIE_DEBUG=1`.
  - **HTTP 429:** restart hub with `NEURAL_JUNKIE_RATE_LIMIT=0`.

## Related

- [CONTEXT_MODEL.md](CONTEXT_MODEL.md) — Conversation Context Stack (what Layer A asserts)
- [COLLABORATION.md](COLLABORATION.md) — collab scenario harness (`scenarios/collab/`)
