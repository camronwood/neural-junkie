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

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-1158-iter2.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:make-me-a-website [code]
```text
=== FAIL: make-me-a-website ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: # Investigation Summary: Resource API Schema Standardization/Registration ##  Critical Finding: Missing Workspace Context The referenced directories  (`resource
    [collaboration_discussion] SoftwareArchitect: The provided WORKSPACE CONTEXT indicates a JavaScript project  using React. This is inferred from the presence  of React-specific files such as `App.tsx`, `Them
    [collaboration_discussion] BackendEngineer: ERROR: Error in read_file: open /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/App.tsx: no such  file or directory
    [collaboration_discussion] SoftwareArchitect: The workspace context indicates a React application. Key  files and directories include React component files like  `App.tsx` and `Header.tsx`, TypeScript decla
    [collaboration_discussion] BackendEngineer: The project is a React application with TypeScript,  as indicated by the presence of `.tsx` files  like `AccentButton.tsx` and `Header.tsx`. It uses CSS modules
    [collaboration_discussion] Assistant: # Resource API Schema Registration Document ## Deliverable  Path `collabs/<id>/resource-api-schema-registration.md` --- ## 1. Executive Summary This  document o
  --- end ---
```

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer:
```

### collab:solo-vs-collab-parity [code]
```text
s collaboration: For Task 1 (Define Scope), we  should focus on three key areas: **inventory mapping**  (iden
    [collaboration_discussion] Assistant: Based on your file tree showing Workbox PWA  tooling (caching strategies, service workers), React components (`Header.tsx`),  and a `workbox-webpack-plugin` con
  --- end ---

agent discussion: total=7 counts={'FrontendEngineer': 4, 'Assistant': 3}
  ok: @Assistant — 3 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @FrontendEngineer — 4 message(s)
  system turn handoffs in channel: 7
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=3/10
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: ### Task 1: Define Scope (@Assistant) with Deliverable  collabs/d914f9be-ec30-4bb4-ba83-2a02f60748a1/scope.md - This task will outline the goals,  objectives, a
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] Assistant: 📋 **Assistant's Planning Perspective:** From a coordination standpoint,  I recommend we start with Task 1 (Scope  Definition) first before proceeding to review 
  --- end ---

agent discussion: total=3 counts={'BackendEngineer': 1, 'Assistant': 2}
  generation_error posts in channel: 1
  ok: @Assistant — 2 message(s)
  ok: @BackendEngineer — 1 message(s)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/10
=== FAIL: resource-api-schema-regression ===

  FAIL [solo]: no Assistant reply (timeout waiting for @Assistant (baseline=0, counts={}))
=== FAIL: solo-vs-collab-parity (solo leg) ===
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: @Assistant -- Task 1: Write `collabs/<id>/findings.md` summarizing README.md  content (fixture repo purpose and scenario test usage);  @SoftwareArchitect -- Tas
    [collaboration_discussion] SoftwareArchitect: The workspace context indicates a JavaScript/TypeScript project using  React, as evidenced by the presence of React-specific  files like `App.tsx`, `Header.tsx`
    [collaboration_discussion] SoftwareArchitect: This project is a React application, as evidenced  by the presence of React component files (e.g.,  `App.tsx`, `Header.tsx`) and the `react-app-env.d.ts` file. 
    [collaboration_discussion] Assistant: # Collaboration Planning Session Recap **Goal:** Create minimal  documentation from a fixture repo for scenario tests.  **Discussed Decisions:** - Task 1: Summa
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: No matching chunks found.
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Gemini: Here is the planning perspective for this collaboration:  The overall goal is to design and implement  a "Collaboration Station" website with home, about, and  
    [collaboration_discussion] SecurityReviewer: I agree with the current plan. The tasks  are well-defined and cover the necessary aspects of  reviewing existing work, conducting a security audit, and  creati
    [collaboration_discussion] FrontendEngineer: ERROR: Error in read_file: open /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/website-prior-collab/collabs/b222bffe-39e8-4b00-91ca-ee1
    [collaboration_discussion] SecurityReviewer: No matching chunks found. I agree with the  current plan. The tasks are well-defined and cover  the necessary aspects of reviewing existing work, conducting  a 
  --- end ---
```

### collab:collaboration-station-website [code]
```text
=== FAIL: collaboration-station-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: ERROR: Error in list_dir: open /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/website-prior-collab/collabs/b222bffe-39e8-4b00-91ca-ee1c
    [collaboration_discussion] SoftwareArchitect: I agree. Let's proceed with the planning phase  for designing the "Collaboration Station" website. The first  step is to review the existing work under  `collab
    [collaboration_discussion] Gemini: My perspective for this collaboration is to first  confirm the absence of prior work, which I  have done. Since `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/` 
    [collaboration_discussion] Gemini: I agree with the overall plan and the  confirmed absence of prior work in `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/`. My  role is to handle the implementat
  --- end ---
```

### collab:collaboration-station-website-sa [code]
```text
=== FAIL: collaboration-station-website-sa ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description
  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description
  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
  --- end ---
