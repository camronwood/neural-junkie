# IDE pack

Neural Junkie **core** always includes workspace file explorer, Monaco editor, remote attach, and terminal. The optional **IDE** domain pack unlocks editor depth on top of that shell.

Install from **Settings → Domain packs → Pack store**, or sideload `ide-<version>.zip` from [neural-junkie-pack-ide](https://github.com/camronwood/neural-junkie-pack-ide).

Independent of the [Software development pack](SOFTWARE_DEVELOPMENT_PACK.md) — install IDE alone for Git/LSP/composer, or both for full engineering workflow.

## What you get

| Capability | What it enables |
|------------|-----------------|
| `layout_profile: ide` | IDE layout owner (project-first preset) |
| `ide-v2` | Git SCM, Problems, quick open, symbols, fast edit, IDE shortcuts, @codebase context |
| `ide-v3-composer` | Ask/Agent composer routing metadata in main chat |
| `ide-v4` | Full Monaco LSP client + remote LSP relay |
| `git-rest` | Hub git, LSP, dev complete, fast-edit, file/symbol search routes |
| `inline-completion` | Ghost-text FIM in Monaco (Settings toggle) |

## Core vs IDE pack

| Feature | Layer |
|---------|-------|
| Workspace tree, open/save, built-in viewers | Core NJ |
| Remote SSH / devcontainer attach UI | Core NJ |
| Terminal, pending file changes | Core NJ |
| IDE layout toggle, Git panel, LSP, inline completion | **IDE pack** |

See [IDE_V2.md](IDE_V2.md), [IDE_V3.md](IDE_V3.md), [IDE_V4.md](IDE_V4.md).

## Enable the pack

**Settings → Domain packs** — toggle **IDE** on or off.

When enabled:

- Hub exposes git/LSP/dev routes (requires `git-rest` capability).
- Desktop shows IDE layout toggle, Git/Problems/Quick open modals, and IDE shortcuts.
- Inline completion available when enabled in **Settings → Layout**.

When disabled, file explorer and Monaco editor remain available; IDE-depth features are hidden.

## Upgrade from software-development–bundled IDE

Existing installs with **Software development** enabled are migrated automatically: the hub installs and enables the **IDE** pack and moves `layout_owner` from `software-development` to `ide`.

## Related packs

- **[Software development](SOFTWARE_DEVELOPMENT_PACK.md)** — engineering specialists + MCP tools (independent; developer wizard enables both).
- Domain workbench packs (CAD, life sciences, web browser) use the core editor shell; their viewers are separate capabilities.
