# Chat quality scenarios (live hub)

General **1:1 / channel chat** regression harness — separate from collaboration scenarios in [COLLABORATION.md](COLLABORATION.md). Tests multi-turn conversation quality with real agents (not mocks).

## Two layers

| Layer | What | When |
|-------|------|------|
| **A — CI** | `internal/agent/chat_quality_router_test.go`, `chat_quality_coverage_test.go`, shortcut/echo unit tests | `make test-go` (every commit) |
| **B — Live** | JSON under `scenarios/chat/` + `scripts/chat-scenarios.py` | Local / pre-release with hub + Ollama |

Layer A catches **routing** (intent, mode, closure, history caps, workspace visibility classification, scan shortcut heuristics, deterministic reply helpers).

Layer B catches **multi-turn behavior** (echo, workspace visibility answers, fake package hallucinations, tool dumps on greetings).

## Prerequisites

- Hub running: `make server` or `make gui`
- Agents in each scenario's `required_agents` online and not paused
- Models configured (e.g. Ollama) for agents that use local LLMs
- Optional: `NEURAL_JUNKIE_DEBUG=1` on the hub for `assert_debug_context` steps
- For sweeps: start the hub with `NEURAL_JUNKIE_RATE_LIMIT=0`. The Makefile sets this on the scenario client only.

## Commands

```bash
# List scenarios (name + tags)
make chat-scenarios-list
python3 scripts/chat-scenarios.py --list
python3 scripts/chat-scenarios.py --list --tag dm --tag regression

# One scenario
make chat-scenario SCENARIO=dm-backend-workspace VERBOSE=1

# All scenarios
make chat-scenarios

# Filter by tag (scenario must include ALL listed tags)
make chat-scenarios-dm
make chat-scenarios-regression
NEURAL_JUNKIE_RATE_LIMIT=0 python3 scripts/chat-scenarios.py --all --tag backend --tag workspace
```

Environment: `NEURAL_JUNKIE_HUB_URL` — default `http://127.0.0.1:18765`

## Agent & channel coverage

| Scenario | Channel | Agent | Tags |
|----------|---------|-------|------|
| `greeting-chat-mode` | public | Assistant | public, assistant, greeting |
| `thanks-closure` | public | Assistant | public, assistant, closure |
| `already-said-closure` | public | Assistant | public, assistant, closure, regression |
| `casual-opinion-chat` | public | Assistant | public, assistant, chat |
| `task-flip-review` | public | Assistant | public, assistant, task |
| `public-backend-theme-workspace` | public | BackendEngineer | public, backend, workspace, regression |
| `dm-greeting` | DM | Assistant | dm, assistant, greeting |
| `dm-backend-workspace` | DM | BackendEngineer | dm, backend, workspace, regression |
| `dm-backend-echo-followup` | DM | BackendEngineer | dm, backend, echo, regression |
| `dm-frontend-greeting` | DM | FrontendEngineer | dm, frontend, greeting |
| `dm-frontend-code-task` | DM | FrontendEngineer | dm, frontend, task |
| `dm-security-review` | DM | SecurityReviewer | dm, security, task |
| `dm-architect-outline` | DM | SoftwareArchitect | dm, architecture, substantive |
| `dm-platform-greeting` | DM | PlatformEngineer | dm, devops, greeting |
| `dm-database-greeting` | DM | DatabaseSpecialist | dm, database, greeting |
| `dm-code-reviewer-task` | DM | CodeReviewer | dm, code-review, task |
| `dm-biology-greeting` | DM | BiologyExpert | dm, biology, greeting, life-sciences *(optional)* |

Scenarios marked `"optional": true` **skip** (exit 0) when required agents are offline (e.g. BiologyExpert without life-sciences pack).

## Regression scenarios (run before release)

These cover the conversation bugs we hit in production chat:

- **`dm-backend-workspace`** / **`public-backend-theme-workspace`** — theme ask → “can you see my workspace?” must not return fake `golang.org/x/themes` / Gin advice
- **`dm-backend-echo-followup`** — “What?” after a long reply must not quote the first user message
- **`already-said-closure`** — “I know you said that already” → canned won't-repeat closure

```bash
make chat-scenarios-regression
```

## Scenario JSON

Steps: `send`, `wait_reply`, `assert_messages`, `assert_reply_count`, `assert_debug_context`.

```json
{
  "name": "dm-backend-workspace",
  "tags": ["dm", "backend", "workspace", "regression"],
  "channel_type": "dm",
  "dm_user": "ChatScenario",
  "target_agent": "BackendEngineer",
  "required_agents": ["BackendEngineer"],
  "steps": [
    {
      "action": "send",
      "content": "can you see my workspace I have open?",
      "metadata": { "conversation_mode": "code", "context_scope": "outline" }
    },
    { "action": "wait_reply", "from": "BackendEngineer", "timeout": "90s" },
    {
      "action": "assert_messages",
      "last_reply_only": true,
      "any_match": ["workspace context|file tree|Yes — I have"],
      "none_match": ["golang.org/x/themes", "gin-gonic"]
    }
  ],
  "cleanup": "clear"
}
```

Public scenarios use `"channel": "chat-scenarios"`, `"mention": "@AgentName"`, and `ensure_channel_with_agents` to join agents.

DM scenarios use `"channel_type": "dm"` — channel created via `/api/channels/create`.

## Adding a scenario

1. Copy a similar JSON from `scenarios/chat/`.
2. Set `tags`, `target_agent`, `required_agents`, and assertions (`any_match` / `none_match`).
3. Run `make chat-scenario SCENARIO=your-name VERBOSE=1`.
4. Add a matching **Layer A** case in `chat_quality_router_test.go` or `chat_quality_coverage_test.go` if the bug was routing-related.

## Interpreting failures

- Per-step ✓/✗ in stdout; last ~12 transcript lines on failure.
- **Agent offline:** check `GET /api/agents`; optional scenarios skip instead of fail.
- **Timeout:** increase `timeout` or check agent logs.
- **assert_debug_context:** hub needs `NEURAL_JUNKIE_DEBUG=1`.
- **HTTP 429:** restart hub with `NEURAL_JUNKIE_RATE_LIMIT=0`.

## Related

- [CONTEXT_MODEL.md](CONTEXT_MODEL.md) — Conversation Context Stack (Layer A)
- [COLLABORATION.md](COLLABORATION.md) — collab scenario harness
- [marketing/CONVERSATIONAL-TEST-HARNESS.md](marketing/CONVERSATIONAL-TEST-HARNESS.md) — overview
