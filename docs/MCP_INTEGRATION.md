# MCP Integration Guide

## Overview

The Neural Junkie now supports Model Context Protocol (MCP) servers for specialized agents, providing them with executable tools for data-driven analysis and diagnostics.

## Architecture

### MCP Server Structure

```
neural-junkie/
├── internal/
│   ├── mcp/                          # MCP server implementations
│   │   ├── server.go                 # Shared MCP utilities
│   │   ├── backend/                  # Backend Agent MCP tools
│   │   │   └── backend_mcp.go        # Go analysis tools
│   │   ├── devops/                   # DevOps Agent MCP tools
│   │   │   └── devops_mcp.go         # Infrastructure tools
│   │   ├── database/                 # Database Agent MCP tools
│   │   │   └── database_mcp.go       # Query analysis tools
│   │   ├── frontend/                 # Frontend Agent MCP tools
│   │   ├── security/                 # Security Agent MCP tools
│   │   ├── codereview/               # CodeReviewer diagnostic tools
│   │   ├── architecture/             # SoftwareArchitect diagnostic tools
│   │   ├── rust/                     # RustExpert MCP tools
│   │   ├── repomcp/                  # Repo agent in-process tools
│   │   ├── confluencemcp/            # Confluence agent in-process tools
│   │   └── biology/                  # BiologyExpert MCP tools
│   │       └── biology_mcp.go        # Sequence + structure tools
```

### Port Allocation

- Hub Server: `:18765` (existing)
- Backend MCP: `:8081`
- DevOps MCP: `:8082`
- Database MCP: `:8083`
- Frontend MCP: `:8084`
- Security MCP: `:8085`
- Resource exports: `:8086` (opt-in, `ENABLE_MCP_RESOURCES=true`)
- Biology MCP: `:8087` (Life sciences pack)
- Rust MCP: `:8088`
- Code Review MCP: `:8089`
- Architecture MCP: `:8090`
- Repo / Confluence agents: in-process only (no HTTP port)

## Configuration

MCP tool servers are controlled from **Settings → AI & providers** and persisted in `~/.neural-junkie/config.json`:

- `mcp.enabled` — master switch (default on)
- `mcp.agents` — per-type overrides (`backend`, `devops`, `database`, `biology`, …). When omitted, enablement follows enabled pack `mcp_agents` in each pack's `pack.yaml`.
- `mcp.ports` — optional port overrides
- `mcp.biology` — ESMFold model, sequence limits, artifacts directory
- `hf.token` — Hugging Face hub token (folding + downloads)

**Life sciences pack:** enabling the pack starts BiologyExpert and its MCP tools automatically — no `ENABLE_BIOLOGY_MCP` env var. See [BIOLOGY_PACK.md](BIOLOGY_PACK.md).

**Software development pack:** enabling the pack starts MCP servers for backend, frontend, platform, database, security, code review, and architecture specialists when those agents are enabled. See [SOFTWARE_DEVELOPMENT_PACK.md](SOFTWARE_DEVELOPMENT_PACK.md).

**External MCP clients:** see [MCP_EXTERNAL_CLIENTS.md](MCP_EXTERNAL_CLIENTS.md) to connect Claude Desktop or other hosts to `:8081/mcp`, etc.

## Discovering tools

| Method | What it shows |
|--------|----------------|
| **Agent ℹ️** (My Agents or agent list) | Chat model, tool-loop model (e.g. qwen fallback for OpenBio), each MCP tool with parameters and example prompts |
| **Tool count badge** | Sidebar agent shortcuts and My Agents list show `N tools` when the specialist is running |
| **`/tools-list`** | MCP tools for every agent **in the current channel** only |
| **HTTP API** | `GET /api/agent-tools?agent_id=…`, `GET /api/channel-tools?channel=…`, `GET /api/agents?include_tool_counts=true` |

CLI agents (Cursor, Gemini, Claude CLI) use external toolsets; the UI notes that instead of listing hub MCP tools.

## Available Tools

### BiologyExpert (Life sciences)

- `summarize_scan_summary(path)` — QC summary for Phoenix-style scan summary folders (`imageMetadata.json` + well TIFFs)
- `summarize_scan_analysis(path)` — QC summary for Phoenix-style analysis exports (`reports/results.json`, `reports/{analyte}_summary_report.csv`, and `process_report.txt`)
- `run_12plex_qc(path)` — Human Inflammatory 12-Plex SOP QC (LLOQ, ULOQ, CV, column/row, spike recovery)
- `summarize_panel_qc(path)` — alias for `run_12plex_qc`
- `summarize_comparator_output(path)` — summary of Plate Comparator Analysis output folder
- `run_secondary_analysis(workflow, config_json)` — inline `12plex_qc` / `summarize_comparator`; other workflows use Secondary Analysis panel