```

### collab:execution-no-stack-commands [code]
```text
=== FAIL: execution-no-stack-commands ===


  --- transcript (agent messages) ---
  --- end ---
```

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: ```markdown - **Task:** Generate task cards for a  collaboration scenario in a minimal sample project. -  **Instructions:** - Create a minimal sample project wi
  --- end ---

agent discussion: total=1 counts={'SoftwareArchitect': 1}
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 4
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
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
    [collaboration_discussion] PlatformEngineer: As the @PlatformEngineer, my primary focus is on  the CI/CD pipeline and runtime configurations. Task 3,  which involves writing the CI/CD pipeline description,
    [collaboration_discussion] PlatformEngineer: # Collaboration Recap ## Goal The goal of  this collaboration is to investigate and standardize the  resource API document schema. The team aims to  create comp
  --- end ---

agent discussion: total=2 counts={'PlatformEngineer': 2}
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @PlatformEngineer — 2 message(s)
  system turn handoffs in channel: 4
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/8
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: {"status":"proposed","path":"markdown_doc_structure.md"}
    [collaboration_discussion] SoftwareArchitect: This workspace is a React application. It contains  typical React components (`.tsx` files), core logic (`.ts`  files), and styles (`.css` and `.js` files). The
    [collaboration_discussion] PlatformEngineer: The provided WORKSPACE CONTEXT indicates a React application.  This is evident from the presence of `.tsx`  files (`AccentButton.tsx`, `Header.tsx`, `App.tsx`),
    [collaboration_discussion] SoftwareArchitect: The workspace context indicates a JavaScript/TypeScript project using  React. It includes standard React components and styling  files. The use of TypeScript fi
    [collaboration_discussion] PlatformEngineer: The workspace is primarily a React application. This  is evident from several key indicators: 1. The  presence of React-specific files and directories such as  
    [collaboration_discussion] PlatformEngineer: 
  --- end ---
```

### collab:planning-two-agent [code]
```text
tioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: I agree. Here's my perspective on the collaboration:  - **@Assistant's Role**: Define the scope and structure  of the investigation. This will ensure we have  a
    [collaboration_discussion] Assistant: - **Task 1: @Assistant - Define Scope for  API Schema Standardization** Write `collabs/<id>/scope.md` outlining required schema  fields, versioning strategies (
    [collaboration_discussion] FrontendEngineer: Task 1: Define Scope for API Schema Standardization  - This task is crucial for setting clear  expectations and guidelines for the schema. The scope  should out
    [collaboration_discussion] FrontendEngineer: The project is primarily written in JavaScript/TypeScript and  uses React for building the user interface. The  presence of files like `App.tsx`, `Header.tsx`, 
    [collaboration_discussion] FrontendEngineer: This workspace is a React application, utilizing TypeScript  and CSS for styling. Key files include React  components (`AccentButton.tsx`, `Header.tsx`), a them
    [collaboration_discussion] Assistant: Hi Camron! 👋 Here's my planning perspective on  this collaboration: For Task 1 (Define Scope), we  should focus on three key areas: **inventory mapping**  (iden
    [collaboration_discussion] Assistant: Based on your file tree showing Workbox PWA  tooling (caching strategies, service workers), React components (`Header.tsx`),  and a `workbox-webpack-plugin` con
  --- end ---

agent discussion: total=7 counts={'FrontendEngineer': 4, 'Assistant': 3}
  ok: @Assistant — 3 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @FrontendEngineer — 4 message(s)
  system turn handoffs in channel: 7
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=3/10
```

### collab:resource-api-schema-regression [code]
```text
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: ### Task 1: Define Scope (@Assistant) with Deliverable  collabs/d914f9be-ec30-4bb4-ba83-2a02f60748a1/scope.md - This task will outline the goals,  objectives, a
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] Assistant: 📋 **Assistant's Planning Perspective:** From a coordination standpoint,  I recommend we start with Task 1 (Scope  Definition) first before proceeding to review 
  --- end ---

agent discussion: total=3 counts={'BackendEngineer': 1, 'Assistant': 2}
  generation_error posts in channel: 1
  ok: @Assistant — 2 message(s)
  ok: @BackendEngineer — 1 message(s)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/10
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-1158-iter2.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
- python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website-sa
- python3 scripts/collab-scenarios.py --scenario execution-no-stack-commands
- python3 scripts/collab-scenarios.py --scenario make-me-a-website
- python3 scripts/collab-scenarios.py --scenario phoenix-resource-api-e2e
- python3 scripts/collab-scenarios.py --scenario plan-dependency-prose-regression
- python3 scripts/collab-scenarios.py --scenario plan-distinct-deliverables-same-agent
- python3 scripts/collab-scenarios.py --scenario plan-findings-task-regression
- python3 scripts/collab-scenarios.py --scenario plan-phoenix-combined-regression
- python3 scripts/collab-scenarios.py --scenario planning-two-agent
- python3 scripts/collab-scenarios.py --scenario resource-api-schema-regression
- python3 scripts/collab-scenarios.py --scenario solo-vs-collab-parity

