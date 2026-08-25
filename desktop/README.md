# Neural Junkie — Desktop App

Tauri + React desktop client for Neural Junkie (v1.2.x). Talks to the **sidecar Go hub** on `localhost:18765` (bundled in macOS/Windows installers; started automatically in release builds).

## Features

- Real-time chat with AI agents via WebSocket
- Command palette (50+ slash commands) and @mention autocomplete
- File explorer, code editor (Monaco), terminal, pending file changes
- **Domain packs** modal (toolbar or **Settings → Domain packs**) — install IDE, software-dev, life sciences, and 9 other official packs
- Collaboration panel, runbooks, task management
- Settings: **Essentials** (appearance, connection, providers, collab routing) + collapsible **Advanced**
- Mermaid diagrams, threads, session persistence, auto-update (macOS/Windows)

Heavy panels (editor, packs, collab, workbenches) load on demand via code-splitting.

## Prerequisites

- **Node.js** 18+
- **Rust** (Tauri builds): `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`
- **Go 1.25+** (only when running the hub separately during dev)

## Development

From the repo root (starts hub + desktop):

```bash
make gui
```

Or hub separately:

```bash
make server   # terminal 1
cd desktop && npm run tauri:dev   # terminal 2
```

## Production build

```bash
make gui-build
# → src-tauri/target/release/bundle/
```

## Tech stack

Tauri 2 · React 18 · TypeScript · Vite · Zustand · Tailwind · WebSocket

## Configuration

- Hub URL: **Settings → Connection** (default `http://127.0.0.1:18765`)
- Window size: `src-tauri/tauri.conf.json`

## Troubleshooting

**WebSocket fails:** ensure the hub is running (`make server` in dev, or restart the desktop app in release).

**Port 1420 in use:** change `vite.config.ts` or stop the other Vite process.
