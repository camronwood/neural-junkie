# Testing

## North star: Cursor-like agent behavior

Neural Junkie tests are the contract for **Cursor-like parity** on this platform: workspace-aware replies, substantive coding answers, every collab participant contributing when invited, deliverables on real paths, and the same agent stack in chat, IDE layout, collab, and Slack.

When a live scenario fails, triage **product/hub/agent behavior first**, harness second. Do not weaken assertions to greenwash flakes.

| Behavior | Implementation | Verified by |
|----------|----------------|-------------|
| Workspace / open-file awareness | [CONTEXT_MODEL.md](CONTEXT_MODEL.md), [IDE_V3.md](IDE_V3.md) | `make chat-scenarios-debug` (`assert_debug_context`); `dm-backend-workspace`, `public-backend-theme-workspace` |
| Substantive coding replies | Intent routing, specialist prompts | `make chat-scenarios-regression`, DM task scenarios |
| `@codebase` semantic search | `internal/codeindex`, `POST /api/repo/search/semantic` | `dm-backend-codebase-semantic` |
| IDE routing from open file | `ide_route_agent_type` metadata | `dm-ide-route-backend` |
| Workspace MCP tools (read/grep/glob) | `internal/mcp/workspace` | manual IDE Agent mode |
| Everyone contributes in collab | Discussion turns, `@mention` out-of-turn | `make collab-scenarios-all`; `wait_discussion` + nudges |
| Deliverables on real paths | Plan parser, execution sandbox | Collab regression scenarios, Phoenix with real repo |
| Slack = same agent surface | Slack bridge → bound channel | `make slack-smoke`; optional `LIVE=1` |
| Cursor CLI on PATH | [CLI_AGENTS.md](CLI_AGENTS.md) | Optional manual `@Cursor` chat/collab (not CI) |
| **Phase 1 implement in repo** | [IMPLEMENTATION_SESSION.md](IMPLEMENTATION_SESSION.md) | `make implement-scenarios` (≥4/5 PASS) |
| Conversation routing + collab wiring | Agent intent/closure, hub DM/collab, desktop chat UI | `make test-conversation-contract` |

See also [CHAT_SCENARIOS.md](CHAT_SCENARIOS.md) and [COLLABORATION.md](COLLABORATION.md).

## Phase 1 acceptance (~80% Cursor “implement this feature”)

**Gate:** `make implement-scenarios` with a regression hub and Ollama tool model (`qwen3.5:9b` or Settings → Implementation tool model).

```bash
ollama serve
ollama pull qwen3.5:9b    # specialists, tool loop, and utility (OLLAMA_CODE_MODEL / OLLAMA_MODEL)
make server-regression         # terminal 1
make agents                    # terminal 1b — picks up env.local models
make implement-scenarios       # terminal 2 — need 7/7 PASS
make test-parity-stable        # optional — 3× sweeps at 7/7 under server-regression
```

Scenarios assert **files on disk** (not just reply text). See `scenarios/implement/*.json`.

**Deliverable contract (`expect_deliverables`):** File-producing implement and collab scenarios declare expected paths plus question-aligned quality bars (`for_question.any_match` / `none_match` / `contains_all`). Live runs with `llm_judge: true` use an **independent judge** — **cloud-first** (hub `@Gemini` by default), with **Ollama fallback** (`qwen2.5-coder:14b`) when quota or API errors occur (`NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA=1`). Set `NJ_DELIVERABLE_JUDGE_PROVIDER=ollama` for local-only judging. CI smoke enforces the JSON contract via `make test-scenario-assert`; `NJ_DELIVERABLE_JUDGE_SKIP=1` skips LLM judge during live runs.

**Stability gate (beta.24+):** archives logs to `docs/testing/parity-stable-*.log`:

```bash
make server-regression
make test-parity-stable              # 3× back-to-back (may OOM hub)
make test-parity-stable-restart      # 3× with hub restart between sweeps
```

**Regression bundle (beta.25+):** implement + chat + conversation in one run:

