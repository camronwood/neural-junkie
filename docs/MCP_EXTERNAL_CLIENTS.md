# Connecting external MCP clients

Neural Junkie hub specialists expose **HTTP MCP endpoints** that external clients (Claude Desktop, Cursor, other MCP hosts) can register alongside built-in tools.

## Endpoints

When the hub is running and the specialist is enabled, MCP servers listen on:

| Specialist | URL |
|------------|-----|
| BackendEngineer | `http://localhost:8081/mcp` |
| PlatformEngineer | `http://localhost:8082/mcp` |
| DatabaseSpecialist | `http://localhost:8083/mcp` |
| FrontendEngineer | `http://localhost:8084/mcp` |
| SecurityReviewer | `http://localhost:8085/mcp` |
| BiologyExpert | `http://localhost:8087/mcp` |
| RustExpert | `http://localhost:8088/mcp` |
| CodeReviewer | `http://localhost:8089/mcp` |
| SoftwareArchitect | `http://localhost:8090/mcp` |

Export/resource server (opt-in): `http://localhost:8086/mcp` when `ENABLE_MCP_RESOURCES=true`.

**Repo** and **Confluence** agents use **in-process tools only** (no HTTP port) — they are not available to external MCP clients.

Override ports in `~/.neural-junkie/config.json` under `mcp.ports`.

## Claude Desktop example

Add to Claude Desktop MCP config (path varies by OS):

```json
{
  "mcpServers": {
    "neural-junkie-backend": {
      "url": "http://localhost:8081/mcp"
    },
    "neural-junkie-devops": {
      "url": "http://localhost:8082/mcp"
    }
  }
}
```

Restart Claude Desktop after editing.

## Cursor / other hosts

Use the same streamable HTTP `/mcp` URL format your host supports. Ensure:

1. Neural Junkie hub is running (`make start-all` or your usual workflow).
2. The **Software development** or **Life sciences** pack is enabled for the specialist.
3. **MCP tool servers** are on in Settings → Domain packs → MCP specialist tools.

## Troubleshooting

- **Connection refused** — specialist not running or MCP disabled for that agent type.
- **Empty tool list** — check Agent ℹ️ in Neural Junkie desktop for live tool registration.
- **Port conflict** — set custom `mcp.ports` in config.json.

See [MCP_INTEGRATION.md](MCP_INTEGRATION.md) for architecture and tool catalogs.
