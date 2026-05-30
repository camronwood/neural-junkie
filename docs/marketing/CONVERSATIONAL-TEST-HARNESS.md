# LinkedIn article — Conversational & collaboration test harness (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**Cover image:** `assets/neural-junkie-test-harness-ad-1200.png` (1200×627 — LinkedIn article header)

**Regenerate cover:**

```bash
chmod +x ./scripts/compose-test-harness-article.sh
./scripts/compose-test-harness-article.sh
./scripts/sync-gallery.sh   # optional — copies to docs/media/gallery/ads/
```

**Suggested title (pick one):**
- You Can’t Unit Test a Conversation. So I Built This Instead.
- How I Test Multi-Agent Chat Without Clicking Through the UI
- Three Layers for Testing AI Collaboration That Actually Holds Up

**LinkedIn publish checklist**

1. **New article** → paste title from list above.
2. **Cover image** → upload `assets/neural-junkie-test-harness-ad-1200.png`.
3. **Body** → paste everything below the second `---` divider (starts with `# You Can’t Unit Test…`).
4. **Optional feed post** (when sharing the article link):

> Prompts and models change weekly. Click-testing a multi-agent hub doesn’t scale. Here’s the three-layer harness I built for Neural Junkie — deterministic Go tests for the orchestrator, CI smoke for the pipeline, and JSON scenarios with real Ollama agents for conversation quality.

5. **Hashtags (end of feed post):** `#AI #SoftwareEngineering #Testing #DeveloperTools #OpenSource #MultiAgent`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest  
**Technical docs:** [CHAT_SCENARIOS.md](../CHAT_SCENARIOS.md) · [COLLABORATION.md](../COLLABORATION.md) § Live scenario harness

---

## Article body (paste below the line)

---

# You Can’t Unit Test a Conversation. So I Built This Instead.

Neural Junkie is a desktop hub where specialists you control can chat, review code, and run structured multi-agent collaborations. Getting the **orchestrator** right — phases, turn-taking, task graphs, workspace gates — is hard enough. Getting **conversation quality** right on top of that is a different problem entirely.

A greeting shouldn’t dump MCP tool blocks. A “thanks, that’s all” should close cleanly instead of re-explaining. A `/collaborate` planning thread should reach review with structured tasks, not infinite grounding spam. And when execution finishes, the file on disk should reference `core/sample/main.go`, not a hallucinated `index.js`.

You can’t capture that with a single `go test` that mocks the LLM. You also can’t rely on manually clicking through the desktop app every time you tweak a prompt. So I built a **three-layer test harness** — and split **chat** from **collaboration** because they fail in different ways.

---

## The problem: two different things to test

When people say “test the agents,” they usually mean one of two things:

1. **Does the hub enforce the rules?**  
   Phase transitions, discussion budgets, DAG readiness, “only one executing collab per channel,” artifact versioning, workspace ack before dispatch. This should be **deterministic**. The LLM is irrelevant.

2. **Do real models behave well in conversation?**  
   Tone, closure, tool spam, plan shape, silent agents, grounded file content. This is **inherently flaky** — models change, prompts drift, Ollama load varies. You still need regression signal; you just can’t pretend it’s a unit test.

Mixing those concerns into one test suite gives you either false confidence or endless CI pain. The harness keeps them separate on purpose.

---

## Layer 1 — Deterministic Go tests (fast, CI always)

**What it covers:** orchestrator logic and the Conversation Context Stack router — not model output.

**Collaboration:** lifecycle transitions, turn-taking, consensus, task assignment, plan parsing, dependency cycles, artifact edits. See `test/collaboration_test.go`.

**Chat quality router:** table-driven cases in `internal/agent/chat_quality_router_test.go` — intent (`closure` / `casual` / `substantive` / `task`), conversation mode (`chat` / `code` / `collab`), tooling gates, history caps, persona tier. This is the **Layer A** foundation described in [CONTEXT_MODEL.md](../CONTEXT_MODEL.md).

```bash
make test-go    # entire module; chat router + collab tests included
```

These run on every change. They tell you the **machinery** still works. They do not tell you whether `@Assistant` will stay under 1200 characters on “hello.”

---

## Layer 2 — Collab smoke (CI-friendly pipeline proof)

