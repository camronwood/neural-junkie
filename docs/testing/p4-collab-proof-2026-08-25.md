# P4-A — Collab proof (2026-08-25)

Validation matrix from P4 plan against hub `127.0.0.1:18765`, Ollama regression roster (`qwen3.5:9b`), `PROFILE=fast`.

## Preflight

| Step | Command | Result |
|------|---------|--------|
| 1 | `make collab-preflight` | **PASS** |

## Core matrix (×2 required)

| Run | Command | Scenarios | Result | Log |
|-----|---------|-----------|--------|-----|
| 1 | `make collab-scenarios-core PROFILE=fast VERBOSE=1` | 8/8 | **PASS** | `/tmp/nj-p4-core-run1.log` |
| 2 | `make collab-scenarios-core PROFILE=fast` | 8/8 | **PASS** | `/tmp/nj-p4-core-run2.log` |

Scenarios in core profile:

- `collab-participation-two-agent-strict` — PASS
- `collab-participation-three-agent` — PASS (#20 coverage)
- `collab-human-planning-interject` — PASS
- `collab-generation-error-resilience` — PASS
- `planning-two-agent` — PASS (#20 coverage)
- `plan-dependency-prose-regression` — PASS
- `collab-minimal-completion-regression` — PASS
- `document-findings-execution` — PASS (#20 repro; harness nudged silent agents once, then planning completed)

Additional isolated run: `planning-two-agent PROFILE=fast VERBOSE=1` — **PASS** (retry after earlier hub contention).

## Targeted repros

| Step | Command | Target | Result | Notes |
|------|---------|--------|--------|-------|
| #20 | Included in core (`document-findings-execution`) | Agent silence | **PASS** | One nudge cycle; P0 handoff + watchdog recovered |
| #21 cloud | `NJ_REGRESSION_CLAUDE_CLOUD=1 make collab-scenario SCENARIO=plan-findings-task-regression VERBOSE=1` | Cloud `generation_error` | **SKIPPED** | Claude preflight failed (`claude not authenticated`); no `sk-ant-…` API key |

## Automated unit tests

| Command | Result |
|---------|--------|
| `make test-go` | **PASS** |
| `npm test -- --run` (desktop) | **PASS** (599 tests) |

## Triage notes

- Earlier failure (`ConnectionResetError` during concurrent collab boot) was **infrastructure contention**, not a product regression — resolved by serial runs with `NJ_RESTART_HUB_BETWEEN_SCENARIOS=1`.
- P0 mitigations in place: `maxPlanningGenerationErrorsPerTurn=2` handoff, CollaborationPanel gen-error banner, pending-file copy.

## Issue disposition

| Issue | Disposition | Rationale |
|-------|-------------|-----------|
| [#20](https://github.com/camronwood/neural-junkie/issues/20) | **Downgrade → Mitigated** | Core matrix green ×2; #20 repro scenario passes with harness nudge |
| [#21](https://github.com/camronwood/neural-junkie/issues/21) | **Remain open (mitigated on Ollama)** | Cloud repro not run — needs operator re-smoke with Anthropic/Gemini creds |

Evidence artifact: this file. Update [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) to match.
