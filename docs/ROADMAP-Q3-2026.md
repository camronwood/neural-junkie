# Q3 2026 roadmap (Jul–Sep)

Prioritized plan for Neural Junkie from **v1.2.0-beta.5** through stable cut, reliability soak, simplification, and platform runway.

**Last updated:** July 2026  
**Baseline tag:** `v1.2.0-beta.5`  
**Related:** [PLATFORM_ROADMAP.md](PLATFORM_ROADMAP.md) · [PHASE_D_BACKLOG.md](PHASE_D_BACKLOG.md) · [FUTURE_ENHANCEMENTS.md](FUTURE_ENHANCEMENTS.md) · [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md) · [TESTING.md](TESTING.md)

---

## Executive summary

| Month | Theme | Primary outcome |
|-------|-------|-----------------|
| **Jul** | Ship confidence | Collab + parity gates green; stable or beta.6 tag |
| **Aug** | Simplify and polish | D5 specialist simplification; collab + IDE v4.1 gaps |
| **Sep** | Platform runway | D4 distributed hub spike; auth registration approval |

**North star:** Prove Cursor-like reliability under soak before adding platform scale.

---

## Starting point (July 6, 2026)

### Shipped in beta.5

ReAct tools, per-turn routing trace MVP, Runbooks v2, multi-repo workspace scope, collab hardening, LoRA v2, personal learning, Slack diagnostics, layer-gate / fix-loop / test-growth automation.

### In flight on `main` (pre-tag)

| Area | Files / signals |
|------|-----------------|
| Collab reliability | `discussion.go`, `knowledge_route.go`, collab scenarios |
| CLI agent auth | `claude_cli_env.go`, `cli_agent.go`, judge scripts |
| Release judges | `gemini_judge_auth.py`, `deliverable_judge.py`, `check-claude-judge.py` |
| Settings UX | `AutomationSettingsTab`, `ProvidersSettingsTab` |

### Known blockers

| Blocker | Evidence | Gate |
|---------|----------|------|
| Collab soak failures | `layer-gate-collab-full-2026-07-05-2233-iter2.md` | `make layer-gate LAYER=collab-full` |
| Parity soak not confirmed | [reliability-pass-3-2026-06-28.md](testing/reliability-pass-3-2026-06-28.md) | `make test-parity-stable` |
| Platform smoke pending | [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md) Gate 5 | [stable-platform-smoke.md](testing/stable-platform-smoke.md) |
| D5 deferred | [PHASE_D_BACKLOG.md](PHASE_D_BACKLOG.md) | Parity gate green first |

---

## Month 1 — Ship confidence (Jul 6 → Aug 2)

**Theme:** Close the reliability loop and cut stable.

### Priorities

| P | Work | Done when |
|---|------|-----------|
| **P0** | Finish in-flight collab + CLI judge work | `collab-full` gate green |
| **P0** | Parity soak | `test-parity-stable` 3/3 @ 20/20 logged |
| **P1** | Stable or beta.6 tag | Gate 5 minimum smoke signed off |
| **P1** | Claude CLI env + judge parity | `python3 scripts/check-claude-judge.py` + deliverable judge smoke pass |
| **P2** | macOS notarization (`v1.2.1`) | Opportunistic — only when Apple Developer creds available |

### Week-by-week checklist (Month 1)

Use `make release-help` for the full command list.

#### Week 1 (Jul 6–12) — Merge and fast gates

- [ ] Commit / merge in-flight collab, judge, and settings work on `main`
- [ ] `make test-go` — unit + integration green
- [ ] `make test-conversation-contract` — wiring contract green
- [ ] `make test-scenario-assert` — deliverable JSON contracts green
- [ ] `make layer-gate LAYER=ci` — fast CI smoke (no hub)
- [ ] `make layer-gate LAYER=implement` — 20/20 implement scenarios
- [ ] `python3 scripts/lib/deliverable_judge_smoke.py` — judge auth smoke
- [ ] `python3 scripts/check-claude-judge.py` — Claude judge path (when wired)
- [ ] `python3 scripts/check-gemini-judge.py` — Gemini judge path

