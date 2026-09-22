# layer fix loop — user-flows — iteration 1 — 2026-09-22-0505-iter1 UTC

layer=user-flows
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-user-flows-2026-09-22-0505-iter1.md
fix_branch=release-prep/layer-user-flows-2026-09-22-0505
git_commit=3c483238
layer_gate_rc=124
agent_rc=2

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collaboration-station-branded (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario ios-trivia-swift (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario journey-blackjack-cli-correction (exit 1)

## Cursor agent output

```text
ActionRequiredError: Named models unavailable Free plans can only use Auto. Switch to Auto or upgrade plans to continue.
```