```bash
make server-regression
make test-regression-bundle
# log: docs/testing/regression-bundle-*.log
```

**Full test sweep (beta.26+):** CI smoke + live harness in one run; reviewable summary + full log:

```bash
make test-everything              # CI + live (hub + agents required)
make test-everything SKIP_LIVE=1  # CI/smoke only, no hub
make test-everything-full         # above + collab-scenarios-all (~1-3h extra)
# reports: docs/testing/test-everything-*.md (summary) + test-everything-*.log (full output)
```

**Release prep (one command):** `test-everything-full` + `test-parity-stable-restart` + `model-benchmark` with a unified report:

```bash
ollama serve
make server-regression    # terminal 1 (optional — release-prep can start/restart hub)
make release-prep VERBOSE=1
# unified: docs/testing/release-prep-*.md + release-prep-*.log
# child:   test-everything-*.md, parity-stable-restart-*.log, model-benchmark-*.{md,json,tsv}
```

`make release-prep` automatically:

- Sources `load-env.sh` (env.local + `.gemini-api-key` when present)
- Cloud-first deliverable judge (hub Gemini) with Ollama fallback and RPM pacing
- Verifies hub health + judge smoke (restarts regression hub once if needed)
- Runs `collab-preflight --require-gemini` before the long phases

Prerequisites: `ollama serve` + `ollama pull qwen2.5-coder:14b` (fallback judge). Optional: `.gemini-api-key` for cloud judge when quota allows.

**Deliverable judge:** tries hub Gemini first; on quota/API errors falls back to local Ollama so sweeps keep running.

Options: `SKIP_LIVE=1` (CI only), `NO_FULL=1` (skip collab-scenarios-all), `SKIP_BENCHMARK=1`, `SKIP_PARITY=1`, `NO_RESTART_HUB=1`, `BENCHMARK_SUITE=standard`, `BENCHMARK_MODELS='qwen3.5:9b,qwen2.5-coder:14b'`, `PULL=1`, `STOP_ON_FAIL=1`.

**Model benchmark:** compare top local coder models against the same scenarios — see [testing/MODEL_BENCHMARK.md](testing/MODEL_BENCHMARK.md):

```bash
make model-benchmark-list
make model-benchmark SUITE=quick MODELS='qwen2.5-coder:14b,qwen3.5:9b'
# reports: docs/testing/model-benchmark-*.md|.json|.tsv
```

**Manual spot-check (real app):** Share workspace on a React+Tailwind repo (e.g. dickory-docs with `.neural-junkie/rules.md`), Agent mode + `auto_apply_edits`, prompt: implement light/dark theme. Expect root `tailwind.config.js` with `darkMode`, `.tsx` paths only (no `.vue`), and honest session summary (`applied and verified` or `proposals submitted`).

## Test tiers

| Tier | When | Commands |
|------|------|----------|
| **CI** | Every push/PR to `main` | `make test-all` (GitHub Actions [`.github/workflows/test.yml`](../.github/workflows/test.yml)) |
| **Smoke** | Local dev, no LLM | `make test-conversation-contract`, `make collab-smoke`, `make test-collab-plan`, `make test-scenario-assert`, `make learning-lora-smoke`, `./scripts/mcp-smoke.sh`, `make slack-smoke` |
| **Live regression** | Pre-release / beta | Checklist below |

## CI vs live: what each tier proves

| Proven in CI (`make test-all`) | Proven only in live regression |
|-------------------------------|--------------------------------|
| Go + desktop unit tests | Multi-agent LLM discussion quality |
| Hub wiring, plan parser, collab API smoke | Full 15-scenario `collab-scenarios-all` sweep |
| Slack handler mocks (`slack-smoke`) | `chat-scenarios-regression` / `chat-scenarios-debug` |
| `collab-smoke`, `learning-lora-smoke` | `learning-scenarios`, file deliverables on disk |
| No Ollama, no 1–3h serial LLM work | Phoenix repo paths (`NEURAL_JUNKIE_SCENARIO_REPO`) |

