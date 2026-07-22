# LinkedIn article — Semantic turn routing (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/semantic-turn-routing/creatives/neural-junkie-semantic-turn-routing-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-semantic-turn-routing-article.sh`

**Suggested title (pick one):**
- One Decision Per Turn: Why We Killed Phrase Matching in Neural Junkie
- “Will Not Boot” Broke Our Router. Semantic Turn Routing Fixed It.
- Meaning Over Regex: Centralizing Intent in a Local Agent Hub

**Feed post teaser:**
> We had phrase lists for boot fixes, canvas artifacts, continuations, and code review — and they kept fighting each other. Neural Junkie now resolves one server-authoritative semantic decision per turn: meaning from a local classifier, permissions from deterministic policy. Here's the architecture.

**Hashtags:** `#AI #LocalAI #MultiAgent #DeveloperTools #OpenSource #LocalFirst #Architecture #Ollama`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/semantic-turn-routing.html

**Download CTA:** https://camronwood.github.io/neural-junkie/download.html

**Related articles:** [The Hub](../hub/HUB-LINKEDIN.md) · [Inference layer](../inference-layer/INFERENCE-LAYER-LINKEDIN.md) · [Loop stack](../loop-stack/LOOP-STACK-LINKEDIN.md) · [Conversation memory](../conversation-memory/CONVERSATION-MEMORY-LINKEDIN.md)

---

## Ad / feed copy variants

**Variant A — problem/solution:**
> Your agent hears “create a canvas” and opens a code file. Or “the app won’t start” and chats instead of diagnosing. That’s phrase matching. Neural Junkie now classifies meaning once at the hub, then policy decides action, recipient, retrieval, and write permission — locally.

**Variant B — architecture hook:**
> Desktop sends explicit mode and IDE context. The hub stamps one `TurnDecision`. Knowledge routing, model selection, trust, and the implementation gate all consume that decision. No second opinion from a regex list halfway down the pipeline.

**Variant C — local-first:**
> Classification runs on your utility model (Ollama / LM Studio), structured JSON only. Frontier models never get to decide what “intent” means for a turn. Ask and Plan still cannot mutate — policy owns that, not the classifier.

---

## PASTE START

You tell the agent: *“Something is wrong — the application exits before showing its interface. Diagnose it and sort it out.”*

Or: *“Create a Neural Canvas beside this conversation that explains the architecture.”*

Or: *“Proceed with that.”*

Three different meanings. Three different actions. Same messy language.

For a long time, Neural Junkie tried to catch those meanings with **distributed phrase lists** — on the desktop, in the agent, in knowledge routing, in trust signals, in implementation gates. Boot-fix regexes. Canvas detectors. Continuation affirmations. Code-review cues.

It worked until it didn’t.

“Will not boot” and “Neural Canvas” became the poster children: hard-coded phrases that misrouted turns, fought each other, and couldn’t survive a paraphrase. Every new product surface meant another list to keep in sync.

So we replaced the lists with a single rule:

**Resolve meaning once. Compile policy once. Stamp an immutable turn decision. Consume it everywhere.**

## The bet in one sentence

**The model describes requested meaning. Deterministic policy grants permission.**

A local classifier never gets to authorize writes, frontier access, destructive ops, or auto-apply. Those stay in code you can audit.

## What broke

Phrase matching fails in predictable ways:

- **Paraphrase blindness** — “won’t start,” “never reaches the UI,” and “exits before the interface” are the same job; regexes only catch the phrases you remembered to write
- **Homonym collisions** — “canvas” as a durable chat artifact vs an HTML canvas component in source
- **Split brain** — desktop picks an agent, server re-infers action, knowledge router rescans the raw text, trust layer runs its own regexes
- **Unsafe authority creep** — if a phrase list can flip `implementation_session`, a clever wording can ask for more power than the UI mode allowed

We didn’t need a smarter regex. We needed a **typed contract**.

## One typed decision per turn

At hub ingress — after mentions and DMs are handled — Neural Junkie now builds bounded `TurnFeatures` and resolves a versioned `TurnDecision`:

- **Interaction** — question, task, continuation, correction, closure, casual
- **Action** — answer, inspect, plan, debug, edit, run, continue, artifact, image, ask-user
- **Domain / recipient** — which specialist should own the turn when nobody was @mentioned
- **Retrieval** — memory, codebase, graph, prior reference, collab artifact
- **Mutation** — none, external, or workspace
- **Provenance** — structural directive, local classifier, or safe fallback

That decision is stamped on the message. Client-authored decisions are stripped. The turn pipeline, knowledge plan, model routing, trust classification, artifact/image gates, and implementation session all **prefer the stamp** over re-reading the user’s sentence.

## How resolution works

```
Desktop → explicit composer mode, trust, IDE context, mentions, reply IDs
    ↓
Hub ingress → structural features
    ↓
Fully determined? ──yes──► Deterministic policy
    │
    no (semantic ambiguity)
    ↓
Local structured classifier (utility model, temp 0, JSON schema)
    ↓
Deterministic policy → TurnDecision
    ↓
Recipient · TurnGoal · KnowledgePlan · model route · evidence contract
```

Structural facts still win without calling a model: Ask/Plan/Export mode, slash commands, explicit mentions, typed reply/action IDs, pending continuation targets, workspace presence, and capability flags.

When the text is ambiguous, a **local** utility model classifies into the schema. One repair attempt on malformed JSON. Low confidence abstains into a safe fallback. Frontier models are never used for turn classification.

## Policy owns the dangerous bits

Examples that matter in practice:

- Ask and Plan force `mutation: none` even if the classifier suggests edit
- Workspace mutation requires both capability flags and an actual workspace
- Debug / edit / run / continue with a workspace get codebase retrieval without re-scanning for “look in the repo”
- Continuation needs a durable pending action ID — “go ahead” alone is not enough to invent one

The classifier can say *what the user asked for*. Policy decides *what the runtime is allowed to do*.

## Desktop becomes a thin client

The send path no longer invents implementation sessions from “please implement…” or routes boot fixes to FrontendEngineer because a phrase matched.

It transports:

- Composer mode and trust preference
- Active editor / workspace context
- Attachments and `@codebase` syntax
- Explicit mentions and reply/action IDs
- A versioned `turn_governance` envelope with explicit provenance

Semantic authority lives on the server. That matches the hub thesis from the earlier architecture article: **orchestration belongs in a server you control**.

## What we kept (and what we quarantined)

We kept narrow deterministic recognizers for **syntax and safety**: slash commands, mentions, paths, structured diagnostics, destructive command checks, and evidence-claim validation.

Legacy phrase lists still exist behind an emergency rollback flag and as fallbacks when no decision is stamped. A repository guard test fails if new semantic phrase regexes appear outside the classifier package or the explicit quarantine list.

Direct cutover — no shadow dual-run of live turns. Telemetry records schema version, source, classifier model, latency, confidence, abstention, and policy overrides without stuffing extra raw user text into the log.

## Try the idea in one sentence

If two wordings mean the same job, they should produce the same `TurnDecision`.

If a wording sounds like a write but Ask mode is on, policy must still refuse the write.

That’s the upgrade: **meaning over phrases, policy over vibes**.

## PASTE END
