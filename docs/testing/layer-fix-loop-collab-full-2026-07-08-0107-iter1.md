# layer fix loop — collab-full — iteration 1 — 2026-07-08-0107-iter1 UTC

layer=collab-full
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-08-0107-iter1.md
fix_branch=release-prep/layer-collab-full-2026-07-08-0107
git_commit=05db96f
layer_gate_rc=2
agent_rc=1

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience (exit 0)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 0)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict (exit 0)

## Cursor agent output

```text
Connection lost, reconnecting to https://agentn.global.api5.cursor.sh (attempt 1)...
Retry attempt 1...
ActionRequiredError: Authentication error If you are logged in, try logging out and back in.
```

