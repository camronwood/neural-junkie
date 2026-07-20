# Context compression (CCR)

Neural Junkie compresses large tool outputs and prompt sections **before** they reach the LLM, stores originals locally, and exposes **`nj_retrieve_context`** for on-demand expansion. Inspired by reversible compression patterns (Headroom CCR); implemented natively in Go.

## What gets compressed

| Source | Strategy | Recoverable |
|--------|----------|-------------|
| `grep`, `glob_file_search`, `list_dir`, `semantic_search`, repo search tools | Top-N lines/chunks + count | Yes (`nj_retrieve_context`) |
| `read_file`, `get_file_content` | Signatures + head/tail preview | Yes |
| Command/log tools | Error tail + exit context | Yes |
| Prompt sections (`SESSION SUMMARY`, `WORKSPACE CONTEXT`, etc.) | Generic CCR when over byte cap | Yes |

Compression runs at the agent MCP choke point ([`internal/agent/mcp_tools.go`](../internal/agent/mcp_tools.go)) and in [`applyContextBudget`](../internal/agent/context_budget.go).

## Configuration

In `~/.neural-junkie/config.json` under `performance`:

```json
{
  "performance": {
    "context_compress_enabled": true,
    "context_compress_max_tool_bytes": 12000,
    "context_cache_max_entries": 500,
    "context_cache_ttl_minutes": 60,
    "output_shaping_enabled": false
  }
}
```

- **`context_compress_enabled`** — default **true** when unset
- **`output_shaping_enabled`** — appends terse hints after read-only tool steps (default **false**)

Cache files: `~/.neural-junkie/context-cache/` (optional disk spill).

## Retrieve tool

Registered on workspace-enabled MCP servers as **`nj_retrieve_context`**:

- **`ref`** — exact `ctx-` + 12 hex characters from a compression marker in the current turn (never invent or copy documentation examples)
- **`query`** — optional substring filter on cached lines

Rate limit: **3 retrieves per tool-loop turn**.

## Observability

Message metadata (when compression occurred):

- `context_compress_bytes_in` / `context_compress_bytes_out`
- `context_compress_strategy`
- `context_compress_refs`

Desktop: **Settings → Models & performance → Compression badges on messages** (optional, off by default).

Debug (`NEURAL_JUNKIE_DEBUG=1`):

```bash
curl 'http://localhost:18765/api/debug/context-compress?tool=grep&text=...'
```

## Harness workflow

Generate a reviewable scenario stub from debug output:

```bash
python3 scripts/routing-failure-to-scenario.py --mode routing --q 'review JWT auth flow'
python3 scripts/routing-failure-to-scenario.py --mode compress --tool grep --text "$(python3 -c 'print(\"x\\n\"*500)')"
```

Edit the emitted JSON before adding to `scenarios/`.

## Related

- [CONTEXT_MODEL.md](CONTEXT_MODEL.md) — context stack and budget
- [CLI_AGENTS.md](CLI_AGENTS.md) — CLI agents receive hub-assembled prompts through the same budget path
