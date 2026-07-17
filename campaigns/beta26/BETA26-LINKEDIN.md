# LinkedIn article — v1.2.0-beta.6 release (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/beta26/creatives/neural-junkie-beta6-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-beta6-article.sh`

**Suggested title (pick one):**

- v1.2.0-beta.6: A Memory of Its Own Code
- What Ships in Neural Junkie v1.2.0-beta.6
- Knowledge Graph, Model Arena, and Streams That Fire Runbooks

**Feed post teaser:**

> v1.2.0-beta.6 gives the workstation a memory of its own code: a native repository knowledge graph, Model Arena matches, MQTT/Kafka stream subscriptions, PrismML Bonsai 27B, Room Chat, Homebrew installs, and the polish beta users asked for — pack toolbar chips and workspace image previews that finally just work.

**Hashtags:** `#AI #LocalAI #MultiAgent #OpenSource #DeveloperTools #KnowledgeGraph`

**Link:** [https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.6](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.6)

**Website:** [https://camronwood.github.io/neural-junkie/articles/beta-6.html](https://camronwood.github.io/neural-junkie/articles/beta-6.html)

**Download CTA:** [https://camronwood.github.io/neural-junkie/download.html](https://camronwood.github.io/neural-junkie/download.html)

**Suggested post date:** Same week as the `v1.2.0-beta.6` GitHub release tag

**Related articles:** [Hub architecture](../hub/HUB-LINKEDIN.md) · [Stream subscriptions](../stream-subscriptions/STREAM-SUBSCRIPTIONS-LINKEDIN.md) · [Loop stack](../loop-stack/LOOP-STACK-LINKEDIN.md) · [beta.5](../beta25/BETA25-LINKEDIN.md)

---



## PASTE START

**Neural Junkie v1.2.0-beta.6** is the open-beta release where the workstation stops treating your repo as a bag of files and starts treating it as a **graph of relationships** you can explore, query, and ground agent turns on.

If beta.5 closed the operational loops — ReAct tools, routing trace, Runbooks v2, collab hardening — beta.6 is what happens when that stack meets **code memory**: a native knowledge graph, head-to-head Model Arena, event-driven stream subscriptions, and distribution polish (Homebrew on macOS and Linux) so beta users can install and update without hunting GitHub assets.

Download: macOS, Windows, Linux — same local-first desktop app, same MIT-licensed hub. Prefer Homebrew? `brew tap camronwood/tap` then install the cask (macOS) or formula (Linux).

## The headline in one paragraph

You get a **repository knowledge graph** (index, APIs, IDE workbench, and a `code_graph` agent route), **Model Arena** with reliable matches and inference usage telemetry, **MQTT/Kafka stream subscriptions** that fire runbooks or channels, **PrismML Bonsai 27B** in the model library, **Room Chat** for ephemeral LAN rooms, **Homebrew distribution**, domain **pack v2 plumbing**, and the everyday fixes that matter when you live in the app: **workspace image previews** and **pack toolbar chips that appear on enable**.

## 1. Repository knowledge graph — memory of its own code

Beta users kept asking the same class of question: *how does this relate to that?* *who calls this?* *what path connects A to B?*

Beta.6 ships a native graph under the hood:

- Build and store a code graph for the active workspace
- Query subgraphs, paths, and node explain
- Open a **Knowledge Graph workbench** in the IDE (toolbar chip, Files panel, or `/nj-open-knowledge-graph`)
- Route agent turns through `code_graph` when the question is about relationships, not just chunks

This is not a separate SaaS product bolted on. It is part of the same hub that already indexes your repo for `@codebase` — now with structure you can see and navigate.

Deep dive: [Knowledge graph docs](https://github.com/camronwood/neural-junkie/blob/main/docs/KNOWLEDGE_GRAPH.md) · [feature guide](https://camronwood.github.io/neural-junkie/features/knowledge-graph.html)

## 2. Model Arena — put two models in the ring

Packs are how Neural Junkie grows without bloating the core. **Model Arena** is the beta.6 pack headline for people who care about model quality in the open:

- Head-to-head matches with observable status
- Hardened move handling so sessions do not stall mid-match
- Inference usage telemetry so you can see what a run actually cost in tokens and time

It is the same local-first philosophy: your hub, your models, your ring.

## 3. Stream subscriptions — MQTT and Kafka that fire work

Chatbots that sit on a topic and narrate every message are the wrong abstraction. Beta.6 continues the stream-subscription story: long-lived MQTT/Kafka listeners that **match messages and do something useful** — run a runbook, post into a hub channel, or hit a webhook.

Settings UI included. Your ops bus stays an ops bus; Neural Junkie becomes the agent that reacts when the signal matches.

## 4. Models and rooms — Bonsai, Room Chat, Homebrew

Three distribution and access wins for beta users:

- **PrismML Bonsai 27B** lands in the model library with `mmproj` support, prompted Ollama updater guidance, and min-version gates so you are not guessing which runtime works.
- **Room Chat** pack: ephemeral LAN rooms hosted on a host hub — bring a room online without standing up a permanent team channel.
- **Homebrew**: `brew install --cask neural-junkie` on macOS and `brew install neural-junkie` on Linux from `camronwood/tap`, with automated tap bumps on release.

## 5. Packs and polish that show up immediately

Domain pack **v2 plumbing** continues for CAD, AWS, and software-development. Platform foundations add turn pipeline, span tracing, and durable collab state.

And the two bugs that make a beta feel unfinished:

- **Workspace image previews** — PNGs in your project open in the editor again (hub data URLs + wider Tauri asset scope).
- **Pack toolbar chips** — enable a pack with a toolbar capability and the chip appears **without** needing an IDE layout refresh.

Small? Yes. Visible every day? Also yes.

## What we are still honest about

Open beta means we ship progress and name the gaps. The live **parity gate** (`test-parity-stable-restart`) is still failing and is tracked for a follow-up beta. Release CI still runs the unit and race gates that block installers; the live-model parity suite is the next bar we intend to clear.

Known issues page stays the source of truth for what is still wrong on purpose.

## Get it

- Releases: [v1.2.0-beta.6](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.6)
- Download page: [camronwood.github.io/neural-junkie/download.html](https://camronwood.github.io/neural-junkie/download.html)
- Homebrew: `brew tap camronwood/tap` then install cask or formula
- Changelog: [docs/CHANGELOG.md](https://github.com/camronwood/neural-junkie/blob/main/docs/CHANGELOG.md)

Star the repo if this is useful. File issues if something breaks. Beta.6 is for people who want a local multi-agent workstation that can **see** how code connects — not just retrieve the nearest chunk.

## PASTE END
