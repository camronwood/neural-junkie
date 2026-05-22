# IDE v2 (Software development pack)

IDE v2 extends the dev pack with navigation/SCM depth (**v2a**) and editor-integrated agent workflow (**v2b**). All features require the **Software development** pack.

## v2a — Navigation and SCM

### v2a.1 Git SCM v2

- Hub: `POST /api/git-add`, `POST /api/git-reset`, `GET /api/git-diff?staged=true`, `GET /api/git-file-sides`
- Desktop: Git modal — stage/unstage per file, stage all / unstage all, Monaco diff viewer

### v2a.2 Go to symbol

- Hub: `GET /api/workspaces/symbols/search`
- Desktop: **⌘⇧O** symbol modal, jump to file + line

### v2a.3 Diagnostics and Problems

- Monaco TypeScript/JavaScript language services for `.ts`, `.tsx`, `.js`, `.jsx`
- Problems panel (toolbar **!** button)
- Click a problem → open file at line

### v2a.3b Go diagnostics (optional)

- Hub: `GET /api/lsp/go/diagnostics` — runs `gopls check` when `gopls` is on PATH
- Merged into Problems for open `.go` tabs

## v2b — Agent in the editor

### v2b.1 Inline hunks

- Parses unified diff from pending file-change preview
- Green/red line decorations on the open file; glyph margin click applies a hunk to the buffer
- Full approve still uses the existing file-change approval flow

### v2b.2 Fast edit

- Hub: `POST /api/dev/fast-edit` — single specialist turn, proposes `[FILE_CHANGE]` when needed
- Desktop: **⌘K** when the code editor is open

## Deferred (v2c)

- Project-first default layout
- Tab completion / ghost text
- Full multi-language LSP (Rust, Python)
- Remote SSH / dev containers
- tree-sitter symbol index

See [SOFTWARE_DEVELOPMENT_PACK.md](./SOFTWARE_DEVELOPMENT_PACK.md) for pack enablement.
