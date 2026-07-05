# Context model v2

Neural Junkie assembles agent context through a **Conversation Context Stack** — a single pipeline with explicit stages, budgets, and metadata contracts. v2 replaces the earlier two-layer doc (intent router + DM session summary only).

## Goals and non-goals

**Goals**

- Pleasant 1:1 chat (natural tone, no tool dumps on greetings)
- Coherent memory across longer conversations
- Code/repo grounding only when the user is actually coding
- Predictable agent persona (DM vs channel vs collaboration)

**Non-goals**

- Full RAG or vector retrieval rewrite (chat/casual stay on small budgets)
- Cross-session long-term memory for casual chat (see [FUTURE_ENHANCEMENTS.md](FUTURE_ENHANCEMENTS.md); **agent-runtime v2** uses full Context Stack + CCR + workspace learnings)
- CLI agent context unification (see [CLI_AGENTS.md](CLI_AGENTS.md) follow-up)

## Agent-runtime v2 exception

When `features.agent_runtime_v2` is enabled (default) and the user is in **Agent** / implementation mode:

- Prompt budget derives from `ollama.num_ctx` ([HARDWARE.md](HARDWARE.md) tiers)
- CCR tool compression uses elevated `agent_runtime_max_tool_bytes` and `nj_retrieve_context` per-turn budget
- Per-file read caps use chunked `read_file` (512 KB agent reads) instead of silent 50 KB truncation
- See [CURSOR_PARITY.md](CURSOR_PARITY.md)

## Baseline fixes (v2 foundation)

These ship as part of the v2 rollout:

- **@mention overrides IDE route** — explicit `@Assistant` no longer loses to `ide_route_agent_type: backend` (`internal/agent/agent.go`, `desktop/src/utils/ideComposer.ts`)
- **Session persist slimming** — `workspace_context` bodies are outline-only on disk; runtime prompts stay rich (`internal/hub/session_persist.go`)

## Conversation Context Stack

Every user turn flows through six stages before the LLM call:

```mermaid
flowchart TB
  userMsg[User message]
  mode[1 Mode: chat / code / collab]
  intent[2 Intent: closure / casual / substantive / task]
  memory[3 Memory: session summary + history slice]
  grounding[4 Grounding: context_scope + optional scan]
  persona[5 Persona: direct / channel / collaboration]
  budget[6 Budget: byte caps per section]
  llm[LLM call]

  userMsg --> mode --> intent --> memory --> grounding --> persona --> budget --> llm
```

| Stage | Input | Metadata / state | Owner |
|-------|-------|------------------|-------|
| **Mode** | Composer UI + heuristics | `conversation_mode`: `chat` \| `code` \| `collab` | Desktop |
| **Intent** | Message text + channel type | Internal: `closure` / `casual` / `substantive` / `task` / `meta` | Server `turn_intent.go` |
| **Memory** | Channel transcript | `session_summary` (hub) + history slice | Hub + agent |
| **Grounding** | Mode + intent + scope | `context_scope` + optional file scan | Desktop + agent |
| **Persona** | Channel type | Prompt framing tier | Agent `buildPrompt` |
| **Budget** | Assembled prompt | Truncation order before LLM | Agent `context_budget.go` |

Implementation entry points: `generateResponse` / `generateResponseStreaming` in `internal/agent/agent.go`.

## Conversation modes

Three modes control how much context is attached:

| Mode | Default `context_scope` | Workspace scan | MCP / FILE_CHANGE in prompt | History cap |
|------|---------------------------|----------------|-----------------------------|-------------|
| **chat** | `none` | off | off | summary + 4 turns |
| **code** | auto (`inferContextScope`) | on when paths/verbs | on for specialists | summary + 8 turns |
| **collab** | collab rules | per collab phase | per phase | collab transcript rules |

**Desktop UX:** Composer mode chip (Auto / Chat / Code) alongside the workspace context cycle. Persisted in `localStorage`. Outbound metadata key: `conversation_mode`.

**Auto (default):** Infer `chat` vs `code` from message signals in `desktop/src/utils/conversationMode.ts` (code verbs, file paths, `@codebase`, IDE layout with open tab). Server re-infers when metadata is missing.

## Turn intent router

Each message is classified before building the prompt:

| Intent | Behavior | History rows | Session summary |
|--------|----------|--------------|-----------------|
| `closure` | Canned reply (thanks, brief ack); no LLM | 0 | n/a |
| `casual` | Minimal prompt (was `low_signal`) | 2 | yes if present |
| `meta` | Minimal prompt when user asks about prompt/context | 2 | yes if present |
| `substantive` | Full agent prompt | 4–8 | yes if present |
| `task` | Full prompt + workspace scan when paths/verbs | 8 | yes if present |

**Mode interaction:** `conversation_mode=chat` biases toward `casual` unless the message has a substantive question (`?` + length) or explicit code/task signals.

Collaboration channels use closure for acks; they always keep the full collab prompt for substantive collab turns.

Implementation: `internal/agent/turn_intent.go`.

## Agent delegation

When [delegation](DELEGATION.md) is enabled, after intent classification the hub may consult other specialists and inject `=== DELEGATE_RESULTS ===` before the main LLM call. **Skipped for `closure` and `casual` intents.**

## Session summary (memory)

The hub maintains a rolling **session summary** per eligible channel:

