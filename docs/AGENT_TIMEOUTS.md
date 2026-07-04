# Agent Timeouts And Waits

This document collects the main timeout, retry, debounce, polling, and TTL values involved when working with agents in `neural-junkie`.

It is focused on:

- agent generation deadlines
- collaboration orchestration waits
- implementation-session limits
- user approval / question blocking windows
- agent process and UI polling relevant to agent behavior

## LLM Generation And Provider Limits

| Duration | Area | Waiting for |
| --- | --- | --- |
| No explicit deadline | Regular chat/DM agent replies | Normal agent replies; only provider-level limits apply |
| 120s | Ollama default HTTP timeout | A single Ollama inference call |
| 360s | Ollama thinking-model timeout | Reasoning models such as `deepseek-r1` and `qwen3` thinking variants |
| 360s | Ollama collab-heavy model timeout | Slower collaboration-oriented local models such as `qwen3.5`, `qwen2.5-coder`, `gemma3`, `codestral`, `devstral` |
| 600s | Ollama large-model timeout | Large local models such as `qwen3.5:27b`, `:35b`, `:122b` |
| 300s default, 300s max cap | CLI provider subprocess timeout | A single CLI-backed provider invocation such as `cursor-cli` or `gemini-cli` |
| Configurable via `ai.providers[].timeout_seconds` | Provider config | Per-provider CLI invocation override |
| 15s | LLM classifier | LLM-based routing/classification |

## Collaboration Generation Deadlines

These are applied through collaboration generation contexts when the agent is handling collaboration message types.

| Duration | Message type | Waiting for |
| --- | --- | --- |
| 180s | Collaboration task, chat/non-file | Agent to finish a collaboration chat task |
| 300s | Collaboration task, file deliverable | Agent to produce file deliverables |
| Configurable via `collaboration.execution_timeout_seconds` | Collaboration config | Override for file-deliverable collaboration tasks |
| 240s | Collaboration recap | Agent to produce pre-approval or final recap |
| 480s | Collaboration discussion | Agent turn during planning, reviewing, or executing discussion |

## Collaboration Orchestration Waits

| Duration | Area | Waiting for |
| --- | --- | --- |
| 240s | Recap timeout | Hub waiting for recap response before using fallback behavior |
| 5m default | Collaboration discussion timeout | Full discussion session wall-clock limit |
| 30m hard max | Collaboration discussion timeout cap | Upper bound on configured discussion timeout |
| 30s | Collaboration idle watchdog tick | Periodic scan for stuck or idle collaboration execution |
| 90s | Idle task redispatch threshold | In-progress task with no update before redispatch |
| 2 retries max | Idle task redispatch limit | Maximum watchdog redispatch attempts |
| 45s | Turn handoff retry delay | Wait before retrying a collaboration handoff |
| 2 retries max | Turn handoff retry limit | Maximum handoff retry attempts |

## Implementation Session Limits

| Duration | Area | Waiting for |
| --- | --- | --- |
| 480s | Implementation session v1 | Full implementation session for non-frontend flows |
| 600s | Frontend implementation session v1 | Full implementation session for frontend flows |
| 60m default | Agent Runtime v2 | Runtime wall-clock limit when `agent_runtime_v2` is enabled |
| 180m max | Agent Runtime v2 config cap | Upper bound for configured runtime timeout |
| 120s | Verify command timeout | Verification command during implementation session |

## Tool And Command Execution Waits

| Duration | Area | Waiting for |
| --- | --- | --- |
| 60s | `run_command` default timeout | Shell/tool command execution during agent work |
| 30s | Boot-fix command timeout | Boot-fix diagnostic command execution |
| 3m | Boot-fix npm install timeout | `npm install` during boot-fix flow |
| 15s | Quick repro timeout | Fast reproduction/diagnostic command |
| 120s default | CAD render timeout | CAD render via MCP |

## User Input Blocking Windows

| Duration | Area | Waiting for |
| --- | --- | --- |
| 10m | User question TTL | User answer to an agent question |
| 30s | User question cleanup tick | Cleanup pass over stale user questions |
| 3m | Tool approval TTL | User approval/rejection of a tool call |
| 30s | Tool approval cleanup tick | Cleanup pass over stale tool approvals |
| 30m | File change TTL | User approval/rejection of a file change proposal |

