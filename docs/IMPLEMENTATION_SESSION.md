# Implementation Session

IDE **Agent mode** runs bounded multi-step **implementation sessions** for scoped coding tasks (read → edit → verify → optional repair).

## Trigger

All of the following:

- IDE layout send with `editor_mode=agent`
- `ide_route_agent_type` set (implicit IDE routing) or `implementation_session: true`
- Implementation intent (`please implement`, theme/feature verbs, or continuation after prior ask)
- Specialist agent that can ship `[FILE_CHANGE]` proposals

Desktop sets `implementation_session: true` automatically when Agent mode + code task signals (`buildIdeDispatchPayload`).

## Loop

1. **Discover** — workspace MCP tools (`read_file`, `grep`, `glob_file_search`, `semantic_search`) up to 20 tool iterations
2. **Edit** — `propose_file_edit` tool or `[FILE_CHANGE]` blocks (up to 3 rounds)
3. **Apply** — file-change pipeline; auto-apply when `editor_agent_trust=auto_apply_edits`
4. **Verify** — auto `go test ./...` or `npm test --if-present` when manifest present
5. **Repair** — one extra round if verify fails

## Provider routing

Settings → **AI Providers → Implementation sessions**

- Local Ollama first (`implementation.routing_enabled`)
- Tool-loop model default: `qwen2.5-coder:7b`
- Fallback provider IDs in hub config (`implementation.fallback_provider_ids`)

## Scenarios

```bash
make implement-scenarios-list
make implement-scenario SCENARIO=go-handler
make implement-scenarios
```

Requires live hub + configured agents (see `scenarios/implement/*.json`).

## Related

- [IDE_V3.md](IDE_V3.md) — IDE layout and Agent/Ask composer
- [MCP_INTEGRATION.md](MCP_INTEGRATION.md) — workspace MCP tools
