# LinkedIn article — Context Stack (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/context-stack/creatives/neural-junkie-context-stack-1200.png` (1200×627)

**Feed ad image:** `campaigns/context-stack/creatives/neural-junkie-context-stack-ad-1080.png` (1080×1080)

**Regenerate images:** `./scripts/compose-context-stack-article.sh`

**Suggested title (pick one):**
- Agents Don't Share a Brain. They Share Context — On Purpose.
- How Neural Junkie Builds, Uses, and Shares Agent Context
- The Conversation Context Stack: Why Your Local Agents Stay Grounded

**Feed post teaser:**
> Most multi-agent demos dump the whole transcript into every model call. Neural Junkie builds context turn-by-turn through a six-stage stack — mode, intent, memory, grounding, persona, budget — then shares only what each specialist actually needs: channel history, delegation results, collab plans, personal learnings. Not a hive mind. Scoped context.

**Hashtags:** `#AI #LocalAI #MultiAgent #DeveloperTools #OpenSource #Ollama #RAG`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

---

## Ad / feed copy variants

**Variant A — problem/solution:**
> Stuff the whole chat into every agent and you blow the context window. Starve them and they forget decisions. Neural Junkie's Conversation Context Stack builds the right prompt per turn — then shares context through channels, collabs, and silent delegation, not one global blob.

**Variant B — architecture hook:**
> Six stages before the LLM call: mode → intent → memory → grounding → persona → budget. Workspace files only when you're coding. "Thanks" never wakes a 14B model. That's how local multi-agent stays fast.

**Variant C — sharing clarity:**
> Agents on the same channel see the same transcript — but each rebuilds its own prompt. Delegation injects specialist answers. Collabs share plans and task paths. Personal learnings travel across agents when you confirm them. Sharing is explicit.

---

## PASTE START

Open a long `#general` thread. Ping a backend specialist. Ask a security reviewer for a second opinion. Start a `/collaborate` run that lasts three hours.

Every one of those turns needs **context** — but not the *same* context, and not *all* of it.

If you stuff the full transcript, open files, and tool dumps into every model call, local multi-agent dies on latency and RAM. If you send almost nothing, agents reinvent decisions you already made.

**Neural Junkie** is my personal open-source hub for running AI specialists on your hardware. The bet isn't a shared global brain. It's a **Conversation Context Stack** that builds the right prompt per turn — and explicit sharing paths when specialists need to see each other's work.

## Build: six stages before the LLM

Every user message flows through the same pipeline before any model call:

1. **Mode** — chat, code, or collab (composer chip or auto from signals like file paths and code verbs)
2. **Intent** — closure, casual, meta, substantive, or task
3. **Memory** — rolling session summary + a capped history slice (threads stay thread-scoped)
4. **Grounding** — workspace scope from none → hint → outline → focus → full
5. **Persona** — direct DM, multi-agent channel, or collaboration framing
6. **Budget** — byte caps per section; compress with retrieve refs instead of silent truncation

This is the part most demos skip. They ask a big model to "behave" with a giant prompt. We **route context first**.

"Thanks!" → canned closure. No LLM. No workspace scan. No MCP tool block.

A casual DM → minimal prompt, two history rows, summary if present.

A refactor with `src/auth/middleware.go` open → code mode, task intent, focus or full scope, specialist tools on.

## Use: what actually lands in the prompt

The agent assembles a **system** prompt and a **user** prompt from labeled sections — so you can debug what the model saw:

System side: persona rules, pack domain blocks, MCP tools (when gated on), collaboration plan/lanes, user + project rules, session summary, conversation memory hits, personal learnings, durable conversation state.

User side: your message, attached uploads, ambient editor state, `=== WORKSPACE CONTEXT ===`, linked repos, and any hub-data you explicitly granted.

Then a **turn pipeline** runs knowledge retrievers when the intent warrants it — codebase search, code graph, memory, learnings, prior references — and stamps metadata so you can audit what was injected.

When context would overflow, **CCR** (context compression + retrieve) stores oversized sections under `ctx-…` refs. The agent can pull them back with `nj_retrieve_context` instead of losing them quietly.

## Share: scoped, not hive-mind

There is no single shared context blob across all agents. Sharing is intentional:

**Same channel** — everyone sees the same transcript; each agent rebuilds its own prompt from that history.

**Delegation** — the hub silently consults another specialist and injects `=== DELEGATE_RESULTS ===` into the responder's prompt. The user never leaves the conversation.

**Agent Review** — you @mention a different specialist in a thread for a second opinion.

**Collaboration** — plan, tasks, participants, and workspace context travel with the collab; task dispatch can attach slimmed path context so workers don't need the whole repo dump.

**Personal learnings** — user-confirmed memories scoped to an agent, the workspace, a collaboration, or globally — portable across sessions when you opt in.

**Conversation memory** — local embeddings over channel history and collab markdown (`plan.md`, findings, etc.), retrieved as `=== RELEVANT PAST CONTEXT ===` when the question needs yesterday's decision.

**Packs** don't reimplement the stack. They contribute capabilities and domain prompt appendices; the hub still owns assembly.

## What you control

- **Composer mode** — Chat / Code / Auto
- **Workspace scope chip** — how much of the open project goes into the turn
- **Linked workspaces** — multi-repo project sets with cross-repo hints
- **Conversation memory + personal learning** — Settings toggles, local SQLite / JSON under `~/.neural-junkie`
- **Debug** — `GET /api/debug/channel-context` when you need to see summary, mode, intent, persona, and budget stats

## Why this matters for local AI

Cloud agents can afford giant windows and opaque retrieval. On your laptop, **what you put in the prompt is the product**.

Build context in stages. Use only what the turn needs. Share through channels, collabs, and delegation — not by cloning one mega-prompt into every specialist.

Docs deep dive: `docs/CONTEXT_MODEL.md` in the repo.

Download: https://github.com/camronwood/neural-junkie/releases/latest

Camron Wood — Neural Junkie (personal project)

## PASTE END