## Agent Lifecycle And Coordination Waits

| Duration | Area | Waiting for |
| --- | --- | --- |
| 1s | Channel discovery tick | Agent discovering newly joined channels |
| 250ms | Unresponded history replay delay | Second pass after channel join for missed messages |
| 20m | Unresponded history max age | Oldest history message eligible for replay-on-join |
| 3s | Collaboration task min reply interval | Minimum spacing between collab task replies from the same agent |
| 20s | Unanswered nudge delay | Assistant safety-net nudge when no agent replied in a public channel |
| 5m | Unanswered track max age | Oldest public user message eligible for unanswered tracking |
| 5s | Unanswered tracker tick | Background scan for unanswered messages |
| 5s | Reminder monitor tick | Background scan for due reminders |
| 30m | File proposal expiry | File proposal lifetime before expiration |

## Agent Process And Connectivity Waits

| Duration | Area | Waiting for |
| --- | --- | --- |
| 10s | Standalone agent HTTP client timeout | Hub API calls in polling mode |
| 1s | Polling-mode fetch tick | New hub messages when using HTTP poll mode |
| 500ms to 15s | WebSocket reconnect backoff | Agent WebSocket reconnection to the hub |
| 90s | WebSocket read deadline | Next message or ping activity on hub WebSocket |
| 30s | WebSocket ping ticker | Keepalive ping loop |
| 5s | WebSocket ping write deadline | Sending ping control frame |

## Desktop UI Waits Relevant To Agents

| Duration | Area | Waiting for |
| --- | --- | --- |
| 50s | Loading screen hub wait | Hub to become reachable during startup |
| 3s default | Frontend WebSocket reconnect interval | Desktop chat reconnect to hub |
| 300ms | Agent list debounce | Refreshing agent list after bursts of events |
| 200ms | Explorer refresh debounce | Refreshing file explorer after file change proposals |
| 400ms | Repo workspace ensure delay | Ensuring repo-agent workspace after load |
| 5s | Channel list poll | Channel list refresh |
| 10s | Collaboration detail poll | Active collaboration refresh |
| 30s | Assistant state poll | Assistant reminders/tasks refresh |
| 900ms | CLI login terminal boot delay | Delay before auto-writing login command into PTY |

## Other Agent-Adjacent Waits

| Duration | Area | Waiting for |
| --- | --- | --- |
| 90s default | Session summary timeout | Async channel session summary LLM call |
| Configurable via `NJ_SESSION_SUMMARY_TIMEOUT` | Session summary config | Override for summary timeout |
| 10m | Hidden repo agent index build timeout | Background code indexing through backend |
| 500ms | Repo watcher debounce | Waiting for file changes to settle before follow-up work |
| 5m | Learning suggestion cooldown | Minimum gap between suggestion proposals |
| 15ms per chunk, capped at 1.5s total | Mock streaming delay | Simulated token streaming pace for CLI agent output |

## Non-Time-Based Wait Gates

These are important agent waits, but they are not simple duration-based timeouts:

- `PlanningSpeakerCooldownBlocked`: prevents one planning participant from speaking again until others have spoken in the round.
- Collaboration dispatch gating: tasks may wait on upstream completion, workspace acknowledgment, or approval state rather than a timer.
- Regular non-collaboration chat replies: there is no extra generation deadline beyond provider-specific timeouts.

## Main Configuration Knobs

These settings are the main places where timeout behavior can be changed:

- `ai.providers[].timeout_seconds`
- `collaboration.execution_timeout_seconds`
- `performance.agent_timeout_minutes`
- `NJ_SESSION_SUMMARY_TIMEOUT`

## Quick Summary

The most important practical numbers are:

- normal CLI-backed provider call: 300s
- collab chat task: 180s
- collab file task: 300s
- collab recap: 240s
- collab discussion: 480s
- implementation session v1: 480s to 600s
- agent runtime v2: 60m default
- user question: 10m
- tool approval: 3m
- file change approval: 30m
