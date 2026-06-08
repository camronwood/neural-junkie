# LinkedIn article — Conversation memory (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-conversation-memory-1200.png` (1200×627)

**Feed ad image:** `assets/neural-junkie-conversation-memory-ad-1080.png` (1080×1080)

**Regenerate images:** `./scripts/compose-conversation-memory-article.sh`

**Suggested title (pick one):**
- Your Agents Forget. Neural Junkie Remembers — Without Sending the Whole Transcript.
- Working Memory vs Long-Term Memory in Local Multi-Agent AI
- How We Gave Neural Junkie Conversation Memory (No Vector DB Required)

**Feed post teaser:**
> Neural Junkie agents used to forget anything outside the last few messages. Conversation memory indexes your full channel history and collab artifacts locally, then pulls back only what's relevant to your latest question — no cloud vector DB, no stuffing the whole transcript into context.

**Hashtags:** `#AI #LocalAI #MultiAgent #DeveloperTools #OpenSource #Ollama #RAG`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

---

## Ad / feed copy variants

**Variant A — problem/solution (feed post with ad image):**
> Long collab or DM thread? Your agent only sees the last handful of messages. Neural Junkie now indexes the full conversation locally and retrieves relevant past turns when you need them — "what did we agree on the API schema?" actually works. No Chroma. No cloud. Ollama embeddings + SQLite on your machine.

**Variant B — collab hook:**
> `/collaborate` sessions run for hours. Plans change. Findings land in `collabs/<id>/findings.md`. Conversation memory pulls that context back when a later task references an earlier decision — without replaying 200 messages into the prompt.

**Variant C — technical audience (comment or second post):**
> We didn't add a vector DB service. Messages persist in `messages.db`; chunks + embeddings go in `memory.db`; retrieval is scoped to channel/collab and injected as `=== RELEVANT PAST CONTEXT ===` at prompt build. Complements tail history and session summary — doesn't replace them.

---

## PASTE START

You ask a follow-up in a long DM or collab channel: *"What did we decide about the auth middleware?"*

The agent answers like the earlier thread never happened.

That isn't a model quality problem. It's a **context window budget** problem — and every multi-agent desktop hub hits it.

**Neural Junkie** is my personal open-source hub for running AI specialists on your hardware. Agents only received the last few channel messages plus a short rolling summary. Everything else lived in SQLite but never came back at query time.

Conversation memory fixes that — without sending the whole transcript to the LLM.

## Three layers of memory

Every turn still uses **working memory**:

1. **Tail history** — the last 2–10 messages for conversational continuity
2. **Session summary** — a ~2KB rolling compression of recent turns (on DMs and public channels)

The new layer is **long-term memory**:

3. **Conversation memory** — embed the user's latest question, search indexed past messages and collab markdown, inject the top matches as `=== RELEVANT PAST CONTEXT ===`

You don't widen the window. You **retrieve on demand**.

## What gets indexed

On write, the hub chunks and embeds:

- **Persisted chat messages** — the same content that would appear in LLM history (noise filtered out)
- **Collab artifacts** — `plan.md`, `planning-summary.md`, `session-summary.md`, `findings.md`, and other `collabs/<id>/*.md` deliverables

Collab channels are the biggest win: they never had session summaries. A three-hour `/collaborate` run could lose institutional decisions by message 50. Now those decisions stay searchable.

## Local-first — no vector DB service

I didn't add Chroma or Qdrant. The stack reuses what Neural Junkie already runs:

- **Ollama** `nomic-embed-text` for vectors
- **`memory.db`** — SQLite chunk store beside `messages.db`
- **Brute-force cosine + keyword prefilter** — fine for desktop-scale history

Default **on** when embed is available. Toggle in **Settings → AI & providers → Conversation memory**.

Clear channel history clears the memory index for that channel too.

## How retrieval stays safe

Retrieval is **scoped**:

- Message chunks: same channel only
- Collab artifacts: same collaboration id
- Chunks already in tail history: excluded (no duplication)

Budget cap ~1.5KB injected — enough for a few relevant excerpts, not a transcript dump.

Debug endpoints: `GET /api/memory/stats` and `GET /api/memory/query` for Pack dev and troubleshooting.

## Try it

```bash
make pull-models   # includes nomic-embed-text
make start-all
```

Download: https://github.com/camronwood/neural-junkie/releases/latest

Enable **Retrieve relevant past messages** under Settings if you turned it off.

Long thread smoke test:

```bash
./scripts/test-conversation-memory.sh
```

If you run long collabs or DMs and hit a retrieval edge case — wrong channel scope, missing collab artifact — GitHub issues welcome.

Camron Wood — Neural Junkie (personal project)

## PASTE END