Do **not** add `collab-scenarios-all` to CI. Use `make collab-preflight` before live sweeps.

Optional: GitHub Actions `workflow_dispatch` job `collab-preflight` (hub must be reachable from the runner — typically local pre-release only).

## Pre-release checklist

1. CI green on branch (`make test-all` locally if needed).
2. `ollama serve` and `ollama pull qwen3.5:9b` (specialists + tool loop; set `OLLAMA_CODE_MODEL=qwen3.5:9b` in `env.local`).
3. **Hub:** `make server-regression` — sets `NEURAL_JUNKIE_RATE_LIMIT=0` and `NEURAL_JUNKIE_DEBUG=1` on the **server process** (not only scenario clients). Never use `make start-all` for sweeps.
4. `make collab-preflight` — hub, Ollama, default agents; add `REQUIRE_GEMINI=1` when running `resource-api-schema-planning`.
5. **`make test-regression-bundle`** — implement (7/7) + `chat-scenarios-regression` + `conversation-scenarios-regression` (18 chat + 6 collab conversation scenarios); log under `docs/testing/regression-bundle-*.log`
6. Optional: **`make test-parity-stable-restart`** — 3× implement with hub restart between sweeps (avoids OOM on memory-limited hosts)
7. `make chat-scenarios-debug`
8. `make collab-scenarios-all` — 15 scenarios, serial, ~1–3h; archive log under `docs/testing/`.
9. `make learning-scenarios`
10. Optional: `collab-scenario-matrix`, `collab-routing-matrix`, Phoenix with `NEURAL_JUNKIE_SCENARIO_REPO=/path/to/clone`
11. Optional: `LIVE=1 make slack-smoke` (runs `scripts/slack-live-smoke.sh`)
12. Optional: `@Cursor` smoke when `agent` binary on PATH

Individual gates (same hub): `make implement-scenarios`, `make chat-scenarios-regression`, `make conversation-scenarios-regression`.

Quick reference: `make test-regression-live` prints the live steps.

### Hub vs client rate limit

HTTP 429 during scenario polling is controlled by the **hub** env var `NEURAL_JUNKIE_RATE_LIMIT` ([`internal/hub/ratelimit.go`](../internal/hub/ratelimit.go)). Makefile targets set `NEURAL_JUNKIE_RATE_LIMIT=0` on Python clients for consistency, but you must start the hub with `make server-regression` (or `NEURAL_JUNKIE_RATE_LIMIT=0` manually) before live sweeps.

### Collab preflight and sweep

```bash
ollama serve
make server-regression    # separate terminal
make collab-preflight     # REQUIRE_GEMINI=1 if running Gemini scenario
```

**Recommended:** one scenario at a time (advance only after PASS):

```bash
make collab-sweep-serial              # stop on first FAIL
make collab-scenario SCENARIO=<name> VERBOSE=1   # fix and re-run one
make collab-sweep-serial RESUME=1     # skip PASS rows in docs/testing/collab-matrix.tsv
```

Batch (all 15 serially, ~1–3h, no gate between failures):

```bash
make collab-scenarios-all 2>&1 | tee /tmp/nj-collab-sweep-$(date +%F).log
```

`PROFILE=fast` only substitutes agents when a scenario JSON **omits** the `agents` field; it does not shorten timeouts.

### Per-scenario agent requirements (all 15)

