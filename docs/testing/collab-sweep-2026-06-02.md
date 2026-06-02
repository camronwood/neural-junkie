# Collab scenario sweep log (2026-06-02)

## Environment (A-D implementation validation)

| Check | Result |
|-------|--------|
| `make server-regression` | Hub up (`NEURAL_JUNKIE_RATE_LIMIT=0`, `NEURAL_JUNKIE_DEBUG=1`) |
| `make collab-preflight` | **PASS** — health, Ollama (11 models), default agent roster, 15 scenarios listed |
| Ollama | `ollama serve` running (`qwen2.5:7b` per `env.local`) |

Log: `/tmp/nj-hub.log`  
Full sweep command: `make collab-scenarios-all 2>&1 | tee /tmp/nj-collab-sweep-2026-06-02.log`

## Batch run (partial, aborted)

Log: `/tmp/nj-collab-sweep-2026-06-02.log` (12/15 finished before switch to serial workflow)

| Scenario | Result | Failure tag / notes |
|----------|--------|---------------------|
| `delivery-sandbox-auto-ack` | **PASS** | |
| `document-findings-execution` | **PASS** | |
| `execute-deliverable` | **PASS** | |
| `execution-no-stack-commands` | **FAIL** | `assert_files`: findings.md exists but **too small** |
| `multi-collab-isolation` | **PASS** | |
| `phoenix-resource-api-e2e` | **PASS** | |
| `plan-dependency-prose-regression` | **FAIL** | `assert_plan`: **tasks=0** (plan parser / discussion → plan) |
| `plan-distinct-deliverables-same-agent` | **FAIL** | `assert_plan`: **tasks=0** |
| `plan-findings-task-regression` | **PASS** | |
| `plan-phoenix-combined-regression` | **FAIL** | `assert_plan`: **tasks=0**; BackendEngineer timeout in discussion |
| `planning-two-agent` | **PASS** | |
| `reject-collabs-subfolder` | **PASS** | |
| `resource-api-schema-planning` | **PENDING** | Aborted mid-run |
| `resource-api-schema-regression` | **PENDING** | Not reached |
| `solo-vs-collab-parity` | **PENDING** | Not reached |

**Serial workflow (recommended):** `make collab-sweep-serial RESUME=1` — see [`collab-matrix.tsv`](collab-matrix.tsv).

## Earlier spot-check results

| Scenario | Result | Notes |
|----------|--------|-------|
| `solo-vs-collab-parity` | **FAIL** (spot) | Solo leg: file not on disk before timeout |

## Failure pattern tags (triage)

| Tag | When to use |
|-----|-------------|
| `ollama_down` | Hub log: `Error generating response` + connection refused on `11434` |
| `silent_agent` | `wait_discussion` diagnosis: agent count 0; no `generation_error` posts |
| `stuck_planning` | Never reaches `reviewing` |
| `solo_leg` | `solo-vs-collab-parity` solo channel / file proposal path |
| `gemini_missing` | `resource-api-schema-planning` without Gemini agent |
| `phoenix_repo` | `NEURAL_JUNKIE_SCENARIO_REPO` unset for Phoenix scenarios |

## Product changes in this pass (B/C)

- **LLM errors on collab channel:** `collaboration_discussion` + `generation_error` metadata when generation fails ([`internal/agent/collab_generation_error.go`](../internal/agent/collab_generation_error.go)).
- **Panel stall hints:** `turns_this_round` + `collaboration-planning-stall-banner` ([`CollaborationPanel.tsx`](../desktop/src/components/CollaborationPanel.tsx)).
- **Harness nudges:** `retries` / `nudge_agents` on multi-agent Phoenix-style scenarios.
- **Desktop workspace on collab commands:** [`collaborationOutboundMetadata.ts`](../desktop/src/utils/collaborationOutboundMetadata.ts).

## Full 15-scenario sweep

Run locally after preflight (serial, ~1–3h):

```bash
ollama serve
make server-regression
make collab-preflight
make collab-scenarios-all 2>&1 | tee /tmp/nj-collab-sweep-2026-06-02.log
```

Update this file with a PASS/FAIL row per scenario when the full log is available.
