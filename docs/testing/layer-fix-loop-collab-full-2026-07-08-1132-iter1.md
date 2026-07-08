# layer fix loop — collab-full — iteration 1 — 2026-07-08-1132-iter1 UTC

layer=collab-full
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-08-1132-iter1.md
fix_branch=release-prep/layer-collab-full-2026-07-08-1132
git_commit=6736581
layer_gate_rc=2
agent_rc=0

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience (exit 0)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject (exit 0)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 0)
- [OK] python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict (exit 0)

## Cursor agent output

```text
**`plan-dependency-prose-regression` re-run: PASS** (~7.5 min)

All steps passed after the scenario contract fix — all three agents contributed (BackendEngineer, SoftwareArchitect, Claude), plan assertions held, and approve-plan reached executing phase. That confirms the earlier failure was the wrong `required_agents` entry (`PlatformEngineer` vs `@Claude`), not product behavior.
```

