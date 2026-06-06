# IDE v3 (Cursor-like coding in main chat)

IDE v3 routes **software-development** work through the **main channel chat** when using the **IDE layout preset** — no separate editor agent panel. Requires the **Software development** pack (see [IDE_V2.md](IDE_V2.md) for symbols, LSP, inline completion).

## Layout modes

| Preset | Description |
|--------|-------------|
| **Team** | Chat-first (default for non-dev workflows) |
| **IDE** | Files + editor + main chat (project-first) |

Settings → **Layout** → preset, or toolbar **IDE** button.

## Main chat in IDE mode

When **IDE layout** and the dev pack are on, each send:

- **Implicit routing** — `ide_route_agent_type` from the open file (Go → backend, TS/CSS/HTML → frontend) if you did not @ someone. No auto-`@BackendEngineer` in the message body (use `@mention` when you want a specific agent).
- **Focus context** — active tab path, buffer, and editor selection (when scoped).
- **Ask / Agent** — composer toggle: Ask adds a read-only instruction prefix; Agent allows file-change proposals.
- **@codebase** — hybrid embedding + keyword search via `POST /api/repo/search/semantic` → `prompt_attachments` (background index build on first query; status at `GET /api/repo/index/status`).
- **Routing metadata** — `ide_route_agent_type`, `editor_mode`, `editor_agent_trust` on the message so only the matching specialist responds.

## Keyboard

IDE-focused shortcuts (full list: [KEYBOARD_SHORTCUTS.md](KEYBOARD_SHORTCUTS.md), **Settings → Keyboard**).

| Shortcut | Action |
|----------|--------|
| **⌘K** / **Ctrl+K** | Fast edit (single-turn; terminal clear when terminal focused) |
| **⌘L** / **Ctrl+L** | Focus main chat composer (IDE layout) |
| **⌘P** / **Ctrl+P** | Quick open |
| **⌘⇧O** / **Ctrl+Shift+O** | Go to symbol |
| **⌘⇧E** / **Ctrl+Shift+E** | Toggle file explorer |
| **⌘⇧F** / **Ctrl+Shift+F** | Open code editor panel |
| **⌘S** / **Ctrl+S** | Save file |
| **⌘⇧S** / **Ctrl+Shift+S** | Save all |
| **⌘W** / **Ctrl+W** | Close editor tab |
| **⌘Tab** / **Ctrl+Tab** | Next editor tab |
| **⌘⇧W** / **Ctrl+Shift+W** | Workspace switcher |
| **⌘J** / **Ctrl+J** | Terminal |

## Editor trust (Settings → Layout)

| Mode | Behavior |
|------|----------|
| **interactive** | Approve file changes in pending panel / review bar |
| **auto_apply_edits** | Auto-approve proposals when the hub returns change IDs |
| **yolo** | Reserved for tool approval parity with CLI agents |

## APIs (still used by fast edit / tooling)

- `POST /api/dev/agent-turn` — optional multi-turn path (legacy panel); main UX uses normal `sendMessage` + metadata.
- `POST /api/dev/complete` — inline completion (Ollama)
- `POST /api/repo/search/semantic` — @codebase chunks

## v4 (not in v3)

- Remote SSH / dev containers
- Full Monaco LSP client
