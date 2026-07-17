# Knowledge Graph

Neural Junkie builds a **local knowledge graph** over each workspace: packages, files, symbols, and import relationships. Edges are tagged `EXTRACTED` or `INFERRED`. Agents can expand neighborhoods and cite paths; the desktop ships an interactive React Flow workbench.

## Open the workbench

- Toolbar: **◈** (Knowledge Graph)
- Files panel menu → **Open knowledge graph**
- Command palette: `/nj-open-knowledge-graph`

## APIs

| Endpoint | Purpose |
|----------|---------|
| `GET /api/repo/graph?repo_path=` | Communities, god nodes, dense UI subgraph |
| `GET /api/repo/graph/subgraph?repo_path=&q=` | Scoped neighborhood |
| `GET /api/repo/graph/path?repo_path=&from=&to=` | Shortest path |
| `GET /api/repo/graph/explain?repo_path=&node=` | Node inspector payload |
| `GET /api/repo/graph/status?repo_path=` | Build status (`rebuild=1` to force) |

## Storage

`~/.neural-junkie/code-graph/<repo-hash>/graph.sqlite` plus `GRAPH_REPORT.md`.

## Agent retrieval

Questions like “how does X relate to Y” or “path between A and B” route to `code_graph` and inject neighborhood/path attachments alongside codebase search.

## Languages (v1)

Go, TypeScript/JavaScript, Python, Rust — symbols + import edges. Semantic chunking uses the same symbol boundaries as the graph pass.

See also: [features/knowledge-graph.html](features/knowledge-graph.html), [`internal/codeindex/graph`](../internal/codeindex/graph).
