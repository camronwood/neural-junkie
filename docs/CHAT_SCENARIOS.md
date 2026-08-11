# Chat quality scenarios (live hub)

General **1:1 / channel chat** regression harness — separate from collaboration scenarios in [COLLABORATION.md](COLLABORATION.md). Tests multi-turn conversation quality with real agents (not mocks).

## Two layers

| Layer | What | When |
|-------|------|------|
| **A — CI** | `internal/agent/chat_quality_router_test.go`, `chat_quality_coverage_test.go`, shortcut/echo unit tests | `make test-go` (every commit) |
| **B — Live** | JSON under `scenarios/chat/` + `scripts/chat-scenarios.py` | Local / pre-release with hub + Ollama |

Layer A catches **routing** (intent, mode, closure, history caps, workspace visibility classification, scan shortcut heuristics, deterministic reply helpers).

Layer B catches **multi-turn behavior** (echo, workspace visibility answers, fake package hallucinations, tool dumps on greetings). Implement scenarios also use `assert_suggested_commands` (shared with `implement-scenarios.py`).

## Prerequisites

- Hub running: `make server` or `make gui`
- **Full regression hub:** `make server-regression` (`NEURAL_JUNKIE_RATE_LIMIT=0` + `NEURAL_JUNKIE_DEBUG=1` on the **server**). See [TESTING.md](TESTING.md).
- Agents in each scenario's `required_agents` online and not paused
- Models configured (e.g. Ollama) for agents that use local LLMs
- Debug context assertions: `make chat-scenarios-debug` (`--require-debug`; scenarios tagged `debug`)
- For sweeps: hub must have `NEURAL_JUNKIE_RATE_LIMIT=0` (via `server-regression`). Makefile also sets it on scenario clients.

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
| `thanks-closure` | public | Assistant | public, assistant, closure, regression |
| `already-said-closure` | public | Assistant | public, assistant, closure, regression |
| `casual-opinion-chat` | public | Assistant | public, assistant, chat |
| `task-flip-review` | public | Assistant | public, assistant, task |
| `public-backend-theme-workspace` | public | BackendEngineer | public, backend, workspace, regression |
| `dm-greeting` | DM | Assistant | dm, assistant, greeting |
| `dm-backend-workspace` | DM | BackendEngineer | dm, backend, workspace, regression |
| `dm-backend-echo-followup` | DM | BackendEngineer | dm, backend, echo, regression |
| `dm-backend-deep-continuation` | DM | BackendEngineer | dm, backend, continuation, regression |
| `dm-topic-switch` | DM | BackendEngineer | dm, backend, topic-switch, regression |
| `dm-assistant-continue-after-closure` | DM | Assistant | dm, assistant, closure, continuation, regression |
| `dm-assistant-trip-followup` | DM | Assistant | dm, assistant, dialogue, regression |
| `dm-pronoun-followup-3turn` | DM | FrontendEngineer | dm, frontend, dialogue, regression |
| `dm-chat-mode-soft-followups` | DM | BackendEngineer | dm, backend, dialogue, regression |
| `dm-correction-soft-followup` | DM | Assistant | dm, assistant, dialogue, continuity, regression |
| `dm-entity-second-option-followup` | DM | FrontendEngineer | dm, frontend, dialogue, continuity, regression |
| `dm-topic-continuity-same-thread` | DM | Assistant | dm, assistant, dialogue, regression |
| `dm-summary-continuity-long-horizon` | DM | Assistant | dm, assistant, dialogue, long-horizon, regression |
| `dm-work-surface-mid-session-correction` | DM | Assistant | dm, assistant, long-horizon, work-surface, regression |
| `dm-work-surface-plan-stickiness` | DM | FrontendEngineer | dm, frontend, long-horizon, work-surface, regression |
| `dm-work-surface-entity-topic-switch` | DM | FrontendEngineer | dm, frontend, long-horizon, work-surface, regression |
| `dm-durable-state-chat-isolation` | DM | BackendEngineer | dm, backend, dialogue, regression |
| `dm-backend-interject-resume` | DM | BackendEngineer | dm, backend, interject, regression |
| `dm-frontend-greeting` | DM | FrontendEngineer | dm, frontend, greeting |
| `dm-frontend-code-task` | DM | FrontendEngineer | dm, frontend, task |
| `dm-security-review` | DM | SecurityReviewer | dm, security, task |
| `dm-architect-outline` | DM | SoftwareArchitect | dm, architecture, substantive |
| `dm-platform-greeting` | DM | PlatformEngineer | dm, devops, greeting |
| `dm-database-greeting` | DM | DatabaseSpecialist | dm, database, greeting |
| `dm-backend-code-review` | DM | BackendEngineer | dm, backend, code-review, task |
| `dm-biology-greeting` | DM | BiologyExpert | dm, biology, greeting, life-sciences *(optional)* |

Scenarios marked `"optional": true` **skip** (exit 0) when required agents are offline (e.g. BiologyExpert without life-sciences pack).

## Regression scenarios (run before release)

These cover the conversation bugs we hit in production chat:

