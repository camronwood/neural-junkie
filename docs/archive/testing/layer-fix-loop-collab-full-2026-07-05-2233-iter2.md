# layer fix loop — collab-full — iteration 2 — 2026-07-05-2233-iter2 UTC

layer=collab-full
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-2233-iter2.md
fix_branch=release-prep/layer-collab-full-2026-07-05-2233
git_commit=09360a5
layer_gate_rc=2
agent_rc=1

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience (exit 0)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-minimal-completion-regression (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-no-edit-after-cancel (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 0)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict (exit 0)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collaboration-station-website (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario collaboration-station-website-sa (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario document-findings-execution (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario execute-deliverable (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario execution-no-stack-commands (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario make-me-a-website (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario phoenix-resource-api-e2e (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario plan-dependency-prose-regression (exit 0)
- [FAIL] python3 scripts/collab-scenarios.py --scenario plan-distinct-deliverables-same-agent (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario plan-findings-task-regression (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario plan-phoenix-combined-regression (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario planning-two-agent (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario resource-api-schema-planning (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario resource-api-schema-regression (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario solo-vs-collab-parity (exit 1)

## Cursor agent output

```text
RetriableError: Connection stalled
```