**Exit criteria:** CI layer + implement layer green; judge smoke passes.

#### Week 2 (Jul 13–19) — Collab reliability

- [ ] `make collab-preflight` — hub, Ollama, agents, scenario list
- [ ] `make layer-gate LAYER=collab` — edge-case regression (~11 scenarios)
- [ ] On failure: `make layer-fix-loop LAYER=collab DRY_RUN=1 MAX_ITER=3` then live loop
- [ ] Targeted repro: `make collab-failure-repro RESTART_HUB=1 VERBOSE=1`
- [ ] `make layer-gate LAYER=collab-full` — full sweep (~15 scenarios, 1–3h)
- [ ] On failure: `make layer-overnight LAYER=collab-full` or `make layer-fix-loop LAYER=collab-full`
- [ ] Log artifacts under `docs/testing/layer-gate-collab-full-*.md`

**Exit criteria:** `collab-full` PASS; no weakened scenario assertions.

#### Week 3 (Jul 20–26) — Parity soak + bundle

- [ ] `make layer-gate LAYER=chat` — chat + conversation regression
- [ ] `make layer-gate LAYER=bundle` — implement + chat + conversation
- [ ] `make test-parity-stable` — 3× implement with hub restart (stable gate)
- [ ] Log to `docs/testing/reliability-parity-soak-YYYYMMDD.log`
- [ ] `make layer-gate LAYER=parity` — alias for 3× implement restart gate
- [ ] Optional full climb: `make layer-climb` — stops at first failure

**Exit criteria:** `test-parity-stable` 3/3 @ 20/20; bundle layer green.

#### Week 4 (Jul 27 – Aug 2) — Platform smoke + tag

- [ ] Run [stable-platform-smoke.md](testing/stable-platform-smoke.md) on beta.5+ installer
  - Minimum: macOS arm64 + one of Windows x64 or Linux x64
- [ ] Update Gate 5 matrix in [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md)
- [ ] `./scripts/verify-updater-manifest.sh <tag> stable` (after tag + CI)
- [ ] Update [STATUS.md](STATUS.md), [CHANGELOG.md](CHANGELOG.md), [release-notes.html](release-notes.html)
- [ ] Tag decision:
  - **Green soak + smoke:** `./scripts/cut-stable-release.sh --execute` → `v1.2.0`
  - **Collab still flaky:** `v1.2.0-beta.6` patch, defer stable one week