- **`dm-backend-workspace`** / **`public-backend-theme-workspace`** — theme ask → “can you see my workspace?” must not return fake `golang.org/x/themes` / Gin advice
- **`dm-backend-echo-followup`** — “What?” after a long reply must not quote the first user message
- **`already-said-closure`** — “I know you said that already” → canned won't-repeat closure
- **`dm-backend-deep-continuation`** — “go deeper on the approach” stays on theme without echoing turn 1
- **`dm-topic-switch`** — code → chat opinion → code without workspace dumps on the chat turn
- **`dm-assistant-continue-after-closure`** — thanks closure then a new question gets a substantive answer
- **`dm-backend-interject-resume`** — channel interject holds agents until the user sends again (requires `make server-regression`)
- **Dialogue continuity** (`tags: dialogue`) — multi-turn thread holding (Layer A companions required):
  - **`dm-assistant-trip-followup`** — trip → enable websearch → stay on trip; never `wrong_route` / FrontendEngineer
  - **`dm-pronoun-followup-3turn`** — anaphoric “move it …” retains ThemeSettings/Appearance
  - **`dm-chat-mode-soft-followups`** — opinion → why → one more tradeoff without tool dumps
  - **`dm-correction-soft-followup`** — city correction → soft “second option” stays answer/none
  - **`dm-entity-second-option-followup`** — ThemeSettings options → “second option?” → “why?” without tools
  - **`dm-topic-continuity-same-thread`** — 4-turn same theme retention
  - **`dm-summary-continuity-long-horizon`** — early constraint survives past summary refresh
  - **`dm-durable-state-chat-isolation`** — code turn then chat opinion without implement dumps
- **Work surface** (`tags: work-surface`) — long-horizon retain/correct/stick pack for `make surface-reliability`:
  - **`dm-work-surface-mid-session-correction`** — page-count constraint survives a genre correction
  - **`dm-work-surface-plan-stickiness`** — AppearanceToggle plan does not revive a rejected segmented control
  - **`dm-work-surface-entity-topic-switch`** — DisplayPreferences survives a topic switch

```bash
make chat-scenarios-regression
make conversation-scenarios-regression
python3 scripts/chat-scenarios.py --all --tag dialogue
```

Tag `dialogue` / `coherence` scenarios must have ≥3 user sends and continuity asserts (`assert_transcript_metrics` and/or `any_match` + `none_match`) — enforced by `scripts/lib/scenario_contract.py`.

## Scenario JSON

Steps: `send`, `wait_reply`, `channel_interject`, `wait_no_reply`, `assert_messages`, `assert_reply_count` (`since_baseline: true` counts from last send/interject baseline), `assert_debug_context`.

`assert_messages` supports hard `none_match` / `max_chars`, optional soft `any_match` (`"optional": true` or `"any_match_optional": true`), and optional `semantic_turn_decision` (hub-stamped routing). Prefer semantic/metadata asserts when checking routing; phrase match is a soft signal only.

For **implement / parity / user-flow implement** completion, do **not** gate on chat phrases — use disk waits (`until_file_match`, `until_file_absent`) and/or `until_metadata_keys: ["implementation_session_outcome"]`. See [TESTING.md](TESTING.md).

`channel_interject` calls `POST /api/channels/:channel/interject`. `wait_no_reply` asserts no new agent messages for `duration` (use after interject; optional `retries` + `reinterject_on_retry`).

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
2. Set `tags`, `target_agent`, `required_agents`, and assertions (`none_match` hard gates; `any_match` for content quality; use `semantic_turn_decision` when checking routing).
3. For implement-style scenarios, declare `expect_deliverables` and wait on disk/metadata — not canned session phrases.
4. Run `make chat-scenario SCENARIO=your-name VERBOSE=1`.
5. Add a matching **Layer A** case in `chat_quality_router_test.go`, `turn_intent_test.go`, or `dialogue_continuity_test.go` if the bug was routing/history-related. **Dialogue** scenarios must always pair with a Layer A companion.

### Auto-authored scenarios (test-growth loop)

When `make test-growth-loop` adds or strengthens live scenarios:

- Follow the same JSON patterns as manual scenarios above.
- Always include `none_match` guardrails on regression scenarios when a known failure mode exists.
- Pair routing-related chat scenarios with Layer A unit tests.
- The loop rejects assertion weakening (removed `none_match`, dropped `expect_deliverables` quality bars).
- If a new test exposes a product defect, the loop hands off to `make layer-fix-loop` instead of patching product code.

See [TESTING.md](TESTING.md) for `test-growth-list`, `test-growth-loop`, and report locations under `docs/testing/test-growth-*.md`.

## Interpreting failures

- Per-step ✓/✗ in stdout; last ~12 transcript lines on failure.
- **Agent offline:** check `GET /api/agents`; optional scenarios skip instead of fail.
- **Timeout:** increase `timeout` or check agent logs.
- **assert_debug_context:** hub needs `NEURAL_JUNKIE_DEBUG=1`.
- **HTTP 429:** restart hub with `NEURAL_JUNKIE_RATE_LIMIT=0`.

## Related

- [CONTEXT_MODEL.md](CONTEXT_MODEL.md) — Conversation Context Stack (Layer A)
- [COLLABORATION.md](COLLABORATION.md) — collab scenario harness
- [marketing/CONVERSATIONAL-TEST-HARNESS.md](../campaigns/test-harness/CONVERSATIONAL-TEST-HARNESS.md) — overview