- Updated asynchronously with `qwen2.5:3b` (`config.SessionSummaryOllamaModel`) after every 3 user turns (or sooner when empty and enough transcript exists)
- **90s** LLM timeout (`NJ_SESSION_SUMMARY_TIMEOUT`); override model via `NJ_SESSION_SUMMARY_MODEL`
- **Eligible channels:** DM, custom, public (`#general`, etc.), and `dm-*` specialist slugs — **not** regression harness channels (`implement-scenarios`, `chat-scenarios`, etc.)
- Persisted in `last-session.json` as `session_summary` / `session_summary_at`
- Cleared with **Clear message history**
- Injected into agent prompts as `=== SESSION SUMMARY ===`

Implementation: `internal/hub/channel_summary.go`, `internal/chatcontext` for transcript filtering.

### Thread-scoped history

When `msg.IsInThread()`, history for the LLM is **thread messages + parent** only — not the full channel transcript. Implementation: `historyForGeneration` in `internal/agent/history_llm.go`.

## Persona tiers

Prompt framing in `buildPrompt` uses three tiers:

| Tier | When | System prompt shape |
|------|------|---------------------|
| **direct** | DM / single-agent channel | 1:1 with the user; no peer agent list |
| **channel** | `#general`, multi-agent | Multi-agent chat room framing |
| **collaboration** | Collab channels | Existing collab blocks (unchanged) |

MCP tools and `[FILE_CHANGE]` docs are **off** for `direct` + `casual`; **on** for `code`/`task` + specialists.

Implementation: `promptPersonaTier()` in `internal/agent/prompt_persona.go`.

## Context budget

Before the LLM call, `applyContextBudget()` enforces a ~32KB prompt target (tunable):

1. User message + attachments (required, never truncated)
2. Session summary (cap 2KB)
3. Recent turns in prompt body (cap 12KB)
4. Workspace outline (cap 4KB)
5. Open file bodies / scan (remainder, cap 12KB)

Truncate tail sections first. When context compression is enabled, oversized sections are stored in the hub cache with a ref marker instead of silent truncation — see [CONTEXT_COMPRESSION.md](CONTEXT_COMPRESSION.md).

Implementation: `internal/agent/context_budget.go`.

## Desktop ↔ server metadata contract

| Key | Set by | Read by |
|-----|--------|---------|
| `conversation_mode` | Desktop | Agent intent + grounding |
| `context_scope` | Desktop (`inferContextScope`) | Agent workspace append |
| `context_scope_reason` | Desktop | Debug UI / composer chip |
| `linked_workspaces` | Desktop | Agent multi-repo scope (outline + open tabs per linked repo) |
| `ide_route_agent_type` | Desktop IDE | Agent routing (skipped when `@mentions` present) |
| `session_summary` | Hub (persisted) | Agent prompt injection |

Keys must stay in sync between `desktop/src/constants/promptMetadata.ts` and `internal/agent/prompt_context.go`.

## Persistence rules

Runtime prompts may include full `workspace_context` (open files, selections). **Disk persistence** stores outline-only metadata:

- `open_files[].content` stripped
- `file_tree` capped at 12KB
- `prompt_attachments`, `user_images`, `granted_hub_data_access` bodies stripped

See `cloneMessageForSessionPersist` in `internal/hub/session_persist.go`.

## Migration and compatibility

- New metadata keys are optional; missing `conversation_mode` → server-side Auto inference
- No breaking changes to `last-session.json` schema
- During development: `NEURAL_JUNKIE_CONTEXT_V2=1` enables v2 intent/persona paths (default on in v2 release)

## Debug

When `NEURAL_JUNKIE_DEBUG=1`:

```bash
curl 'http://localhost:18765/api/debug/channel-context?channel=dm-camron-assistant'
```

Response includes: `session_summary`, `conversation_mode`, resolved intent (when message query param provided), persona tier, budget stats.

## Test plan

| Area | Test file |
|------|-----------|
| **Chat quality router (CI catalog)** | `internal/agent/chat_quality_router_test.go` |
| Intent + mode interaction | `internal/agent/turn_intent_test.go` |
| Mention overrides IDE route | `internal/agent/should_respond_test.go` |
| Session summary on public channels | `internal/hub/channel_summary_test.go` |
| Thread history | `internal/agent/history_llm_test.go` |
| Context budget truncation | `internal/agent/context_budget_test.go` |
| Composer mode metadata | `desktop/src/utils/conversationMode.test.ts` |
| DM persona (no MCP block) | `internal/agent/prompt_persona_test.go` |
| **Live multi-turn chat (local)** | `scenarios/chat/*.json` — `make chat-scenario` — see [CHAT_SCENARIOS.md](CHAT_SCENARIOS.md) |

## Preconditions

- Ollama running with `qwen2.5:3b` for session summaries (`ollama pull qwen2.5:3b`)
- Rebuild hub: `make stop && make start-all`

## Related

- [ARCHITECTURE.md](ARCHITECTURE.md) — system overview and context stack pointer
- [ADAPTIVE-ORCHESTRATION-NOTES.md](ADAPTIVE-ORCHESTRATION-NOTES.md) — external “adaptive intelligence” framing mapped to this stack
- [DELEGATION.md](DELEGATION.md) — cross-specialist consult after intent
- [CLI_AGENTS.md](CLI_AGENTS.md) — CLI subprocess context (separate stack today)
