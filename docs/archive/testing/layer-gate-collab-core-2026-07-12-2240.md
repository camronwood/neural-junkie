# Layer gate — collab-core — 2026-07-12-2240 UTC

layer=collab-core
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-core` | FAIL | 4094s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-12-2240.log`

## Failures (tail)

### collab-scenarios-core (exit 2)

```text
resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase=None discussion.status=None msgs=None/None
=== FAIL: plan-dependency-prose-regression ===


⚠ hub crash detected [collab:plan-dependency-prose-regression] at http://127.0.0.1:18765
  documenting crash; up to 3 restart attempt(s)
  → recovery attempt 1/3 (make stop && make server-regression)
  ✓ hub healthy after restart attempt 1

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 6
  generation_error posts in channel: 6
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 178
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/4
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 101
  generation_error posts in channel: 101
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/3
=== FAIL: collab-minimal-completion-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 100
  generation_error posts in channel: 100
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/3
=== FAIL: collab-minimal-completion-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your messag
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 101
  generation_error posts in channel: 101
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/3
=== FAIL: document-findings-execution ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Ollama is not running on this machine (localhost:11434). Start it with `ollama serve`, then send your message 
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 100
  generation_error posts in channel: 100
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/3
=== FAIL: document-findings-execution ===

make[1]: *** [collab-scenarios-core] Error 1

RESULT collab-scenarios-core: FAIL (exit 2, 4094s)
```

