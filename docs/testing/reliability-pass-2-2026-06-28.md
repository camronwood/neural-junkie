# Agent Runtime Reliability — Pass 2 Report (2026-06-28)

## Scores vs targets

| Metric | Target | Result |
|--------|--------|--------|
| `go test ./internal/implementation/routing/...` | Green | **PASS** (incl. `TestSelectProviderIDReliableRepairTier`) |
| Outcome backend coverage | redirect + routing metadata | **Shipped** |
| `make implement-scenarios` | ≥18/20 | **Not run** (live hub) |
| `make test-parity-stable` | 1 sweep ≥18/20 | **Not run** (live hub) |

## Pass 2 RUN — shipped

### Reliable-tier routing

- `implementation.reliable_tool_model` (default `qwen2.5-coder:14b`) — used repair round 1+
- `implementation.reliable_provider_id` — opt-in cloud, repair round 2+ only (`reliable_repair_tier`)
- `ClassifyImpl` maps boot-fix / repair / verify-failure to `TaskImplementHeavy`
- Session context hints: `ContextWithImplementationRoutingHints`
- Repair loop re-plans provider + tool model before repair round

### Outcome completeness (backend)

- Wrong-route DM redirect emits `implementation_session_outcome` (`wrong_route`, `suggested_agent`)
- `implementation_session_complete` only when session ran or outcome attached
- Outcome includes `routing_reason` / `routing_tool_model` from routing snapshot

### Settings UI

- Implementation sessions: reliable tool model + reliable provider dropdown

## Diagnostic answer

> **What is the biggest area we still need to improve NJ in?**

**User-visible observability** — backend now emits rich `implementation_session_outcome` metadata, but users previously only saw prose summaries. Pass 3 adds the outcome card and toasts. Live parity gate still unverified without hub soak.

## Pass 3 scope delta

- `ImplementationSessionOutcomeCard` in chat
- Failure toasts on verify/circuit-breaker/wrong-route
- Full `make test-parity-stable` 3/3 soak
- Phase 4 simplification only if gate green
