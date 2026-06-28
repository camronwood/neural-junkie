# Agent Runtime Reliability — Pass 3 Report (2026-06-28)

## Scores vs targets

| Metric | Target | Result |
|--------|--------|--------|
| Outcome UI | Shipped | **PASS** — `ImplementationSessionOutcomeCard`, ChatWindow toasts |
| `go test ./internal/...` | Green | **PASS** |
| `make test-parity-stable` 3/3 @ 20/20 | Gate | **Not run** (requires live hub + 3 sweeps ~hours) |
| Phase 4 simplification | If gate green | **Deferred** (gate not measured this session) |

## Pass 3 RUN — shipped

### Desktop observability

- `ImplementationSessionOutcomeCard` — collapsible outcome badge on agent messages
- TypeScript `ImplementationSessionOutcome` interface
- `IMPLEMENTATION_SESSION_OUTCOME_KEY` constant
- ChatWindow toasts: verify failed, circuit breaker, wrong_route + suggested agent

### Docs

- [IMPLEMENTATION_SESSION.md](../IMPLEMENTATION_SESSION.md) — reliable tier settings
- [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) — implementation mitigation for model variance
- [CHANGELOG.md](../CHANGELOG.md) — unreleased entry
- [PHASE_D_BACKLOG.md](../PHASE_D_BACKLOG.md) — D5 specialist simplification epic

## Diagnostic answer (program complete — pending live gate)

> **What is the biggest area we still need to improve NJ in?**

After this program, the **#1 gap shifts from invisible runtime failures to proving parity under soak**:

1. **Run `make test-parity-stable`** on a healthy hub with software-development pack + Ollama models pulled — confirms 20/20 × 3
2. **Collaboration model variance** remains for planning quality (`collab-model-variance`) — separate from implementation sessions
3. **Specialist simplification (D5)** — intentionally deferred until parity gate is green

## Operator next step

```bash
make server-regression
make implement-scenarios-list
make test-parity-stable   # 3 sweeps, hub restart between
```

Log to `docs/testing/reliability-parity-soak-YYYYMMDD.log`.
