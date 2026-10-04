# Layer gate — user-flows — 2026-09-30-1104 UTC

layer=user-flows
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `user-flow-scenarios` | FAIL | 21601s | 124 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-user-flows-2026-09-30-1104.log`

## Failures (tail)

### user-flow-scenarios (exit 124)

```text
er:14b, qwen3.5:9b, qwen2.5:3b, gemma3:12b
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 4s (keep_alive=24h)
  warming qwen3.5:9b …
  OK warm qwen3.5:9b: loaded in 4s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 3s (keep_alive=24h)
  smoke qwen2.5-coder:14b …
  OK smoke qwen2.5-coder:14b: ok
  smoke qwen3.5:9b …
  OK smoke qwen3.5:9b: ok
  smoke qwen2.5:3b …
  OK smoke qwen2.5:3b: ok
  re-warm qwen2.5-coder:14b (primary agent model) …
  OK re-warm qwen2.5-coder:14b: loaded in 4s (keep_alive=24h)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5:3b)
  loaded now: qwen2.5-coder:14b, qwen3.5:9b
OK: Ollama models ready
>>> Starting regression hub (in-process specialists)...
OK: hub healthy at http://127.0.0.1:18765
>>> Enabling regression roster (product default is slim)...
OK: roster already enabled
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen2.5-coder:14b (Switched 28 agents to ollama (qwen2.5-coder:14b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen2.5-coder:14b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [implement-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (implement-scenarios) ===


=== implement: journey-boot-fix-then-feature ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from SoftwareArchitect (metadata: implementation_session_outcome; absent:src/App.js; match:src/App.tsx)
  ✓ [3] send: sent
  ✓ [4] wait_reply: reply from SoftwareArchitect
  ✓ [5] send: sent
  ✓ [6] wait_reply: reply from SoftwareArchitect (metadata: implementation_session_outcome; absent:src/App.js; match:src/App.tsx)
  ✓ [7] send: sent
  wait_reply: nudged silent @SoftwareArchitect
  ✓ [8] wait_reply: reply from SoftwareArchitect (metadata: implementation_session_outcome; absent:src/App.js; match:src/App.tsx)
  ✓ [9] assert_file_absent: src/App.js absent
  ✓ [10] assert_file_exists: src/App.tsx
=== PASS: journey-boot-fix-then-feature ===

EVAL_JSON:{"schema_version":1,"kind":"implement","scenario":"journey-boot-fix-then-feature","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":["silent agent after 660s"],"actual_provider":"qwen2.5-coder:14b","actual_model":"qwen2.5-coder:14b","validation_failures":["implementation verification failed"],"escalation_reasons":[],"repair_attempts":3,"tool_calls":null,"ttft_ms":4996.840624,"retry_count":0,"nudge_count":1,"escalation_count":0,"wall_duration_ms":894983.095}
METRICS_JSON:{"circuit_breaker_triggered":false,"files_changed":["src/App.js"],"outcome":"proposals_submitted","repair_attempts":3,"repair_used":true,"routing_reason":"semantic_turn_decision","verify_failed":true,"verify_skipped":false,"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":1,"nudge_reasons":["silent agent after 660s"],"actual_provider":"qwen2.5-coder:14b","actual_model":"qwen2.5-coder:14b","validation_failures":["implementation verification failed"],"escalation_count":0,"escalation_reasons":[],"tool_calls":null,"ttft_ms":4996.840624,"wall_duration_ms":894983.095}

=== user-flow [implement/user-flows]: journey-notes-rename-to-memos ===
    Notes CRUD → rename to /memos mid-session → scrub /notes

>>> python3 scripts/implement-scenarios.py --scenario journey-notes-rename-to-memos --hub http://127.0.0.1:18765

=== Regression boot (implement-scenarios) ===
>>> Stopping Neural Junkie processes...
>>> Restoring scenario fixtures from git...
  removed orphan /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/react-vite-corrupt-appjs/src/App.js
>>> Removing fixture collab runtime dirs...
>>> Restored fixture .scenario-baseline seeds
OK: Ollama (http://127.0.0.1:11434/api/tags)
>>> Warming Ollama models (suite=release)...
>>> Ollama model readiness
  warm: qwen2.5-coder:14b, qwen3.5:9b, qwen2.5:3b
  pull roster: qwen2.5-coder:14b, qwen3.5:9b, qwen2.5:3b, gemma3:12b
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 4s (keep_alive=24h)
  warming qwen3.5:9b …
  OK warm qwen3.5:9b: loaded in 1s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 2s (keep_alive=24h)
  smoke qwen2.5-coder:14b …
  OK smoke qwen2.5-coder:14b: Ok
  smoke qwen3.5:9b …
  OK smoke qwen3.5:9b: ok
  smoke qwen2.5:3b …
  OK smoke qwen2.5:3b: ok
  re-warm qwen2.5-coder:14b (primary agent model) …
  OK re-warm qwen2.5-coder:14b: loaded in 4s (keep_alive=24h)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5:3b)
  loaded now: qwen2.5-coder:14b, qwen3.5:9b
OK: Ollama models ready
>>> Starting regression hub (in-process specialists)...
OK: hub healthy at http://127.0.0.1:18765
>>> Enabling regression roster (product default is slim)...
OK: roster already enabled
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen2.5-coder:14b (Switched 28 agents to ollama (qwen2.5-coder:14b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen2.5-coder:14b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 1 pending file change(s)

>>> [implement-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (implement-scenarios) ===


=== implement: journey-notes-rename-to-memos ===
  ✓ [1] send: sent
  wait_reply: nudged silent @BackendEngineer
  ✗ [2] wait_reply: timeout waiting for BackendEngineer
=== FAIL: journey-notes-rename-to-memos ===


>>> flake retry 2/2 for journey-notes-rename-to-memos: timeout waiting for BackendEngineer

=== implement: journey-notes-rename-to-memos ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from BackendEngineer (metadata: implementation_session_outcome; exists:package.json; match:package.json)
  ✓ [3] send: sent
  ✓ [4] wait_reply: reply from BackendEngineer (metadata: implementation_session_outcome; match:src/server.ts)
  ✓ [5] send: sent
  ✓ [6] wait_reply: reply from BackendEngineer (metadata: implementation_session_outcome; match:src/server.ts)
  ✓ [7] send: sent
  ✓ [8] wait_reply: reply from BackendEngineer (metadata: implementation_session_outcome; match:src/server.ts)
  ✓ [9] assert_messages: message assertions ok
  ✓ [10] assert_file_exists: package.json
  ✓ [11] assert_file_exists: src/server.ts
=== PASS: journey-notes-rename-to-memos ===

EVAL_JSON:{"schema_version":1,"kind":"implement","scenario":"journey-notes-rename-to-memos","attempts":2,"passed_at_1":false,"eventual_pass":true,"retry_reasons":["timeout waiting for BackendEngineer"],"nudge_reasons":["silent agent after 660s"],"actual_provider":"qwen2.5-coder:14b","actual_model":"qwen2.5-coder:14b","validation_failures":[],"escalation_reasons":[],"repair_attempts":3,"tool_calls":null,"ttft_ms":16468.643666,"retry_count":1,"nudge_count":1,"escalation_count":0,"wall_duration_ms":1460580.354}
METRICS_JSON:{"circuit_breaker_triggered":false,"completion_tokens":495,"files_changed":["src/server.ts"],"inference_usage":{"calls":1,"completion_tokens":495,"prompt_tokens":3326,"tok_per_s":21.630709843669134,"ttft_ms":16468.643666},"outcome":"proposals_submitted","prompt_tokens":3326,"repair_attempts":3,"repair_used":true,"routing_reason":"semantic_turn_decision","tok_per_s":21.630709843669134,"ttft_ms":16468.643666,"verify_failed":false,"verify_skipped":true,"passed_at_1":false,"eventual_pass":true,"attempts":2,"retry_count":1,"retry_reasons":["timeout waiting for BackendEngineer"],"nudge_count":1,"nudge_reasons":["silent agent after 660s"],"actual_provider":"qwen2.5-coder:14b","actual_model":"qwen2.5-coder:14b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"tool_calls":null,"wall_duration_ms":1460580.354}

=== user-flow [implement/user-flows]: journey-landing-brand-correction ===
    Landing page → brand rename → tagline → finish

>>> python3 scripts/implement-scenarios.py --scenario journey-landing-brand-correction --hub http://127.0.0.1:18765

=== Regression boot (implement-scenarios) ===
>>> Stopping Neural Junkie processes...
>>> Restoring scenario fixtures from git...
  removed orphan /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/react-vite-corrupt-appjs/src/App.js
>>> Removing fixture collab runtime dirs...
>>> Restored fixture .scenario-baseline seeds
OK: Ollama (http://127.0.0.1:11434/api/tags)
>>> Warming Ollama models (suite=release)...
>>> Ollama model readiness
  warm: qwen2.5-coder:14b, qwen3.5:9b, qwen2.5:3b
  pull roster: qwen2.5-coder:14b, qwen3.5:9b, qwen2.5:3b, gemma3:12b
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 2s (keep_alive=24h)
  warming qwen3.5:9b …
  OK warm qwen3.5:9b: loaded in 1s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 2s (keep_alive=24h)
  smoke qwen2.5-coder:14b …
  OK smoke qwen2.5-coder:14b: ok
  smoke qwen3.5:9b …
  OK smoke qwen3.5:9b: ok
  smoke qwen2.5:3b …
  OK smoke qwen2.5:3b: ok
  re-warm qwen2.5-coder:14b (primary agent model) …
  OK re-warm qwen2.5-coder:14b: loaded in 4s (keep_alive=24h)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5:3b)
  loaded now: qwen2.5-coder:14b, qwen3.5:9b
OK: Ollama models ready
>>> Starting regression hub (in-process specialists)...
OK: hub healthy at http://127.0.0.1:18765
>>> Enabling regression roster (product default is slim)...
OK: roster already enabled
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen2.5-coder:14b (Switched 28 agents to ollama (qwen2.5-coder:14b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen2.5-coder:14b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [implement-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (implement-scenarios) ===


=== implement: journey-landing-brand-correction ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from FrontendEngineer (metadata: implementation_session_outcome; match:index.html)
  ✓ [3] send: sent
  wait_reply: auto-approved 1 file change(s) ids=['cda4bab4']
  ✓ [4] wait_reply: reply from FrontendEngineer (metadata: implementation_session_outcome; match:index.html)
  ✓ [5] send: sent
  wait_reply: nudged silent @FrontendEngineer
  ✗ [6] wait_reply: timeout waiting for FrontendEngineer
=== FAIL: journey-landing-brand-correction ===


>>> flake retry 2/2 for journey-landing-brand-correction: timeout waiting for FrontendEngineer

=== implement: journey-landing-brand-correction ===
  ✓ [1] send: sent
  wait_reply: auto-approved 1 file change(s) ids=['650d58dc']
  wait_reply: auto-approved 1 file change(s) ids=['2fafc9f2']
  ✓ [2] wait_reply: reply from FrontendEngineer (metadata: implementation_session_outcome; match:index.html)
  ✓ [3] send: sent
  wait_reply: nudged silent @FrontendEngineer
  wait_reply: auto-approved 1 file change(s) ids=['77c4abec']
  ✓ [4] wait_reply: reply from FrontendEngineer (metadata: implementation_session_outcome; match:index.html)
  ✓ [5] send: sent

[layer-gate] STAGE TIMEOUT after 21600s — killed process tree

RESULT user-flow-scenarios: FAIL (exit 124, 21601s)
```

