# LinkedIn article — v1.2.0-beta.5 release (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-beta5-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-beta5-article.sh`

**Suggested title (pick one):**

- v1.2.0-beta.5: The Release Where the Loops Close
- What Ships in Neural Junkie v1.2.0-beta.5
- Runbooks, Routing Trace, and Collaboration You Can Ship

**Feed post teaser:**

> v1.2.0-beta.5 bundles the loop-stack vision into one download: ReAct tools on Gemma, per-turn routing transparency, replayable runbooks, multi-repo workspace scope, collab hardening from live scenario gates, LoRA v2 specialists, and the release engineering that keeps betas honest.

**Hashtags:** `#AI #LocalAI #MultiAgent #OpenSource #DeveloperTools #AgenticCoding`

**Link:** [https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.5](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.5)

**Website:** [https://camronwood.github.io/neural-junkie/articles/beta-5.html](https://camronwood.github.io/neural-junkie/articles/beta-5.html)

**Download CTA:** [https://camronwood.github.io/neural-junkie/download.html](https://camronwood.github.io/neural-junkie/download.html)

**Suggested post date:** Same week as the `v1.2.0-beta.5` GitHub release tag

**Related articles:** [Loop stack](LOOP-STACK-LINKEDIN.md) · [ReAct tools](REACT-TOOLS-LINKEDIN.md) · [Collaboration](COLLABORATION-LINKEDIN.md) · [Personal learning](PERSONAL-LEARNING-LINKEDIN.md)

---



## PASTE START

**Neural Junkie v1.2.0-beta.5** is the open-beta release we have been building toward since beta.4: not one headline feature, but a stack of closed loops — repair, routing, collaboration, repeatable operations, and the test harness that proves they work on real local models.

If beta.4 was *Agent Runtime reliability* (fix loops, image gen, music pack, turn telemetry), beta.5 is what happens when those foundations meet **operational maturity**: runbooks you can replay, routing you can audit, collab you can regression-test, and specialists you can train without losing control.

Download: macOS, Windows, Linux — same local-first desktop app, same MIT-licensed hub.

## The headline in one paragraph

You get **ReAct MCP on models that lack native tool calling**, **per-turn routing trace** on every chat and collab message, **Runbooks v2** (persisted definitions + run history), **multi-repo workspace scope** via project sets, **collaboration hardening** from weeks of live scenario gates, **LoRA v2 compound specialists**, **personal learning** with explicit consent, **Slack setup diagnostics**, and **layer-gate / fix-loop / test-growth** automation so the next beta does not ship on vibes.

## 1. ReAct tools — one brain, not two models in RAM

Beta.4 gave you turn telemetry. Beta.5 completes a routing tier that beta users asked for loudly: **run MCP tools on the chat model you picked**, even when Ollama says that model has no native `tools` capability.

**Gemma 3 12B** is the first allowlisted tag. The hub wraps the provider in a ReAct loop with a strict contract:

```text
<tool_call>{"name":"read_file","arguments":{"path":"src/App.tsx"}}</tool_call>
```

Native tool calling when available → ReAct on the same model → Qwen swap fallback when parsing fails or the iteration cap hits. The typing indicator tells you which path ran: `(react_tools)` or `(react_fallback_swap)`.

That matters because "swap in Qwen for tools" worked — but it split persona, doubled RAM pressure, and hid the story in routing telemetry. Now you can keep one mid-tier model for chat *and* light MCP without pretending every Ollama tag is tool-native.

See the deep dive: [Gemma Can't Call Tools. We Taught It Anyway.](https://camronwood.github.io/neural-junkie/articles/react-tools.html)

## 2. Per-turn routing trace — stop guessing which model ran

Every substantive turn now carries **routing metadata** you can inspect without reading hub logs:

- Model tier and provider
- Retrieval mode (memory, codebase, none)
- Composer mode and context scope
- Implementation session state when relevant

In the desktop app: **routing badges** on messages, a live **Turn telemetry drawer** during generation, and a post-hoc **Routing trace panel** for debugging "why did that reply feel wrong?"

This is the user-facing half of the inference layer story — decide before you generate, then **show the decision on the message**.

## 3. Runbooks v2 — from one-off collabs to repeatable SOPs

Collaboration v1 was powerful and ephemeral: draw a DAG, approve, execute, done. Runbooks v2 adds **ownership**:


| Concept               | What it means                                                                            |
| --------------------- | ---------------------------------------------------------------------------------------- |
| **RunbookDefinition** | Named, versioned workflow with tasks, graph, policies, and inputs                        |
| **RunExecution**      | One run — still a collaboration under the hood, with run metadata                        |
| **Connector profile** | Named Slack/webhook/HTTP credentials referenced by action tasks — never embedded in JSON |


**Library UI:** save drafts, browse bundled + pack + user definitions, start runs with input forms, view run history.

**Resolution order:** user library → enabled pack runbooks → collab assets → repo templates.

Docs: [RUNBOOKS_V2.md](https://github.com/camronwood/neural-junkie/blob/main/docs/RUNBOOKS_V2.md)

## 4. Multi-repo workspace scope — ambient context without chaos

Real work spans repos. Beta.5 adds **project sets**: group workspace roots, expose a **workspace scope chip** in the desktop UI, and route **cross-repo hints** into agent context when your question touches linked codebases.

Scenario coverage includes ambient scope regression — the harness checks that agents respect boundaries instead of grep-bombing every checkout on disk.

Connection settings got a proper home: hub URL, server/network, automation, and connectors tabs — runtime config you can change without hand-editing JSON.

## 5. Collaboration hardening — live gates, not demo magic

Multi-agent collab is easy to demo and brutal to ship. Beta.5 includes weeks of **collab-full layer-gate** work:

- **Plan parser fix** — use the newest agent turn's task block instead of unioning every replan (stops 6–7 task lists from stacked discussions)
- **Planning recap timeout** — recap fallback 240s → 90s so scenarios get `complete` or `failed` before client timeouts
- **Planning discussion watchdog** — advance timed-out discussions instead of silent stalls; scaled timeouts for short collabs
- **Turn handoff retries** — faster retry cadence when an agent misses its slot
- **Participation enforcement** — `@mention` contracts and discussion readiness tracked per scenario
- **Approve-plan HTTP timeout** — harness/client 60s → 180s (client timeouts were masquerading as hub failures)

The full 15-scenario collab sweep is not fully green yet — local LLM flakes are real — but participation, recap, and harness reliability moved measurably (1 → 5 passing scenarios in the best gate iteration). We are not weakening assertions to greenwash flakes; we are fixing orchestration first.

Read the problem framing: [Multi-Agent Collaboration Is Easy to Demo. Hard to Ship.](https://camronwood.github.io/neural-junkie/articles/collaboration.html)

## 6. LoRA v2 + personal learning — specialists with consent

**LoRA v2** completes the compound specialist lifecycle: import rows, review training JSONL, bootstrap repo indexes, compose Ollama tags from one base + adapters — across chat, collab, IDE, and pack surfaces.

**Personal learning** stores only what you **explicitly confirm**, scoped per expert, globally, or per collaboration. Retrieved locally by embeddings; optionally exported into LoRA training rows. Nothing silently trains on your typos.

Together: context you curate *and* weights that lean inference toward your domain — without a cloud fine-tuning bill.

## 7. Slack — setup you can diagnose

Slack integration got a **setup checklist** in settings, **smoke and diagnose** endpoints/tests, and clearer handler errors when OAuth or channel routing is misconfigured.

If you run agents from your phone while the hive works locally, beta.5 is easier to trust — you can see *why* a relay failed instead of a silent inbox.

## 8. Release engineering — how we ship the next beta

User-facing features are half the story. Beta.5 also ships the **machinery**:

```bash
make layer-gate LAYER=collab-full    # live scenario sweep
make layer-fix-loop LAYER=chat       # gate → Cursor agent → verify → commit
make test-growth-loop                # discover gaps → strengthen tests
```

Layer gates boot Ollama, warm models, and restart the regression hub automatically. Fix loops isolate agent edits in git worktrees. Test-growth loops rank coverage candidates and add tests without greenwashing flakes.

This is how we plan to cut stable scope: **one layer at a time**, with artifacts in `docs/testing/`, not a single 30-hour prayer run.

## What carried forward from beta.4

Still in the box from the previous tag:

- **NJ Fix Loop** — command circuit breaker, boot-fix grounding, guaranteed session outcomes
- **Ollama image generation** — hub store, agent tools, settings panel
- **Music creation v1.0.2** — ACE-Step variants and inference tuning
- **Turn telemetry drawer** foundation — expanded in beta.5 with routing trace MVP
- **IDE v4** — Monaco LSP, remote SSH, dev containers (unchanged baseline)



## Try beta.5

1. Download the installer for your platform from [GitHub Releases](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.5) or the [downloads page](https://camronwood.github.io/neural-junkie/download.html).
2. Pull a local model (`gemma3:12b` if you want ReAct tools on one brain).
3. Open **Settings → Connection** — confirm hub URL and providers.
4. Send a chat message — open the **routing trace** on the reply.
5. Build a runbook in the **RB** builder — save to library, start a run.
6. Run `/collaborate @Agent1 @Agent2 goal` — watch planning discussion and task DAG execution.

**Neural Junkie** is local-first, open source, and built for specialists that share tools, approvals, and routing policy — not one monolithic planner guessing in a vacuum.

Repo: [https://github.com/camronwood/neural-junkie](https://github.com/camronwood/neural-junkie)

Release notes: [https://camronwood.github.io/neural-junkie/release-notes.html](https://camronwood.github.io/neural-junkie/release-notes.html)

## PASTE END

