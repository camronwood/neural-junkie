# Implementation Session

IDE **Agent mode** runs bounded multi-step **implementation sessions** for scoped coding tasks (read → edit → verify → optional repair).

## Trigger

All of the following:

- IDE layout send with `editor_mode=agent`
- `ide_route_agent_type` set (implicit IDE routing) or `implementation_session: true`
- Implementation intent (`please implement`, theme/feature verbs, or continuation after prior ask)
- Specialist agent that can ship `[FILE_CHANGE]` proposals

Desktop sets `implementation_session: true` automatically when Agent mode + code task signals:

- **IDE layout** — [`buildIdeDispatchPayload`](../desktop/src/utils/ideComposer.ts)
- **Team / general channels** — same metadata via `buildImplementationSessionMetadata` when the dev pack is enabled, workspace is shared, and the composer is in Agent mode

## Loop

1. **Stack manifest** — auto-detect React/Vue/Vite/Tauri/Tailwind, entry point, extension census; inject into round-0 prompt
2. **Discover** — seed files (`package.json`, `tailwind.config.js`, `src/App.tsx`, …) plus MCP tools (`read_file`, `grep`, `glob_file_search`, `semantic_search`) up to 20 tool iterations
3. **Grounding gate** — proposals blocked until seeds loaded (≥2), a discover tool ran, or stack manifest has an entry point
4. **Edit** — `propose_file_edit` tool or `[FILE_CHANGE]` blocks (up to 3 rounds); **preflight** rejects wrong-stack paths (e.g. `.vue` in a React repo)
5. **Apply** — file-change pipeline; auto-apply when `editor_agent_trust=auto_apply_edits`
6. **Verify** — only when trust is `auto_apply_edits`: `npm run build` (or `npx tsc --noEmit`) then `npm test --if-present` for Node; `go test ./...` / `cargo test` for other stacks
7. **Repair** — one extra round if preflight or verify fails
8. **Multi-file continue** — when stack manifest targets remain (e.g. `tailwind.config.js` then `src/App.tsx`), the session loops in the **same user turn** up to 5 files — no manual "go ahead" between files

Interactive trust skips verify (proposals await manual approval).

## Session summary headers

| Outcome | Message prefix |
|---------|----------------|
| Proposals only (interactive) | `proposals submitted for approval` |
| Auto-applied + verify passed | `applied and verified` |
| Auto-applied + verify failed | `applied but verification failed` |
| No file changes | `finished without file changes` |

## Provider routing (local-first)

Settings → **AI Providers → Implementation sessions**

- **Local Ollama first** (`implementation.routing_enabled`) — default tool-loop model: `qwen2.5-coder:7b` (`ollama pull qwen2.5-coder:7b`)
- **`implementation.fallback_provider_ids`** — used only when the configured **local Ollama provider is missing or unavailable**, not when a local model returns weak output. There is no automatic cloud escalation on implementation failure.
- **Cloud-grade work** — use an explicit CLI agent (e.g. `@Cursor` in chat) when you choose; see [CLI_AGENTS.md](CLI_AGENTS.md).

Restart the hub after changing implementation routing or agent code (`make server-regression`, not `make start-all` for scenario sweeps).

## Scenarios

```bash
make implement-scenarios-list
make implement-scenario SCENARIO=go-handler
make implement-scenario SCENARIO=react-theme-toggle
make implement-scenario SCENARIO=react-theme-multi-file
make implement-scenarios
make test-parity-stable
```

Requires live hub + configured agents (see `scenarios/implement/*.json`).

## Related

- [IDE_V3.md](IDE_V3.md) — IDE layout and Agent/Ask composer
- [MCP_INTEGRATION.md](MCP_INTEGRATION.md) — workspace MCP tools
