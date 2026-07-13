You are fixing failures from a Neural Junkie release-prep test run.

Rules (mandatory):
- Triage product/hub/agent behavior first, harness second (docs/TESTING.md).
- Do NOT weaken test assertions or scenario contracts to greenwash flakes.
- Prefer minimal, focused fixes in the neural-junkie repo.
- After edits, run the targeted verification commands listed below.
- Summarize what you changed and which commands you ran.

Release prep summary: docs/testing/release-prep-2026-06-27-2204.md
Failed phases: test-everything, test-parity-stable-restart, model-benchmark

## Failures to address

### test-all [code]
```text
🔍 go vet...
# github.com/camronwood/neural-junkie/test
# [github.com/camronwood/neural-junkie/test]
vet: test/agent_review_test.go:86:7: undefined: m
make[2]: *** [test-all] Error 1
```

### test-conversation-contract [code]
```text
🧪 Agent conversation routing...
ok  	github.com/camronwood/neural-junkie/internal/agent	0.469s
🧪 Hub conversation/collab wiring...
ok  	github.com/camronwood/neural-junkie/internal/hub	0.559s
ok  	github.com/camronwood/neural-junkie/cmd/server	0.372s
🧪 Collab API smoke...
# github.com/camronwood/neural-junkie/test [github.com/camronwood/neural-junkie/test.test]
test/agent_review_test.go:86:7: undefined: m
test/agent_review_test.go:87:7: undefined: m
test/assistant_test.go:485:7: undefined: m
test/assistant_test.go:486:7: undefined: m
test/deduplication_test.go:59:7: undefined: m
test/deduplication_test.go:60:7: undefined: m
test/design_analysis_test.go:52:7: undefined: m
test/design_analysis_test.go:53:7: undefined: m
test/integration_test.go:82:7: undefined: m
test/integration_test.go:83:7: undefined: m
test/integration_test.go:83:7: too many errors
FAIL	github.com/camronwood/neural-junkie/test [build failed]
FAIL
make[2]: *** [test-conversation-contract] Error 1
```

### collab-smoke [code]
```text
collab-smoke: running in-process API test (go test -run TestCollabSmokePhaseTransitions)
# github.com/camronwood/neural-junkie/test [github.com/camronwood/neural-junkie/test.test]
test/agent_review_test.go:86:7: undefined: m
test/agent_review_test.go:87:7: undefined: m
test/assistant_test.go:485:7: undefined: m
test/assistant_test.go:486:7: undefined: m
test/deduplication_test.go:59:7: undefined: m
test/deduplication_test.go:60:7: undefined: m
test/design_analysis_test.go:52:7: undefined: m
test/design_analysis_test.go:53:7: undefined: m
test/integration_test.go:82:7: undefined: m
test/integration_test.go:83:7: undefined: m
test/integration_test.go:83:7: too many errors
FAIL	github.com/camronwood/neural-junkie/test [build failed]
FAIL
make[2]: *** [collab-smoke] Error 1
```

### chat-scenarios-regression [unknown]
```text
nd: can you see my workspace I have open?
  ✓ [2] wait_reply: SoftwareArchitect replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-architect-workspace ===


=== scenario: dm-assistant-continue-after-closure ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-assistant agent=Assistant
  ✓ [1] send: In one short paragraph: how would you add a light/dark theme toggle in a React s…
  ✓ [2] wait_reply: Assistant replied (1 new)
  ✓ [3] send: ok thanks
  ✓ [4] wait_reply: Assistant replied (1 new)
  ✓ [5] send: One more thing — where should the theme toggle live in the settings UI?
  ✓ [6] wait_reply: Assistant replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-assistant-continue-after-closure ===


=== scenario: dm-backend-codebase-semantic ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: @codebase What does ComputePhoenixWidget return?
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-backend-codebase-semantic ===


=== scenario: dm-backend-deep-continuation ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: I want to add light and dark theme support to this project
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: go deeper on the approach — what would you implement first?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✗ [5] assert_messages: any_match not found: 'theme|dark|light|toggle|config|CSS|provider' (agents: ['BackendEngineer'])
  --- transcript (last messages) ---
    [question] camronwood: I want to add light and dark theme support to this project
    [chat] BackendEngineer: Grounding: I loaded 2 file(s) from the workspace  context for this answer. [FILE_CHANGE] --- path: src/theme.css  ---
```

### collab-scenario-regression [unknown]
```text
Engineer', 'SoftwareArchitect', 'PlatformEngineer'] >= 1): counts={'BackendEngineer': 4}
agent discussion: total=4 counts={'BackendEngineer': 4}
  ok: @BackendEngineer — 4 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 2
  pending file changes (hub): 4
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/4

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: - Task 1: @BackendEngineer - Write collabs/dd8df103-ce3e-46ff-b624-25e8385f8373/api_schema.md defining  minimal Go API contracts for `core/sample/main.go` handl
    [collaboration_discussion] BackendEngineer: ## Planning Perspective @BackendEngineer The current 3-task plan  is lean and sensible, but I see two  adjustments worth making: 1. **Task order matters**: api_
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect @PlatformEngineer — The current 3-task plan aligns  well with lanes. Two dependency notes warrant clarification:  - Task 1 (`api_schema.md`) 
    [collaboration_discussion] BackendEngineer: ### 📋 Collaboration Session Recap **Goal:** Produce a  coordinated 3-file development plan under `collabs/dd8df103-ce3e-46ff-b624-25e8385f8373/` --- ##  ✅ Decis
  --- end ---

agent discussion: total=4 counts={'BackendEngineer': 4}
  ok: @BackendEngineer — 4 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 2
  pending file changes (hub): 4
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/4
=== FAIL: plan-dependency-prose-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[2]: *** [collab-scenario-regression] Error 1
```

## Likely flakes/infra (retry only — do not hack code to pass)

test-everything, test-parity-stable-restart, implement-scenarios, conversation-scenarios-regression, collab-scenarios-all

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/test-everything-2026-06-27-2204.md
- /Users/camronwood/development/projects/neural-junkie/docs/testing/test-everything-2026-06-27-2204.log
- /Users/camronwood/development/projects/neural-junkie/docs/testing/parity-stable-restart-2026-06-28-0454.log
- /Users/camronwood/development/projects/neural-junkie/docs/testing/model-benchmark-quick-2026-06-28-1413.md
- /Users/camronwood/development/projects/neural-junkie/docs/testing/model-benchmark-quick-2026-06-28-1413.json
- /Users/camronwood/development/projects/neural-junkie/docs/testing/model-benchmark-quick-2026-06-28-1413.tsv

## Targeted verification (run after your fixes)
- make test-all
- make test-conversation-contract
- make collab-smoke
- python3 scripts/implement-scenarios-stable.py --runs 1 --min-pass 20 --restart-between --hub http://127.0.0.1:18765
- python3 scripts/chat-scenarios.py --all --tag regression
- python3 scripts/conversation-scenarios-regression.py

