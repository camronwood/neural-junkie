You are fixing failures from Neural Junkie **layer gate: collab-full**.
Layer goal: Full collab-scenarios sweep (~15 scenarios, 1–3h)
Do not weaken assertions. Fix product/hub/agent behavior first (docs/TESTING.md).
After edits, run the targeted verification commands in this brief.

---


Rules (mandatory):
- Triage product/hub/agent behavior first, harness second (docs/TESTING.md).
- Do NOT weaken test assertions or scenario contracts to greenwash flakes.
- Prefer minimal, focused fixes in the neural-junkie repo.
- After edits, run the targeted verification commands listed below.
- Summarize what you changed and which commands you ran.

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-2233-iter1.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:collab-participation-two-agent-strict [code]
```text
=== FAIL: collab-participation-two-agent-strict ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] FrontendEngineer: **FrontendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] SecurityReviewer: **SecurityReviewer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---
```

### collab:collaboration-station-website [code]
```text
=== FAIL: collaboration-station-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] FrontendEngineer: **FrontendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---
```

### collab:collaboration-station-website-sa [code]
```text
=== FAIL: collaboration-station-website-sa ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
  --- end ---
```

### collab:document-findings-execution [code]
```text
=== FAIL: document-findings-execution ===


  --- transcript (agent messages) ---
  --- end ---
```

### collab:execute-deliverable [code]
```text
=== FAIL: execute-deliverable ===


  --- transcript (agent messages) ---
  --- end ---
```

### collab:execution-no-stack-commands [code]
```text
=== FAIL: execution-no-stack-commands ===


  --- transcript (agent messages) ---
  --- end ---
```

### collab:make-me-a-website [code]
```text
=== FAIL: make-me-a-website ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---
```

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] PlatformEngineer: **PlatformEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=3 counts={'BackendEngineer': 2, 'PlatformEngineer': 1}
  generation_error posts in channel: 3
  ok: @BackendEngineer — 2 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @PlatformEngineer — 1 message(s)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
```

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={}
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
```

### collab:plan-distinct-deliverables-same-agent [code]
```text
=== FAIL: plan-distinct-deliverables-same-agent ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={}
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
```

### collab:plan-findings-task-regression [code]
```text
=== FAIL: plan-findings-task-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=1 counts={'Assistant': 1}
  generation_error posts in channel: 1
  ok: @Assistant — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 2
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/6
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] PlatformEngineer: **PlatformEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---
```

### collab:planning-two-agent [code]
```text
=== FAIL: planning-two-agent ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: Here is my planning perspective for this collaboration:  I will focus on drafting the markdown document  based on the approved outline and analyzing existing  s
  --- end ---

agent discussion: total=1 counts={'Gemini': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Gemini — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 8
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
```

### collab:resource-api-schema-planning [code]
```text
=== FAIL: resource-api-schema-planning ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: My planning perspective for this collaboration focuses on  delivering a comprehensive markdown document. To effectively draft  this document, a clear understand
  --- end ---

agent discussion: total=1 counts={'Gemini': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Gemini — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 8
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
```

### collab:resource-api-schema-regression [code]
```text
=== FAIL: resource-api-schema-regression ===

  FAIL [solo]: no Assistant reply (timeout waiting for @Assistant (baseline=0, counts={}))
```

### collab:solo-vs-collab-parity [code]
```text
=== FAIL: solo-vs-collab-parity (solo leg) ===

make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 11799s)
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: - Task 1: @Assistant - Write collabs/fixtures/minimal-repo/collabs/001/findings.md summarizing  README purpose and scenario usage from `/Users/camronwood/develo
    [collaboration_discussion] SoftwareArchitect: I'm ready to help with your React project!  Based on the workspace context, I can see  you have: - A React/TypeScript application with service  worker capabilit
    [collaboration_discussion] Assistant: - Task 1: @Assistant - Write `collabs/<id>/findings.md` extracting  repository purpose from README.md lines 1-4. - Task  2: @SoftwareArchitect - Write `collabs/
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: - Task 1: @Assistant - Write `collabs/<id>/findings.md` summarizing  README.md content (lines 1-4) - Task 2: @SoftwareArchitect  - Write `collabs/<id>/architect
  --- end ---
```

### collab:collab-human-planning-interject [code]
```text
=== FAIL: collab-human-planning-interject ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={}
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
```

### collab:collab-minimal-completion-regression [code]
```text
=== FAIL: collab-minimal-completion-regression ===


  --- transcript (agent messages) ---
  --- end ---
```

### collab:collab-no-edit-after-cancel [code]
```text
=== FAIL: collab-no-edit-after-cancel ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: I agree this is a minimal scenario. Given  the simple fixture repo structure, here's my task  list: - Task 1: @Assistant - Write collabs/9cf5c5fb-0130-4d6b-a74e
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/2
```

### collab:collab-participation-three-agent [code]
```text
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-2233-iter1.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
- python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience
- python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject
- python3 scripts/collab-scenarios.py --scenario collab-minimal-completion-regression
- python3 scripts/collab-scenarios.py --scenario collab-no-edit-after-cancel
- python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent
- python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website-sa
- python3 scripts/collab-scenarios.py --scenario document-findings-execution
- python3 scripts/collab-scenarios.py --scenario execute-deliverable
- python3 scripts/collab-scenarios.py --scenario execution-no-stack-commands
- python3 scripts/collab-scenarios.py --scenario make-me-a-website
- python3 scripts/collab-scenarios.py --scenario phoenix-resource-api-e2e
- python3 scripts/collab-scenarios.py --scenario plan-dependency-prose-regression
- python3 scripts/collab-scenarios.py --scenario plan-distinct-deliverables-same-agent
- python3 scripts/collab-scenarios.py --scenario plan-findings-task-regression
- python3 scripts/collab-scenarios.py --scenario plan-phoenix-combined-regression
- python3 scripts/collab-scenarios.py --scenario planning-two-agent
- python3 scripts/collab-scenarios.py --scenario resource-api-schema-planning
- python3 scripts/collab-scenarios.py --scenario resource-api-schema-regression
- python3 scripts/collab-scenarios.py --scenario solo-vs-collab-parity

