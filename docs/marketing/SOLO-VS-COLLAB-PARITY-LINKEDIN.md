# LinkedIn article — Solo vs collab parity (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-solo-vs-collab-parity-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-solo-vs-collab-parity-article.sh`

**Suggested title (pick one):**
- When Does Multi-Agent Actually Help?
- Solo vs. Collab: Same Task, Same Repo, Different Outcomes
- I Built a Parity Scenario to Stop Guessing About Collaboration

**Feed post teaser:**
> `/collaborate` looks impressive in demos. But when does two agents actually beat one specialist on the same deliverable? I built a parity scenario — same repo, same findings.md task, solo DM vs. structured collab — so we can measure it instead of debating it.

**Hashtags:** `#AI #SoftwareEngineering #DeveloperTools #OpenSource #MultiAgent #AgenticCoding`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/solo-vs-collab-parity.html

**Download CTA:** https://camronwood.github.io/neural-junkie/download.html

**Related articles:** [Collaboration](COLLABORATION-LINKEDIN.md) · [Conversational test harness](CONVERSATIONAL-TEST-HARNESS.md) · [Fix + growth loops](FIX-GROWTH-LOOPS-LINKEDIN.md)

---

## PASTE START

Multi-agent collaboration is easy to demo and hard to justify.

You run `/collaborate @SoftwareArchitect @BackendEngineer`, get a planning thread, approve a plan, and watch tasks execute. It *looks* like a team. But teams have overhead — discussion budgets, plan parsing, approval gates, turn-taking rules. A single `@BackendEngineer` DM might finish the same job in half the time.

So when does multi-agent actually help?

I got tired of arguing from screenshots. I built a **parity scenario** that runs the same deliverable two ways on the same fixture repo — and compares outcomes with the same assertions.

## The question in one sentence

**Does structured collaboration produce a better grounded deliverable than a solo specialist — or just more chat?**

## The setup

Fixture: `minimal-repo` — a tiny workspace with `README.md` and `core/sample/main.go`. Nothing else. No React boilerplate to hallucinate.

Task: write `findings.md` with three bullet findings about **this repo only** — grounded in README and main.go.

Two legs, same assertions:

| Leg | How | Channel | Agents |
|-----|-----|---------|--------|
| **Solo** | DM `@BackendEngineer` directly | `collab-scenarios-solo` | BackendEngineer |
| **Collab** | `/collaborate` with planning → approve → execute | `collab-scenarios` | BackendEngineer + SoftwareArchitect |

The collab goal is intentionally minimal — one task, one owner:

```
@BackendEngineer Plan one task: @BackendEngineer writes collabs/<collab-id>/findings.md
with three bullet findings about this repo (README.md and core/sample/main.go only).
```

Bounded: `--rounds 1 --messages 3`. We're not testing infinite planning theater. We're testing whether the **collab pipeline** can land the same file a solo DM would.

## What we assert (both legs)

Hard checks, not vibes:

- `findings.md` exists with ≥ 40 bytes
- Content references `main.go`, `core/sample`, or `minimal`
- Content does **not** hallucinate `index.js`, `utils.js`, `app.js`, or generic "data handling module" boilerplate
- LLM judge on collab leg: *"PASS if findings summarize this minimal-repo fixture. FAIL for hallucinated paths."*

This is the failure mode that hurts most in production — agents that sound confident while writing about files that don't exist in your workspace.

## Why parity, not just collab regression

Collab regression scenarios answer: *"Does the pipeline work?"* — phases advance, tasks complete, files land on disk.

Parity scenarios answer: *"Is collab worth the overhead for this class of work?"*

If solo passes and collab fails → the pipeline is the problem (plan parsing, workspace binding, execution dispatch).

If both fail the same way → the specialist or workspace context is the problem, not multi-agent orchestration.

If collab passes and solo fails → rare, but interesting — structured planning might help grounding.