**Tools:**
- `analyze_sequence(sequence)` — DNA/RNA/protein validation and summary
- `fold_protein(sequence)` — ESMFold via HF Inference (`router.huggingface.co/hf-inference`; hub token in Settings)

See [BIOLOGY_PACK.md](BIOLOGY_PACK.md).

### Backend Agent (Go Expert)

**Tools:**
- `analyze_go_code(file_path)` - Run static analysis using go vet, staticcheck, golangci-lint
- `run_go_tests(package_path)` - Execute Go tests and return results
- `profile_performance(binary_path, endpoint)` - Profile Go application performance using pprof
- `check_dependencies(module_path)` - Check Go module dependencies for vulnerabilities
- `detect_race_conditions(package_path)` - Run Go race detector on tests

**Example Usage:**
```
User: "Why is my Go API slow?"
Backend Agent: *Uses profile_performance tool*
              "I profiled your API. The issue is in getUserPosts() 
               which takes 450ms. It's doing 50 sequential DB queries.
               Here's the exact line: handlers/user.go:145"
```

### DevOps Agent

**Tools:**
- `kubectl_query(resource, namespace)` - Query Kubernetes cluster using kubectl
- `check_docker_image(image_name)` - Analyze Docker image for size, layers, and vulnerabilities
- `validate_yaml(yaml_file)` - Validate Kubernetes or Helm YAML files
- `check_pod_logs(pod_name, namespace)` - Fetch and analyze logs from Kubernetes pods
- `query_prometheus(query)` - Query Prometheus metrics for monitoring data

**Example Usage:**
```
User: "What's wrong with my deployment?"
DevOps Agent: *Uses kubectl_query and check_pod_logs tools*
              "I checked your pods. The issue is in the 'api-server' pod:
               - Status: CrashLoopBackOff
               - Last error: 'database connection timeout'
               - Recommendation: Check your database connectivity"
```

### Database Agent (SQL Master)

**Tools:**
- `explain_query(sql_query)` - Run EXPLAIN ANALYZE on SQL queries to analyze performance
- `check_indexes(table_name)` - Analyze table indexes for optimization opportunities
- `validate_schema(schema_name)` - Check database schema for consistency and best practices
- `suggest_optimizations(table_name)` - Analyze query patterns and suggest database optimizations
- `check_table_stats(table_name)` - Get table statistics including size, row count, and storage info
- `generate_migration(description, changes)` - Generate database migration scripts based on schema changes

**Example Usage:**
```
User: "My query is slow, how can I optimize it?"
Database Agent: *Uses explain_query tool*
                "I analyzed your query with EXPLAIN ANALYZE:
                 - Sequential scan on 1M rows (cost: 25000)
                 - Missing index on 'user_id' column
                 - Recommendation: CREATE INDEX idx_users_user_id ON users(user_id)"
```

## How It Works

### Tool Execution Flow (in-app)

```
User: "Run go vet on ./cmd/server"
  ↓
Backend Agent receives message
  ↓
buildPrompt lists tools from mcpServer.ListTools()
  ↓
generateWithAgentTools → provider tool loop (Claude tool_use, Ollama/OpenAI-compat native tools, or ReAct text fallback)
  ↓
executeMCPTool calls handler in-process on the agent's MCPServer
  ↓
tool result returned to the model
  ↓
Final answer sent to the user
```

**Supported providers for hub MCP tool loops:** Claude (native `tool_use`), Ollama (native tools + ReAct + model swap), LM Studio and generic OpenAI-compatible endpoints (native OpenAI `tool_calls` + ReAct when native unsupported), and HuggingFace Router (via OpenAI-compat layer).

External MCP clients (Claude Desktop, Cursor) can also connect to the agent HTTP servers — see [MCP_EXTERNAL_CLIENTS.md](MCP_EXTERNAL_CLIENTS.md).

**Claude requirement:** Direct Anthropic API (`USE_AI_HUB=false`, `ANTHROPIC_API_KEY` set) for Claude agents. MCP tool loops through AI Hub are not supported for Claude today.

### Agent Integration