**What it covers:** the same HTTP surface the desktop uses — `POST /api/send`, `GET /api/collaborations`, `POST /api/collaboration-workspace-ack` — through **planning → review → executing → cancel**, with one synthetic discussion turn standing in for agent chatter.

```bash
make collab-smoke
```

Implemented in `test/collab_smoke_api_test.go`. No Ollama, no live agents, no flakiness. It catches wiring regressions: “approve plan” no longer transitions phase, workspace ack endpoint broke, collaboration redirect missing from send response.

This is the thin path that proves **the pipeline still connects** after hub changes. It’s not conversation quality; it’s **integration smoke**.

---

## Layer 3 — Live JSON scenarios (real agents, local dev loop)

**What it covers:** multi-turn conversation and multi-agent collaboration with **real in-process agents** (typically local Ollama models) against a **live hub**.

Two runners, two scenario directories, same design language:

| Harness | Scenarios | Runner | Make target |
|---------|-----------|--------|-------------|
| **Chat** | `scenarios/chat/*.json` | `scripts/chat-scenarios.py` | `make chat-scenario SCENARIO=…` |
| **Collaboration** | `scenarios/collab/*.json` | `scripts/collab-scenarios.py` | `make collab-scenario SCENARIO=…` |
| **Learning / LoRA** | `scenarios/learning/*.json` | `scripts/learning-scenarios.py` | `make learning-scenario SCENARIO=…` |

**Prerequisites:** hub running (`make server` or `make gui`), agents online, models configured. For sweeps, start the hub with `NEURAL_JUNKIE_RATE_LIMIT=0` so scenario polling doesn’t hit HTTP 429.

These stay **local-only**. They need time, GPU, and model availability. CI keeps Layers 1 and 2 green; Layer 3 is where I iterate on prompts and agent behavior.

---

## Chat scenarios: conversation quality without collab noise

Chat scenarios test **1:1 and channel chat** — greetings, closure, opinion questions, mode flips from casual to code review, DMs via the real channel API.

Example: `greeting-chat-mode` sends “hello” in chat mode and asserts the reply has no MCP blocks, no `FILE_CHANGE`, no kubectl, and stays under a character cap:

```bash
make chat-scenario SCENARIO=greeting-chat-mode
```

Other scenarios cover:

- **`thanks-closure`** — substantive answer, then “ok thanks” → canned closure  
- **`already-said-closure`** — “I know you said that already” → won’t-repeat closure  
- **`casual-opinion-chat`** — chat mode opinion without grounding spam  
- **`task-flip-review`** — hello, then a code review request; asserts debug intent is `task`  
- **`dm-greeting`** — same structural checks through a real DM channel

Steps are declarative: `send`, `wait_reply`, `assert_messages`, `assert_reply_count`, `assert_debug_context`. When a step fails, the runner prints pass/fail per step and dumps the last ~12 transcript lines. No guessing why `@Assistant` went silent.

Full reference: [CHAT_SCENARIOS.md](../CHAT_SCENARIOS.md).

---

## Collab scenarios: phases, plans, files on disk

Collaboration scenarios drive `/collaborate` end-to-end — or slice off just the planning phase for fast iteration.

**Fast planning loop** (`planning-two-agent`):

```bash
make collab-scenario SCENARIO=planning-two-agent
```

The scenario waits for `planning`, checks each agent spoke (`wait_discussion` with `min_per_agent`), waits for `reviewing`, asserts no `Grounding: I loaded` spam, and validates the parsed plan has at least two tasks with sane titles.

**Full execution loop** (`execute-deliverable`):

```bash
make collab-scenario SCENARIO=execute-deliverable PROFILE=fast
```

This runs against a **fixture repo** (`scenarios/fixtures/minimal-repo`). It approves the plan, acks the workspace, waits for task completion, hub-approves file changes, and asserts `findings.md` exists with content grounded in `core/sample/main.go` — while explicitly **rejecting** hallucinated paths like `index.js`.

Collab steps include everything chat has, plus collaboration-specific actions:

- `wait_phase`, `wait_discussion`, `wait_planning_recap`  
- `assert_plan` (min tasks, title patterns)  
- `approve_plan`, `workspace_ack`  
- `wait_tasks`, `approve_file_changes`, `assert_files`