If both pass → collab isn't *worse* for this task. Whether it's *better* depends on whether architect input improved the findings — the judge catches qualitative gaps the regex assertions miss.

## What we've learned so far

Early runs surfaced real platform issues, not scenario bugs:

**Solo leg failures** often mean workspace binding — the agent describes the Neural Junkie desktop repo instead of the scenario fixture. That's a context injection problem, not "BackendEngineer can't write markdown."

**Collab leg failures** cluster differently:

- **Plan parsing** — `assert_plan` finds 0 tasks or too many (parser explosion on prose-heavy plans)
- **Execution stalls** — `wait_tasks` timeouts after planning looked fine
- **Participation gaps** — `@BackendEngineer` silent during planning while architect talks
- **Generation errors** — Ollama HTTP timeout before discussion budget completes (we fixed one class by raising client timeout from 360s → 540s)

`solo-vs-collab-parity` failed on the solo leg in a recent `collab-full` fix-loop iteration — while planning-participation scenarios improved after the timeout fix. That's exactly the signal parity is designed to produce: **don't assume collab is the bottleneck.**

## When multi-agent is worth it (and when it isn't)

**Worth it** when:

- Tasks have real dependencies across specialists (security review before implementation)
- Planning quality matters — architect shapes scope before execution burns tokens
- You need audit trail — phases, approvals, task graph on disk
- The deliverable benefits from cross-domain synthesis

**Probably not worth it** when:

- One specialist owns the entire task end-to-end
- The "collab" is really one agent with an audience
- Planning overhead exceeds execution time
- Workspace context is already clear and bounded

The parity scenario uses the **minimal** case on purpose. If collab can't match solo here, it won't magically excel on harder work without fixing the pipeline first.

## How it fits the test harness

`solo-vs-collab-parity` lives in `scenarios/collab/` and runs as part of `make layer-gate LAYER=collab-full`. It's Layer 3 of the conversational harness — live JSON scenarios with real Ollama agents.

It composes with release engineering loops:

```bash
make layer-gate LAYER=collab-full
make layer-fix-loop LAYER=collab-full
```

When parity fails, the fix-loop brief includes targeted rerun commands for both legs. No weakening assertions to greenwash — if solo passes and collab doesn't, the hub has work to do.

## The scenario shape (simplified)

```json
{
  "name": "solo-vs-collab-parity",
  "workspace": { "fixture": "minimal-repo" },
  "solo_leg": {
    "message": "@BackendEngineer Create collabs/parity-solo/findings.md …",
    "any_match": ["main\\.go", "core/sample"],
    "none_match": ["index\\.js", "utils\\.js"]
  },
  "collaborate": {
    "goal": "@BackendEngineer Plan one task: …",
    "flags": ["--workspace", "--rounds", "1", "--messages", "3"]
  },
  "expect_deliverables": [
    { "path": "collabs/<collab-id>/findings.md", "llm_judge": { … } }
  ]
}
```

Full scenario: `scenarios/collab/solo-vs-collab-parity.json`

## What we didn't try to solve

- Proving collab is *always* better — parity measures equivalence and grounding, not creativity scores
- Subjective "was the architect's input valuable?" — the judge catches hallucination; human review catches insight
- Every task type — this is one deliverable class. Website builds and security refactors need their own parity scenarios

The bet: **measure collaboration overhead** instead of demoing it.

## Try it

```bash
make collab-scenario SCENARIO=solo-vs-collab-parity
make layer-gate LAYER=collab-full
```

Docs: [COLLABORATION.md](https://github.com/camronwood/neural-junkie/blob/main/docs/COLLABORATION.md) · [TESTING.md](https://github.com/camronwood/neural-junkie/blob/main/docs/TESTING.md)

Download: https://camronwood.github.io/neural-junkie/download.html

---

*Neural Junkie is a personal open-source project. Feedback and scenario failures welcome on GitHub.*

## PASTE END