Each specialized agent:
1. Creates its MCP server on startup when `ENABLE_MCP=true` and per-agent flags are set
2. Registers domain-specific tools
3. Starts HTTP server on `/mcp` for external clients
4. Lists tools dynamically in the system prompt
5. Executes tools in-process when Claude requests them during chat

### Enhanced Prompts

Agents with MCP tools have enhanced prompts:

```
You are Go Expert, a backend agent.
Your expertise: Go, Node.js, REST APIs, Microservices, ...

AVAILABLE TOOLS:
- analyze_go_code(file_path): Run static analysis on Go code
- run_go_tests(package_path): Execute Go tests and return results
- profile_performance(binary_path, endpoint): Profile API endpoint performance
- check_dependencies(module_path): Analyze go.mod for vulnerabilities

Use these tools to provide data-driven answers. When diagnosing issues,
USE THE TOOLS to get actual data rather than guessing.
```

## Security & Safety

### Tool Security
- **Read-only by default** - No destructive operations
- **Sandboxed** where possible
- **Rate-limited** to prevent abuse
- **Logged** for audit trails

### Graceful Degradation
- If MCP server fails, agent still works (just no tools)
- Environment flag to disable MCP entirely
- Per-agent MCP enable/disable
- Tools can fail without breaking agent responses

## Development

### Adding New Tools

1. **Create tool handler** in the appropriate MCP server
2. **Register tool** in the `registerTools()` method
3. **Update agent prompts** to include new tool description
4. **Test tool** with real data

Example:
```go
// In backend_mcp.go
func (b *BackendMCP) registerTools() {
    b.mcpServer.AddTool(mcp.Tool{
        Name:        "new_tool",
        Description: "Description of what the tool does",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "param": map[string]any{
                    "type":        "string",
                    "description": "Parameter description",
                },
            },
            Required: []string{"param"},
        },
    }, b.handleNewTool)
}
```

### Testing MCP Tools

```bash
# Test Backend Agent with MCP
make agent-backend

# Test DevOps Agent with MCP  
make agent-devops

# Test Database Agent with MCP
make agent-database
```

### Debugging

Check MCP server logs:
```bash
# Backend MCP server logs
tail -f /tmp/backend-mcp.log

# DevOps MCP server logs
tail -f /tmp/devops-mcp.log
```

## Troubleshooting

### Common Issues

**MCP server not starting:**
- Check if port is available: `lsof -i :8081`
- Verify environment variables: `echo $ENABLE_BACKEND_MCP`
- Check logs for startup errors

**Tools not working:**
- Verify required tools are installed (go, kubectl, etc.)
- Check file permissions for tool execution
- Review MCP server logs for errors

**Agent not using tools:**
- Verify MCP server is running
- Check agent logs for MCP connection errors
- Ensure tools are properly registered

### Performance

- **Tool execution timeout**: 30 seconds
- **Tool result caching**: 5 minutes
- **Concurrent tool limits**: Per-agent rate limiting
- **Memory usage**: ~10MB per MCP server

## Future Enhancements

- **Custom MCP tools** per project/repository
- **User-defined tool configurations**
- **Tool composition** (chain tools together)
- **AI-powered tool selection** optimization
- **Tool result caching** and indexing
- **Multi-agent tool collaboration**

## Examples

### Backend Development Workflow

```
User: "I'm getting a panic in my Go service"
Backend Agent: *Uses analyze_go_code tool*
              "I found the issue in handlers/user.go:45:
               - Panic: 'nil pointer dereference'
               - Line: user.Profile.Name
               - Fix: Add nil check before accessing Profile"
```

### DevOps Troubleshooting

```
User: "My pods are failing to start"
DevOps Agent: *Uses kubectl_query and check_pod_logs tools*
              "I checked your cluster:
               - Pod status: Pending
               - Issue: 'Insufficient CPU resources'
               - Recommendation: Increase CPU limits or scale cluster"
```

### Database Optimization

```
User: "My database queries are slow"
Database Agent: *Uses explain_query and check_indexes tools*
                "I analyzed your query performance:
                 - Missing index on 'created_at' column
                 - Sequential scan on 500K rows
                 - Recommendation: CREATE INDEX idx_orders_created_at ON orders(created_at)"
```

## Conclusion

MCP integration transforms specialized agents from conversational assistants into powerful diagnostic tools that can:

- **Analyze real code** instead of guessing
- **Execute actual tests** and return results
- **Profile live systems** for performance issues
- **Query databases** for optimization opportunities
- **Inspect infrastructure** for operational issues

This provides users with **data-driven insights** rather than generic advice, making the Neural Junkie a truly powerful development and operations tool.
