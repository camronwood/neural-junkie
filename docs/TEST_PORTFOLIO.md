# Test portfolio (tiers, keep/cut, cost)

Canonical map of Neural Junkie automated tests after the 2026-07 suite thinning. Day-to-day commands: `make release-help`. Workflow detail: [TESTING.md](TESTING.md).

## Cost reality

| Claim / estimate | Observed |
|------------------|----------|
| Layer climb sum ~16.5h | Spec upper bound with retries; exclusive Tier A ~**45–60m** |
| Overnight / release-prep ~**30h** | Mostly fix-loop thrashing + flake retries. Clean green pass is ~**4h** (`test-everything-full` + short parity) |
| CI layer ~15m est | ~**2m** median |

Live LLM layers that historically fail often (chat full regression, collab edge/full, user-flows) are **soak / quarantine**, not pre-tag blockers.

## Naming trap

**`LAYER=parity` ≠ `scenarios/parity/`.**  
`make layer-gate LAYER=parity` runs **implement scenarios 3× with hub restart**. The JSON under `scenarios/parity/` is opt-in via `make parity-scenarios`.

## Tiers

```text
Tier A (daily / pre-tag / layer-climb default)
  ci → implement → collab-core → chat (canary)

Tier B (overnight soak — invoke explicitly)
  chat-full → collab → collab-full → parity

Tier C (quarantine / advisory — not ship gates)
  user-flows, bundle, sut-loop, test-growth (live), model-benchmark Arena when pack missing
```

| Layer | Est (recalibrated) | Role |
|-------|--------------------|------|
| `ci` | 5m | `test-all` + conversation contract |
| `implement` | 15m | Non-optional implement file gates |
| `collab-core` | 30m | Participation + planning (~8) |
| `chat` | 30m | Chat **canary** tag + conversation regression |
| `chat-full` | 120m | Full chat `regression` tag (soak) |
| `collab` | 75m | Edge suite (thinned website/findings) |
| `collab-full` | 120m | All collab scenarios |
| `parity` | 45m | Implement ×3 restart |
| `bundle` | — | Overlaps implement+chat; available but **not** in climb |
| `user-flows` | — | Product journeys; quarantine until 2 consecutive green overnight runs |

## Strongest signal (keep)

- Unit/CI: Go, Vitest, `scripts/lib/*_test.py`, conversation contract
- Implement: `until_file_match` / `until_metadata_keys` (non-optional)
- Collab-core participation/planning
- Chat canaries with hard asserts (workspace, durable state, topic switch, soft followups, interject, closure)

## Overlap clusters (collapsed)

Keep one representative; demote twins with `"optional": true` and/or tag `soak` (still runnable via `--scenario` / `--include-optional` / soak layers).

| Cluster | Keep | Demote |
|---------|------|--------|
| Specialist workspace | `dm-backend-workspace`, `dm-frontend-workspace` | other `dm-*-workspace` → soak (drop `regression`) |
| Theme implement | `rules-constrained-implement` | `theme-toggle`, `react-theme-toggle`, `react-theme-multi-file` |
| Go repair | `verify-failure-one-repair` | `go-build-error-fix`, `go-test-failure-repair` |
| Tauri/Makefile boot | `tauri-make-start-all-missing` | `tauri-make-start-all-best-of-k`, `app-wont-boot-fix-like` |
| Website collab | `collaboration-station-website` | `collaboration-station-website-sa`, `make-me-a-website` (edge list) |
| Findings execute | `document-findings-execution` | `execute-deliverable`, `execution-no-stack-commands` (edge list) |
| Entity continuity | SUT `dm-long-horizon-entity-retention` | chat `dm-pronoun-followup-3turn` → soak |
| Greetings | (manual / non-gate) | already off `regression` |

## Chat tags

| Tag | Meaning |
|-----|---------|
| `canary` | Tier A `LAYER=chat` |
| `regression` | Tier B `LAYER=chat-full` |
| `soak` | Overnight / optional; not canary |
| `optional` | Skip when agents offline; advisory fails |

## Meta-harnesses

| Tool | Policy |
|------|--------|
| `sut-loop` | Release-eng only; not in layer-climb; require live re-verify before accepting fixes |
| `test-growth-loop` | Defaults to `SKIP_LIVE=1` (Layer A / unit companions); do not auto-strengthen flaky live scenarios |
| `model-benchmark` | Arena track: if Model Arena pack is missing, **skip** (do not fail release-prep) |
| `layer-fix-loop` | Isolate one failing layer first; do not run on full release-prep thrash |
| `semantic-eval` | Live classify+policy on `scenarios/routing/semantic-intents.json`; dual gate = action accuracy ≥0.90 and misstamp_rate ≤0.05 |
| `semantic-eval-compare` | Baseline vs candidate (default now `qwen3.5:9b`); promote a *new* candidate only if dual-gate holds and wins |
| `memory-retrieval-corpus` | CI gold gate for conversation memory `Search` (`scenarios/memory/retrieval-corpus.json`) |
| `memory-eval` | Live embed retrieval dual gate (`hit_rate ≥ 0.90`, `forbidden_hit_rate ≤ 0.05`; needs Ollama `nomic-embed-text`) |
| Scoreboard | Latest JSON under `docs/testing/semantic-eval-*.json` and `docs/testing/memory-eval-*.json`; see [CONTEXT_MODEL.md](CONTEXT_MODEL.md) |

## Related

- [TESTING.md](TESTING.md) — commands
- [CONTEXT_MODEL.md](CONTEXT_MODEL.md) — semantic stamp graduation
- [USER_FLOW_SCENARIOS.md](USER_FLOW_SCENARIOS.md) — quarantine journeys
- [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md) — stable cut (separate from beta thinning)
