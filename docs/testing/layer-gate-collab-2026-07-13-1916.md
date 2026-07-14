# Layer gate — collab — 2026-07-13-1916 UTC

layer=collab
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenario-regression` | FAIL | 184s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-2026-07-13-1916.log`

## Failures (tail)

### collab-scenario-regression (exit 2)

```text
>>> make collab-scenario-regression


=== Regression boot (collab-scenarios) ===
>>> Stopping Neural Junkie processes...
>>> Restoring scenario fixtures from git...
>>> Removing fixture collab runtime dirs...
OK: Ollama (http://127.0.0.1:11434/api/tags)
>>> Warming Ollama models (suite=release)...
>>> Ollama model readiness
  warm: qwen3.5:9b, qwen2.5-coder:14b, qwen2.5:3b
  pull roster: qwen3.5:9b, qwen2.5-coder:14b, qwen2.5:3b, gemma3:12b
  installed: qwen3.5:9b
  installed: qwen2.5-coder:14b
  installed: qwen2.5:3b
  installed: gemma3:12b
  warming qwen3.5:9b …
  OK warm qwen3.5:9b: loaded in 15s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 10s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 2s (keep_alive=24h)
  smoke qwen3.5:9b …
  OK smoke qwen3.5:9b: ok
  smoke qwen2.5-coder:14b …
  OK smoke qwen2.5-coder:14b: Ok
  smoke qwen2.5:3b …
  OK smoke qwen2.5:3b: ok
  loaded now: qwen2.5:3b, qwen3.5:9b
OK: Ollama models ready
>>> Starting regression hub (in-process specialists)...
OK: hub healthy at http://127.0.0.1:18765
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): Codex, Copilot, CodeReviewer, DatabaseSpecialist, DataMLEngineer, Swift-Development-iOSDeveloper, Gemini, SwitchTarget…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: collab-minimal-completion-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 737e1059 → collab-737e1059-d1f2-45dd-ad55-9ee4405f40c9
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/737e1059-d1f2-45dd-ad55-9ee4405f40c9
  ✓ [10] send: /resume-plan 737e1059-d1f2-45dd-ad55-9ee4405f40c9
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✗ [13] assert_files: file collabs/737e1059-d1f2-45dd-ad55-9ee4405f40c9/findings.md none_match 'React'
--- file snippet (first 20 lines) ---
# findings.md

- **README.md** provides a brief description of the project, stating it is a minimal fixture repo used by `execute-deliverable` and `reject-collabs-subfolder` scenarios. It does not contain detailed information about the code or its functionality.

- **core/sample/main.go** is a simple Go program that defines a function `HelloWorld`, which prints "Hello, World!" to the console. The `main` function calls `HelloWorld`.

- The project structure includes a mix of directories and files, with the core logic located in the `core` directory and a `README` file at the root. The `collabs` directory is used for collaboration-related files, and the `src` directory contains what appears to be frontend code, possibly for a React application.
  ✓ cleanup: cancelled and removed workspace artifacts
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: I agree with Task 1. Before we finalize the plan and execute, I need to validate the core assumptions from the reference files to ensure the summary is accurate
    [collaboration_discussion] SoftwareArchitect: Task 1 is ready to execute. I will have @BackendEngineer write the findings document summarizing the minimal fixture repo based strictly on `README.md` and `cor
    [collaboration_discussion] SoftwareArchitect: 
    [collaboration_discussion] SoftwareArchitect: The plan looks good to me. I will proceed to execute Task 1 immediately.  **Task 1 Execution:** @BackendEngineer - Write `collabs/737e1059-d1f2-45dd-ad55-9ee440
    [collaboration_discussion] SoftwareArchitect: The plan is incomplete. The user provided a high-level view of the project, but I need to inspect the specific files mentioned in the task goal (`README.md` and
    [chat] cea0519f-cc2c-4c12-9b08-4e3f7bf8926a: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/737e1059-d1f2-45dd-ad55-9
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/737e1059-d1f2-45dd-ad55-9ee4405f40c9/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/737e1059-d1f2-45dd-ad55-9ee4405f40c9/findings.md).  Verification skipped
  --- end ---

=== FAIL: collab-minimal-completion-regression ===

make[1]: *** [collab-scenario-regression] Error 1

RESULT collab-scenario-regression: FAIL (exit 2, 184s)
```

