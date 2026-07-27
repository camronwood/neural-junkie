# LinkedIn article — The Composition Model (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/composition-model/creatives/neural-junkie-composition-model-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-composition-model-article.sh`

**Suggested title (pick one):**
- The Composition Model: Agents, Tools, and Runbooks You Can Actually Take With You
- Share an Agent Like a File: Neural Junkie's Composition Model
- Portable AI Work: Bundles, Grants, and Provenance Instead of a Hive Mind

**Feed post teaser:**
> Neural Junkie now treats agents, tools, and runbooks as portable, composable units instead of hub-locked state. Share Agent bundles knowledge you can hydrate anywhere. The MCP Tool Wizard grants a home-grown HTTP tool to one agent by name. Runbook definitions export/import as JSON, and every run can trace back to the definition and events that produced it.

**Hashtags:** `#AI #LocalAI #MultiAgent #DeveloperTools #OpenSource #LocalFirst #Architecture #MCP #Ollama`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/composition-model.html

**Download CTA:** https://camronwood.github.io/neural-junkie/download.html

**Related articles:** [Modular AI composition](../modular-ai/MODULAR-AI-COMPOSITION-LINKEDIN.md) · [The Hub](../hub/HUB-LINKEDIN.md) · [Context stack](../context-stack/CONTEXT-STACK-LINKEDIN.md) · [Semantic turn routing](../semantic-turn-routing/SEMANTIC-TURN-ROUTING-LINKEDIN.md)

---

## Ad / feed copy variants

**Variant A — problem/solution:**
> You built a great custom agent. Now a coworker wants it on their machine, without your absolute repo path baked in. Neural Junkie's Share Agent packages knowledge, custom rules, learnings, and LoRA metadata into one file — hydrate, don't re-index.

**Variant B — architecture hook:**
> Agents, tools, and runbooks all get the same treatment in Neural Junkie: a portable bundle, a grant model scoped by name, and a provenance trail back to the events that ran. That's the Composition Model.

**Variant C — ops hook:**
> "Which definition produced this run, and what actually happened?" Neural Junkie's runbook provenance API answers that in one call — definition, version, task counts, and the full event trace.

---

## PASTE START

We keep shipping features that solve the same underlying problem: **how do you take a piece of AI work — an agent, a tool, a workflow — out of the machine it was built on, without losing what makes it useful?**

Share Agent solves it for knowledge. The MCP Tool Wizard solves it for capability. Runbook export/import and provenance solve it for process. Individually they read like three features. Together they're one idea: the **Composition Model** — treat agents, tools, and runbooks as portable, composable units instead of hub-locked state.

## The bet in one sentence

**A unit of AI work should be exportable as a file, importable with explicit consent, and traceable back to what produced it — without requiring a shared filesystem, a shared hub, or a leap of faith.**

## Pillar 1: Share Agent — knowledge you can hydrate, not just re-index

Repo agents already supported MCP export/import, but import meant **re-indexing from the same absolute path** — useless the moment a friend or coworker doesn't have `/Users/you/dev/thing` on their machine. Learnings, custom rules, and LoRA metadata didn't travel with the bundle at all.

Share Agent changes the contract:

