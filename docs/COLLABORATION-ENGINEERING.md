# Collaboration engineering notes (repo)

**LinkedIn publish copy** (article + teaser + hashtags): [`docs/marketing/COLLABORATION-LINKEDIN.md`](marketing/COLLABORATION-LINKEDIN.md)

**User / operator reference:** [`docs/COLLABORATION.md`](COLLABORATION.md)

---

## Delivery quality harness

After prompt or orchestration changes, run:

```bash
# Hub for live scenarios (disable rate limit for matrix sweeps)
NEURAL_JUNKIE_RATE_LIMIT=0 NEURAL_JUNKIE_SKIP_SESSION_RESTORE=1 make server

# In-process smoke (no live agents)
make collab-smoke

# Planning + auto-ack + execution (needs hub + Ollama/agents)
python3 scripts/collab-scenarios.py --list
make collab-scenario SCENARIO=planning-two-agent
make collab-scenario SCENARIO=delivery-sandbox-auto-ack
make collab-scenario SCENARIO=resource-api-schema-planning PROFILE=fast
make collab-scenario SCENARIO=execute-deliverable PROFILE=fast
```

Key scenarios under `scenarios/collab/`. Scenario runner supports `assert_collab.workspace_acknowledged` and `assert_plan.max_tasks`.

Internal-only detail (commands, scenario JSON, Makefile targets) also lives in `COLLABORATION.md` § Live scenario harness. Do not paste that section into LinkedIn; use the marketing file above.
