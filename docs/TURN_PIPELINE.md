# Turn pipeline

Neural Junkie agent turns run through an explicit composable pipeline (`internal/pipeline`) with span-level tracing (`internal/trace`). This replaces the implicit stage ordering that previously lived only in `handleMessage`.

## Default turn steps

```
prepare_turn
  → intent_classify
  → knowledge_plan
  → knowledge_execute
  → governance_record
  → provider_route
  → generate
  → post_process
  → stamp_metadata
  → deliver_response
```

Entry point: [`runTurnPipeline`](../internal/agent/turn_pipeline.go) from [`handleMessage`](../internal/agent/agent_message.go).

Each step maps 1:1 to a trace span name for debugging and layer-gate rubrics.

## Knowledge layer

Planning: [`PlanKnowledgeRoute` / `PlanKnowledgeRouteForTurn`](../internal/routing/knowledge_router.go)

Execution: [`ExecuteKnowledgePlan`](../internal/agent/knowledge_retrievers.go) registry:

| Retriever | Phase | Route target |
|-----------|-------|----------------|
| codebase | early | `codebase` |
| memory | prompt | `conversation_memory` / `collab_artifact` |
| learnings | prompt | `learning` |
| prior_reference | response | `prior_reference` |
| repo_consult | response | `codebase` (consult path) |

Casual/closure turns skip default conversation-memory vector search via intent gating.

**Dialogue window:** Before knowledge overlays, `conversationHistoryForIntent` always builds a protected ConversationWindow of recent complete user↔assistant exchanges (`recentCompleteExchanges`). Session summary and durable state are overlays — see [CONTEXT_MODEL.md](CONTEXT_MODEL.md).

## Observability

- **Live:** Turn telemetry drawer (`routing` + `tool` events)
- **Persisted:** `trace_id`, `trace_spans`, `tool_steps` on agent response metadata
- **Files:** `~/.neural-junkie/traces/{trace_id}.json`
- **API:** `GET /api/debug/turn-trace?channel=&message_id=`
- **UI:** Routing trace panel shows span tree + timings

Classifier fields (`routing_classifier_*`) are stamped alongside routing badges.

## Workflow durability (multi-agent)

- **Event log:** `~/.neural-junkie/workflow-events/{collab_id}.jsonl`
- **Dispatch tokens:** `dispatch_token` on `collaboration_task` messages
- **Hub timeout:** idle in-progress tasks → `blocked` after redispatch budget + 5m
- **Impl checkpoints:** keyed by `collab_id:task_id` when present

## Schema boundaries

[`internal/schema`](../internal/schema/validate.go) validates tool args and collab task IR at parse boundaries (`ask_user`, MCP tools, `ParsePlanTasks`).

## Layer-gate trace rubrics

After a scenario reply, assert via turn-trace API:

```bash
curl -s "http://127.0.0.1:18765/api/debug/turn-trace?channel=dm-backend&message_id=<user-msg-id>" \
  | jq '.spans[].name, .tool_steps | length, .retrieval.executed'
```

Expected chat/codebase scenario (`scenarios/chat/dm-backend-codebase-semantic.json`):

- At least one `knowledge_execute.codebase` span when repo context is injected
- `retrieval.executed` includes `codebase`
- `tool_steps` present when tools ran (server-persisted, not client-only)

## Related docs

- [CONTEXT_MODEL.md](CONTEXT_MODEL.md) — prompt/context stack
- [COLLABORATION.md](COLLABORATION.md) — DAG + workflow
- [ADAPTIVE-ORCHESTRATION-NOTES.md](ADAPTIVE-ORCHESTRATION-NOTES.md) — enterprise mapping (updated for unified knowledge executor)