- [ ] If Apple creds ready: notarization CI → `v1.2.1` (see [DEVELOPMENT_NOTES.md](DEVELOPMENT_NOTES.md#macos-notarization-release-ci))

**Exit criteria:** Tagged release; Gate 5 signed off; known issues accurate.

### Month 1 — Explicitly out of scope

D4 distributed hub, mobile companion, PostgreSQL backend, SSO.

---

## Month 2 — Simplify and polish (Aug → Sep 2)

**Theme:** Pay down complexity after gates are green.

### Priorities

| P | Work | Epic | Done when |
|---|------|------|-----------|
| **P0** | Specialist simplification | D5 | MCP dedup; QualityReviewer roll-up; fewer redundant specialists |
| **P1** | Collab orchestration | Medium backlog | Fewer `timed_out` discussions; better planning provider defaults |
| **P1** | IDE v4.1 deferred gaps | CHANGELOG | At least one of: remote LSP relay, dev container wizard tab, collab worktrees on remote |
| **P2** | Routing trace UI | Adaptive orchestration | Trace panel usable without `NEURAL_JUNKIE_DEBUG=1` |
| **P2** | Token monitoring (foundation) | FUTURE_ENHANCEMENTS | Per-turn token counts in trace; no full dashboard yet |

### Suggested verification

```bash
make test-go
make test-parity-stable          # confirm no regression after D5
make layer-gate LAYER=collab-full
make collab-scenario SCENARIO=planning-two-agent PROFILE=realistic
```

### Likely tags

- `v1.2.1` — notarization (if creds landed in Month 1)
- `v1.3.0-beta.1` — D5 + collab polish

---

## Month 3 — Platform runway (Sep → Oct 2)

**Theme:** Start enterprise/platform work without destabilizing local-first core.

### Priorities

| P | Work | Epic | Done when |
|---|------|------|-----------|
| **P0** | Distributed hub spike | D4 | Design doc + Redis Pub/Sub POC (2 hub instances, message fan-out) |
| **P1** | Agent registration approval | Auth follow-up | MVP: pending agents require admin approve |
| **P1** | PostgreSQL backend spike | D2 optional | Message store behind interface; SQLite remains default |
| **P2** | SSO / JWT | Phase 3 | RFC + handler sketch; not full SSO |
| **P3** | Semantic code search prototype | Medium backlog | Intent-aware repo search beyond FTS |

### D4 spike scope (keep small)

1. Document message routing boundaries in [ARCHITECTURE.md](ARCHITECTURE.md)
2. Redis Pub/Sub channel for cross-instance broadcast
3. Feature flag — off by default; no production multi-instance promise
4. Closes `single-hub` limitation in design only until soak proves stability

### Likely tags

- `v1.3.0-beta.2` — D4 behind flag + auth registration
- `v1.3.0` stable — only if D4 POC is boring and gates stay green

### Still not scheduled

Mobile companion ([MOBILE_COMPANION_NOTES.md](MOBILE_COMPANION_NOTES.md)), VS Code extension, analytics dashboard, full cost-management suite.

---

## Milestone calendar

| Target date | Milestone |
|-------------|-----------|
| **Mid-Jul** | `collab-full` + `test-parity-stable` green; judge/CLI work merged |
| **Late Jul** | `v1.2.0` stable or `v1.2.0-beta.6` |
| **Aug** | D5 specialist simplification → `v1.3.0-beta.1` |
| **Early Sep** | IDE v4.1 deferred items + routing trace polish |
| **Late Sep** | D4 spike complete; agent registration approval |

---

## Top 5 sequencing (if capacity is tight)

1. Green `collab-full` + `test-parity-stable` — unblocks D5 and stable cut
2. Gate 5 platform smoke + stable tag — product milestone
3. D5 specialist simplification — reduces maintenance surface
4. Collab orchestration hardening — top known limitation (`collab-model-variance`)
5. D4 distributed hub spike — platform phase without full build commitment

---

## Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Collab flakes on local Ollama | Settings → Collaboration planning provider; reliable tier for implementation; do not weaken assertions |
| Stable cut delayed by Gate 5 | Run platform smoke on beta.5 `.dmg` in Week 4 |
| D4 scope creep | Spike only in Month 3; feature flag; SQLite stays default |
| Apple creds unavailable | Ship ad-hoc at `v1.2.0`; track `v1.2.1` for notarization |
| Judge auth failures in CI | `NJ_DELIVERABLE_JUDGE_SKIP=1` for local-only; cloud-first with Ollama fallback per [TESTING.md](TESTING.md) |

---

## How this maps to Phase D

| Epic | Q3 placement | Status |
|------|--------------|--------|
| D1 Agent WebSocket | Pre-Q3 | ✅ Shipped |
| D2 Persistence + search | Pre-Q3 | ✅ Shipped (PostgreSQL → Month 3 spike) |
| D3 Auth + multi-user | Pre-Q3 | ✅ Shipped MVP (SSO → Month 3 RFC) |
| D4 Distributed hub | Month 3 | 🔲 Spike |
| D5 Specialist simplification | Month 2 | 🔲 After parity green |

---

## Operator quick reference

```bash
make release-help                           # start here
make layer-list                             # layers in order
make layer-climb                            # ci → … → parity until fail
make layer-fix-loop LAYER=collab-full       # automated fix loop
make test-parity-stable                     # 3× implement, hub restart
make release-prep                           # full ~30h gate (after layers green)
./scripts/cut-stable-release.sh --execute   # tag stable when checklist passes
```

Report artifacts: `docs/testing/layer-gate-*.md`, `docs/testing/layer-fix-loop-*.md`, `docs/testing/release-prep-*.md`.
