# IDE v2 (Software development pack)

IDE v2 extends the dev pack with navigation/SCM depth (**v2a**), editor-integrated agent workflow (**v2b**), and editor depth (**v2c**). All features require the **Software development** pack.

See [IDE_V3.md](IDE_V3.md) for Cursor-like coding in the main chat (IDE layout).

## v2a — Navigation and SCM

### v2a.1 Git SCM v2

- Hub: `POST /api/git-add`, `POST /api/git-reset`, `GET /api/git-diff?staged=true`, `GET /api/git-file-sides`
- Desktop: Git modal — stage/unstage per file, stage all / unstage all, Monaco diff viewer

### v2a.2 Go to symbol

- Hub: `GET /api/workspaces/symbols/search` (`q`, `kind`, `limit`)
- Cached symbol index under `~/.neural-junkie/symbol-index/`
- Desktop: **⌘⇧O** symbol modal, jump to file + line

### v2a.3 Diagnostics and Problems

- Monaco TypeScript/JavaScript language services for `.ts`, `.tsx`, `.js`, `.jsx`
- Problems panel (toolbar **!** button)
- Click a problem → open file at line

### v2a.3b Go / Rust / Python diagnostics (optional)

- `GET /api/lsp/go/diagnostics` — `gopls check` when on PATH
- `GET /api/lsp/rust/diagnostics` — `cargo check --message-format=json` when on PATH
- `GET /api/lsp/python/diagnostics` — `pyright --outputjson` when on PATH

## v2b — Agent in the editor

### v2b.1 Inline hunks

- Parses unified diff from pending file-change preview
- Green/red line decorations on the open file; glyph margin click applies a hunk to the buffer
- Full approve still uses the existing file-change approval flow

### v2b.2 Fast edit

- Hub: `POST /api/dev/fast-edit` — single specialist turn, proposes `[FILE_CHANGE]` when needed
- Desktop: **⌘K** when the code editor is open

## v2c — Editor depth (shipped)

### v2c.1 Project-first layout

- Settings / toolbar: **IDE** vs **Team** preset
- IDE: embedded file tree + editor + optional Agent dock; **Chat** toggles team channels

### v2c.2 Symbol index

- Go, TS/JS, Rust, Python via regex scan + disk cache (tree-sitter CLI optional later)

### v2c.3 Multi-language diagnostics

- Rust and Python hub routes (subprocess, same pattern as Go)

### v2c.4 Tab completion

- `POST /api/dev/complete` — Ollama FIM
- Settings → inline completion toggle
- Monaco inline completions provider

## Deferred (v4+)

- Remote SSH / dev containers
- Full Monaco LSP client
- tree-sitter CLI / embedded queries (optional upgrade to symbol index)

See [SOFTWARE_DEVELOPMENT_PACK.md](./SOFTWARE_DEVELOPMENT_PACK.md) for pack enablement.
