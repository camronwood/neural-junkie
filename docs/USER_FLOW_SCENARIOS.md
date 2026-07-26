# Real-world user-flow scenarios

Product journeys a person would paste into Neural Junkie — greenfield apps, research docs, websites, and “fix my broken app” — separate from regression locks under `scenarios/chat|implement|collab/`.

## Why a separate suite

| Gate | Role |
|------|------|
| `implement` / `collab-core` / `collab` | Bug regressions + Cursor parity contracts |
| **`user-flows`** | End-to-end “ship me this product” prompts |

JSON lives under `scenarios/user-flows/{implement,collab}/` so the implement **20/20** gate does not grow when we add journeys. One reused core scenario (`vite-boot-fix-corrupt-appjs`) is referenced by name from the default implement dir.

## Suite members

| Scenario | Kind | What the user asks |
|----------|------|--------------------|
| `trip-research-vacation` | implement | Research STL → Seaside FL; write `vacation_2026.md` |
| `rust-blackjack-2d` | implement | Local CLI/2D Rust blackjack vs the house |
| `nodejs-user-crud` | implement | Node.js + TypeScript user CRUD API + seed users |
| `ios-trivia-swift` | implement | Local Swift trivia (3 lives, 15s timer) under `TriviaGame/` |
| `collaboration-station-branded` | collab | Static CS site — home/about/contact; blue/green/yellow (+ gray/white/black/red) |
| `admin-cms-website` | collab | Sample site + API + admin login + content panel |
| `vite-boot-fix-corrupt-appjs` | implement (core) | “App won’t boot — find and fix” |

### Multi-turn journeys (goal completion over back-and-forth)

One-shot user-flows catch “can the agent ship from a single paste?” Journeys catch the failure modes real sessions hit: clarify → constrain → mid-course correction → finish. Each is tagged `journey` + `long-horizon` (≥4 user sends) and still gates on **disk deliverables**, not chat phrases.

**Default suite:** journeys (and a few greenfield flows still landing) have `skip_reason` so `make user-flow-scenarios` / `--all` does **not** block beta releases. Force-run any one with:

```bash
make user-flow-scenario SCENARIO=journey-crud-clarify-correct VERBOSE=1
```

| Scenario | Turns shape | Goal on disk |
|----------|-------------|--------------|
| `journey-crud-clarify-correct` | plan → Node/TS CRUD → add `/health` → verify | `package.json` + server with users + `/health` |
| `journey-blackjack-cli-correction` | plan → implement → CLI-only correction → finish | `Cargo.toml` + CLI blackjack (no Bevy/wgpu) |
| `journey-boot-fix-then-feature` | fix corrupt `App.js` → confirm → add heading → lock | `App.js` gone; `App.tsx` has `Workspace Ready` |
| `journey-notes-rename-to-memos` | build `/notes` → rename to memos → scrub `/notes` → stop | `/memos` CRUD; no `/notes` routes |
| `journey-landing-brand-correction` | Brightest Bio page → rename brand → tagline → lock | `index.html` with Neural Junkie + tagline |

Canonical list: `scripts/lib/user_flow_scenarios.py`.

## Commands

```bash
make user-flow-scenarios-list
make user-flow-scenario SCENARIO=trip-research-vacation VERBOSE=1
make user-flow-scenarios
make layer-gate LAYER=user-flows
```

## Model benchmark

Compare coder models on the same journeys:

```bash
make model-benchmark SUITE=user-flows-quick   # trip + blackjack + Node CRUD + boot fix
make model-benchmark SUITE=user-flows         # full 7-scenario layer
```

See [testing/MODEL_BENCHMARK.md](testing/MODEL_BENCHMARK.md).

Prerequisites match other live gates: regression hub + models (`make layer-gate` boots the stack). Claude-required collabs need Claude Code CLI online.

## Adding a new user flow

1. Prefer natural-language goals (the user’s words), then pin **exact deliverable paths** so asserts stay stable.
2. Put new JSON in `scenarios/user-flows/implement/` or `…/collab/` (not the core dirs).
3. Use fixture `user-flow-empty` for greenfield; the suite runner wipes agent leftovers after each run.
4. Register the entry in `scripts/lib/user_flow_scenarios.py`.
5. Declare `expect_deliverables` with quality bars (`for_question` / `llm_judge`) — CI validates via `make test-scenario-assert`.
6. Tag scenarios with `"user-flow"`.
7. For multi-turn journeys: ≥4 `send` steps, `"evaluation": {"long_horizon": true}`, tags `journey` + `long-horizon`. Script the **user** turns; gate completion with `until_file_match` / `until_metadata_keys` (note: `until_file_match.contains_all` is **literal** substring; use `any_match` for regex).

## Related

- [TESTING.md](TESTING.md) — layer gates
- [CHAT_SCENARIOS.md](CHAT_SCENARIOS.md) — conversation quality (not product journeys)
- [COLLABORATION.md](COLLABORATION.md) — collab engine