| Scenario | Required agents | Notes |
|----------|-----------------|-------|
| `planning-two-agent` | (profile default: 2+ online) | No pinned roster |
| `delivery-sandbox-auto-ack` | (profile default) | |
| `execute-deliverable` | (profile default) | |
| `document-findings-execution` | (profile default) | |
| `reject-collabs-subfolder` | (profile default) | |
| `solo-vs-collab-parity` | Assistant | Solo leg auto-approves file proposals |
| `multi-collab-isolation` | (profile default) | Blocker setup uses fast agents |
| `plan-findings-task-regression` | Assistant, BackendEngineer, SoftwareArchitect | |
| `plan-distinct-deliverables-same-agent` | SoftwareArchitect, BackendEngineer | |
| `plan-phoenix-combined-regression` | BackendEngineer, SoftwareArchitect, PlatformEngineer | `NEURAL_JUNKIE_SCENARIO_REPO` |
| `plan-dependency-prose-regression` | BackendEngineer, SoftwareArchitect, PlatformEngineer | `NEURAL_JUNKIE_SCENARIO_REPO` |
| `resource-api-schema-regression` | Assistant, BackendEngineer, FrontendEngineer | `NEURAL_JUNKIE_SCENARIO_REPO` |
| `resource-api-schema-planning` | Assistant, **Gemini**, PlatformEngineer | Gemini CLI agent |
| `phoenix-resource-api-e2e` | Assistant, SoftwareArchitect, BackendEngineer | `NEURAL_JUNKIE_SCENARIO_REPO` |
| `execution-no-stack-commands` | Assistant, PlatformEngineer | |

Sweep logs: [testing/collab-sweep-2026-06-02.md](testing/collab-sweep-2026-06-02.md).

### Triage playbook (live failures)

1. Re-run one scenario: `make collab-scenario SCENARIO=<name> VERBOSE=1`
2. Read harness diagnosis (printed on `wait_discussion` failure); includes `generation_error` posts when the model failed.
3. Hub log: `grep -E 'COLLABORATION TURN|Error generating response' /tmp/nj-hub.log | tail -40`
   - Many `Error generating response` + zero discussion → **Ollama down** (`ollama_down`), not harness.
   - `COLLABORATION TURN` without discussion → participation / `shouldRespond` (`silent_agent`).
   - Stuck in `planning` → `stuck_planning` (budget, consensus, or silent agents).
4. Live debug: `make debug-collab COLAB=<id8> LIVE=1`
5. Product-first: do not weaken `wait_discussion` assertions to greenwash flakes.

Latest archived sweep: see `docs/testing/collab-sweep-*.md`.

### `wait_discussion` flakes

A silent specialist (e.g. BackendEngineer never spoke) is a **participation defect**, not noise to ignore. Harness `retries` and `nudge_agents` (`@mention` for out-of-turn) are a safety net; fix subscription/turn budget in product code if nudges do not help.

## Test isolation

Go tests that touch hub storage, repo agents, or collaboration sandboxes should not write to your real `~/.neural-junkie` tree.

- **`internal/testutil/isolate.go`** — `IsolateNeuralJunkieHome(t)` sets `HOME`, `USERPROFILE`, `NEURAL_JUNKIE_REPO_DIR`, and `NEURAL_JUNKIE_COLLAB_ASSETS_DIR` under a temp directory for the duration of a test.
- **`test/test_main.go`**, **`internal/hub/test_main.go`**, and **`cmd/server/test_main.go`** — package-level `TestMain` hooks apply the same temp-home pattern before any test in those packages runs.
- **Hub collab tests** — use `newTestHub(t)` (in `internal/hub/collab_test_helpers_test.go`) when a test creates collaborations and runs `approveAndExecuteCollabForTest`; it combines home isolation with `SetCollaborationAssetsRootResolver`.

Integration tests in the top-level `test/` package call `useIsolatedRepoStorage(t)`, which delegates to `testutil.IsolateNeuralJunkieHome`.

## Scenario runners and `KEEP=1`

Live scenario scripts against a running hub clean up after themselves by default:

| Script | Default cleanup | Keep artifacts |
|--------|-----------------|----------------|
| `scripts/collab-scenarios.py` | Cancels collaborations, removes `collabs/<id>/` from the scenario workspace | `--keep` |
| `scripts/learning-scenarios.py` | Deletes learnings created during the run, restores hub settings | `--keep` or `KEEP=1` |

Use `--keep` (or `KEEP=1` for learning scenarios) when you want to inspect hub state, workspace files, or learnings after a failed or successful run.
