# Layer gate — chat — 2026-07-30-0104 UTC

layer=chat
hub=http://127.0.0.1:18765
Overall: **FAIL** (1/2 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `chat-scenarios-canary` | OK | 553s | 0 |
| `conversation-scenarios-regression` | FAIL | 1639s | 1 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-chat-2026-07-30-0104.log`

## Failures (tail)

### conversation-scenarios-regression (exit 1)

```text
udge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":38955.184}

>>> python3 scripts/chat-scenarios.py --scenario dm-backend-interject-resume

=== scenario: dm-backend-interject-resume ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: What does the main function in @file:core/sample/main.go do?
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] channel_interject: channel 'dm-chatscenario-backendengineer' held
  ✓ [4] wait_no_reply: no new replies from @BackendEngineer for 8s (baseline=1)
  ✓ [5] send: What Go package declaration is at the top of that file?
  ✓ [6] wait_reply: BackendEngineer replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ [8] assert_reply_count: reply count since baseline=1
  ✓ cleanup: cleared channel history
=== PASS: dm-backend-interject-resume ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-backend-interject-resume","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":59848.47}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":59848.47}

>>> python3 scripts/chat-scenarios.py --scenario dm-assistant-trip-followup

=== scenario: dm-assistant-trip-followup ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-assistant agent=Assistant
  ✓ [1] send: Help me plan a 4-day trip to Lisbon with a food-focused itinerary.
  ✓ [2] wait_reply: Assistant replied (1 new)
  ✓ [3] send: I have turned on websearch now
  ✓ [4] wait_reply: Assistant replied (1 new)
  ✓ [5] send: What neighborhoods should we stay in for that food itinerary?
  ✓ [6] wait_reply: Assistant replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-assistant-trip-followup ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-assistant-trip-followup","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":103719.741}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":103719.741}

>>> python3 scripts/chat-scenarios.py --scenario dm-pronoun-followup-3turn

=== scenario: dm-pronoun-followup-3turn ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-frontendengineer agent=FrontendEngineer
  ✓ [1] send: Design a ThemeSettings component for dark/light mode.
  ✓ [2] wait_reply: FrontendEngineer replied (1 new)
  ✓ [3] send: Add keyboard accessibility for it.
  ✓ [4] wait_reply: FrontendEngineer replied (1 new)
  ✓ [5] send: Move it under an Appearance section in settings.
  ✓ [6] wait_reply: FrontendEngineer replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ [8] assert_transcript_metrics: correction_latency_turns=0.000, correction_recovery_rate=1.000, direct_answer_rate=1.000, edit_precision_rate=1.000, instruction_retention_rate=1.000, repeated_question_rate=1.000, stale_plan_rate=0.000, tool_follow_through_rate=1.000, truthful_completion_rate=1.000, unsupported_claim_rate=1.000
  ✓ cleanup: cleared channel history
=== PASS: dm-pronoun-followup-3turn ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-pronoun-followup-3turn","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":2,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":91481.082}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":2,"ttft_ms":null,"wall_duration_ms":91481.082}

>>> python3 scripts/chat-scenarios.py --scenario dm-chat-mode-soft-followups

=== scenario: dm-chat-mode-soft-followups ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: What do you think about Go versus Rust for backend services?
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: why?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✓ [5] send: one more tradeoff please
  ✓ [6] wait_reply: BackendEngineer replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ [8] assert_messages: semantic_turn_decision ok
  ✓ cleanup: cleared channel history
=== PASS: dm-chat-mode-soft-followups ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-chat-mode-soft-followups","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":56181.738}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":56181.738}

>>> python3 scripts/chat-scenarios.py --scenario dm-topic-continuity-same-thread

=== scenario: dm-topic-continuity-same-thread ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-assistant agent=Assistant
  ✓ [1] send: I want a simple weekend meal-prep plan for two people.
  ✓ [2] wait_reply: Assistant replied (1 new)
  ✓ [3] send: Make it vegetarian.
  ✓ [4] wait_reply: Assistant replied (1 new)
  ✓ [5] send: What about Sunday lunch?
  ✓ [6] wait_reply: Assistant replied (1 new)
  ✓ [7] send: Summarize the full weekend plan with that Sunday lunch.
  ✓ [8] wait_reply: Assistant replied (1 new)
  ✓ [9] assert_transcript_metrics: correction_latency_turns=0.000, correction_recovery_rate=1.000, direct_answer_rate=1.000, edit_precision_rate=1.000, instruction_retention_rate=1.000, repeated_question_rate=0.000, stale_plan_rate=0.000, tool_follow_through_rate=1.000, truthful_completion_rate=1.000, unsupported_claim_rate=1.000
  ✓ cleanup: cleared channel history
=== PASS: dm-topic-continuity-same-thread ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-topic-continuity-same-thread","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":82081.327}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":82081.327}

>>> python3 scripts/chat-scenarios.py --scenario dm-summary-continuity-long-horizon

=== scenario: dm-summary-continuity-long-horizon ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-assistant agent=Assistant
  ✓ [1] send: Plan a reading club. Constraint: books must be under 250 pages.
  ✓ [2] wait_reply: Assistant replied (1 new)
  ✓ [3] send: Suggest a fiction pick for month one.
  ✓ [4] wait_reply: Assistant replied (1 new)
  ✓ [5] send: Now a nonfiction pick for month two.
  ✓ [6] wait_reply: Assistant replied (1 new)
  ✓ [7] send: How should we structure the discussion questions?
  ✓ [8] wait_reply: Assistant replied (1 new)
  ✓ [9] send: Remind me of the page-count constraint and list both picks.
  ✓ [10] wait_reply: Assistant replied (1 new)
  ✓ [11] assert_transcript_metrics: correction_latency_turns=0.000, correction_recovery_rate=1.000, direct_answer_rate=1.000, edit_precision_rate=1.000, instruction_retention_rate=1.000, repeated_question_rate=1.000, stale_plan_rate=0.000, tool_follow_through_rate=1.000, truthful_completion_rate=1.000, unsupported_claim_rate=1.000
  ✓ cleanup: cleared channel history
=== PASS: dm-summary-continuity-long-horizon ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-summary-continuity-long-horizon","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":2,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":121212.545}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":2,"ttft_ms":null,"wall_duration_ms":121212.545}

>>> python3 scripts/chat-scenarios.py --scenario dm-durable-state-chat-isolation

=== scenario: dm-durable-state-chat-isolation ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: Review internal/hub/hub.go for error handling gaps at a high level.
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: unrelated: what do you think about Go versus Rust for side projects?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✓ [5] send: one more thought on learning curve?
  ✓ [6] wait_reply: BackendEngineer replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ [8] assert_messages: semantic_turn_decision ok
  ✓ cleanup: cleared channel history
=== PASS: dm-durable-state-chat-isolation ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-durable-state-chat-isolation","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":79815.391}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":79815.391}

=== Summary ===
PASS 21/23
FAILED: chat:dm-code-reviewer-workspace, chat:public-frontend-theme-continuation

RESULT conversation-scenarios-regression: FAIL (exit 1, 1639s)
```