- **Hydrate, don't re-index** — `HydrateFromExport` rebuilds an agent's in-memory index directly from the resources embedded in the bundle. No disk scan, no path assumption.
- **Bring the whole agent, not just files** — custom-rules markdown and agent-scoped learnings are merged in on import (`Hub.ImportShareLearnings` dedupes by content hash so re-importing the same bundle twice doesn't double up).
- **LoRA travels as metadata** — when a specialist has an active adapter, its base and composed Ollama tags ride along in the bundle for the receiving hub to resolve.

```bash
# Export
POST /api/agents/{id}/share  →  share-agent-bundle.json

# Import with hydration (no local repo required)
POST /api/import  { "hydrate": true, "repository_path": "" }
```

In the desktop app, Agent Info now has a **Share** button instead of a bare export link, and Import Agent exposes the hydrate option directly — the file picker path and the "trust this machine's copy of the code" path are finally two different, explicit choices.

## Pillar 2: MCP Tool Wizard — grant a tool to one agent, not the world

Custom experts (`/create-expert`) were persona-and-rules only — no tool loop. If you wanted your homemade expert to fetch a public API, you were writing Go and a pack sidecar.

The wizard's first template is deliberately narrow: an **HTTP-fetch tool** — name, URL, method, headers, optional JSON-path extraction — granted to chosen agents **by display name**, because agent IDs regenerate across hub restarts but names don't.

```
POST /api/mcp/user-tools               create a tool
POST /api/mcp/user-tools/{id}/test     preview output before granting
POST /api/mcp/user-tools/{id}/grant    { "agent_name": "Widget Expert", "grant": true }
```

Grants are enforced through the same SSRF gate as `fetch_url`: no loopback, no private ranges, no non-HTTP schemes. Custom experts get a real in-process MCP server the moment a tool is granted — and it's empty overhead otherwise.

The second template ships alongside it, off by default: **external media tools** (`media_submit` / `media_status` / `media_fetch`) for wiring a granted agent to a third-party image/video/audio API. `BaseURL` defaults to empty — the tools simply don't attach until an operator configures a real endpoint. Both templates share one in-process MCP server per custom expert, so an agent with both kinds of grants gets a single unified tool surface instead of two.

## Pillar 3: Runbooks as portable, provenance-tracked definitions

Runbook definitions already lived in a versioned library. What was missing was **getting one out of your library and into someone else's, and knowing where a run actually came from** after the fact.

```
GET  /api/runbook-definitions/{id}/export        → DefinitionBundle (schema_version, exported_at, definition)
POST /api/runbook-definitions/import             → mints a fresh ID by default (no collisions)
POST /api/runbook-definitions/import?keep_id=true → round-trips the same ID, bumps the version
```

Provenance ties a run back to the definition and version that produced it, plus the full append-only event trace (phase transitions, task dispatch/completion/failure):

```
GET /api/runbook-runs/{collabID}/provenance
→ { run, definition, collaboration, events[] }
```

And because "is this runbook actually moving" is a different question than "what happened historically," there's a lightweight progress endpoint that reuses the same dependency-ready logic the dispatcher uses internally:

```
GET /api/runbook-runs/{collabID}/progress
→ { total_tasks, counts, percent_complete, queued_task_ids, in_progress_task_ids, blocked_task_ids }
```

## What ties it together

None of these three pillars share code by accident — they share a **shape**:

1. **A bundle** (JSON, self-describing, versioned) — the export
2. **A grant or explicit import action** — consent, not ambient trust
3. **A way to check what's actually running or where something came from** — provenance/progress, not vibes

Agents, tools, and runbooks all get exported the same way, granted the same way, and audited the same way. That consistency is the point: once you understand Share Agent's bundle-and-hydrate pattern, the runbook export/import API and the tool grant model look exactly like what you'd expect.

## Try it

Personal open-source project — macOS, Windows, Linux.

```bash
make pull-models
make start-all
```

Download: https://github.com/camronwood/neural-junkie/releases/latest

**Docs:**
- [MCP integration](https://github.com/camronwood/neural-junkie/blob/main/docs/MCP_INTEGRATION.md)
- [MCP exports](https://github.com/camronwood/neural-junkie/blob/main/docs/MCP_EXPORTS.md)
- [Runbooks v2](https://github.com/camronwood/neural-junkie/blob/main/docs/RUNBOOKS_V2.md)
- [Future enhancements](https://github.com/camronwood/neural-junkie/blob/main/docs/FUTURE_ENHANCEMENTS.md)

If a Share Agent bundle hydrates wrong, a granted tool trips the SSRF gate when it shouldn't, or a provenance call comes back empty for a run you know happened — GitHub issues welcome. That feedback becomes the next test case.

Camron Wood — Neural Junkie (personal project)

## PASTE END