**Profiles** swap agent rosters without rewriting scenarios: `PROFILE=fast` uses lighter 7b agents; `PROFILE=realistic` uses heavier architect/backend pairs. **`make collab-scenario-matrix`** sweeps profiles and round budgets before/after prompt changes.

Other notable scenarios:

- **`multi-collab-isolation`** — planning works while another collab executes elsewhere  
- **`resource-api-schema-planning`** — production-shaped goal with explicit agent lanes  
- **`delivery-sandbox-auto-ack`** — sandbox workspace auto-ack on approve  

Full reference: [COLLABORATION.md](../COLLABORATION.md) § Live scenario harness.

---

## What makes the harness useful when something breaks

The runners are built for **diagnosis**, not just pass/fail.

**Chat failures** dump recent transcript lines. Common causes: agent offline, slow model timeout, rate limit (restart hub with `NEURAL_JUNKIE_RATE_LIMIT=0`), missing `NEURAL_JUNKIE_DEBUG=1` for debug-context assertions.

**Collab failures** add a **discussion diagnosis** block: per-agent message counts, who stayed silent, handoff counts. That’s how I fix “Gemini never talked” or “plan never reached reviewing” without spelunking logs for an hour.

**`wait_discussion`** is the collab-specific insight. Orchestrator tests prove turn-taking rules exist; live scenarios prove agents actually **use** their turns.

**`KEEP=1`** leaves the collab or channel history intact for manual inspection:

```bash
make collab-scenario SCENARIO=planning-two-agent KEEP=1 VERBOSE=1
make debug-collab COLAB=<id8> LIVE=1
```

When a scenario fails, it becomes the next scenario I add or tighten. That’s the dev loop.

---

## Principles I keep coming back to

1. **Test the orchestrator in code; test the conversation in scenarios.** Don’t assert LLM prose in Go table tests. Don’t skip lifecycle tests because “the agents seemed fine yesterday.”

2. **Separate chat from collab.** Chat mode closure and collab planning recap are different failure modes. One runner per domain keeps scenarios readable.

3. **Assert structure, not poetry.** `min_tasks: 2`, `none_match: ["MCP"]`, `any_match: ["main\\.go"]` — pattern checks that survive model upgrades better than exact string equality.

4. **Ground scenarios in fixture repos.** Execution tests that only check chat markdown lie. Files on disk with real path constraints catch hallucinated project structure.

5. **CI for speed; live scenarios for truth.** Layers 1–2 gate merges. Layer 3 gates releases and prompt refactors.

6. **Failures become scenarios.** Every production “collab hung in review” or “agent dumped tools on hello” gets encoded as JSON so it doesn’t come back.

---

## If you’re building something similar

You don’t need Neural Junkie to steal the pattern:

- **Layer 1:** pure logic your product owns (state machine, router, caps) — table tests, no model.  
- **Layer 2:** HTTP/API smoke through the same paths your UI uses — synthetic events where agents would talk.  
- **Layer 3:** declarative JSON scenarios against a live stack — `send` / `wait` / `assert` steps, transcript dumps on failure, optional matrix sweeps.

The JSON scenario format is intentionally boring. Boring is good. When a prompt change breaks planning quality, you want a one-line `make collab-scenario` failure, not a vague “felt worse in manual testing.”

---

## Try it

Neural Junkie is a personal open-source project (macOS, Windows, Linux).

**Download:** https://github.com/camronwood/neural-junkie/releases/latest  

**Run the harness:**

```bash
make server                                    # or make gui
make test-go                                   # Layer 1
make collab-smoke                              # Layer 2
make chat-scenario SCENARIO=greeting-chat-mode # Layer 3 — chat
make collab-scenario SCENARIO=planning-two-agent # Layer 3 — collab
```

**Docs:** [CHAT_SCENARIOS.md](../CHAT_SCENARIOS.md) · [COLLABORATION.md](../COLLABORATION.md) · [CONTEXT_MODEL.md](../CONTEXT_MODEL.md)

If you add a scenario or hit a failure mode the harness doesn’t cover yet, GitHub issues welcome — that’s how the JSON files grow.

---

*Camron Wood — Neural Junkie (personal project)*
