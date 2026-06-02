# LinkedIn post 3 — Test harness (beta.21)

**Image:** `assets/neural-junkie-beta21-test-ad-1200.png`  
**Audience:** People who **clone the repo**, open PRs, or run pre-release checks — not installers-only users.

---

## On-image copy

**DON'T CLICK-TEST THE HUB.**

Three layers: Go CI · API smoke · live JSON scenarios with real Ollama agents.

---

## Feed post

> You can’t unit-test a conversation. You *can* stop click-testing the hub every time a prompt changes.
>
> **beta.21** expands the **Neural Junkie test harness** (for people who build from source):
>
> 1. **Deterministic Go** — `make test-go` / `make test-all` — router, collab lifecycle, plan parser, workspace visibility shortcuts  
> 2. **API smoke** — `make collab-smoke` — plan → review → execute without Ollama  
> 3. **Live JSON scenarios** — `make chat-scenarios-regression`, `make collab-scenario-regression` — real agents; chat and collab fail for different reasons, so they’re separate suites
>
> New: `docs/TESTING.md` (temp-home isolation for Go tests, scenario cleanup, full checklist).
>
> Installers: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21  
> Clone & run: https://github.com/camronwood/neural-junkie/blob/main/docs/TESTING.md
>
> #Testing #OpenSource #MultiAgent #CI #DeveloperTools

**Do not mention:** Slack marketing, “context model v2” pipeline jargon.

**Pin comment:** “If a scenario flakes, re-run once with `VERBOSE=1` and `NEURAL_JUNKIE_RATE_LIMIT=0` on the hub — Ollama load still varies.”
