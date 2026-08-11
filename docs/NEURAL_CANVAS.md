# Neural Canvas

Neural Canvas is Neural Junkie's native surface for durable agent artifacts.
Artifacts are app-managed, revisioned documents that open beside chat without
writing generated files into a workspace.

## Core renderers

- `nj.document` — default collaborative page (block document)
- `nj.markdown` — legacy page; opened as a document (one markdown block) and
  rewritten to `nj.document` on the next update
- `nj.mermaid` and `nj.code`
- `nj.table`, `nj.chart`, and `nj.timeline` — standalone artifacts
- `nj.image` and `nj.graph`
- `nj.map` — interactive OSM maps with markers and walking/driving route polylines
  (Maps pack; media type `application/vnd.neural-junkie.map+json`)
- Host-owned adapters for knowledge graph, runbook, CAD, structure, music,
  Model Arena, scan, and comparator workbenches

Renderers consume validated declarative data. Neural Canvas never evaluates
artifact JavaScript or pack-provided React components.

## Document schema (`nj.document`)

Media type: `application/vnd.neural-junkie.document+json`. API version `1`.

```json
{
  "schema_version": 1,
  "title": "Trip plan",
  "blocks": [
    { "type": "heading", "level": 1, "text": "Trip plan" },
    { "type": "list", "items": ["Tokyo", "Kyoto"] },
    {
      "type": "table",
      "columns": [{ "key": "name", "label": "Name" }],
      "rows": [{ "name": "Ada" }]
    }
  ]
}
```

v1 block types: `heading`, `markdown`, `list`, `table`, `callout`, `mermaid`,
`image`, `columns`. Discriminator is `type`, never `kind`. Unknown types render
as a compact fallback.

Agents may still emit Markdown. The host compiler lifts headings, lists, GFM
pipe tables, mermaid fences, and images into blocks. Existing `nj.markdown`
string payloads unwrap on read and are not rewritten until the next update.

## Storage and privacy

Artifacts are stored under `~/.neural-junkie/artifacts/`. Each artifact records
its renderer, media type, revision, provenance, and optional workspace,
project-set, channel, thread, and collaboration associations. Content remains
local unless a user exports it or another configured integration transmits it.

Back up this directory to preserve Canvas history. Deleting it removes local
artifacts but does not modify source workspaces.

## Workspace export

Export creates a normal Neural Junkie file-change proposal. Review and approve
that proposal before the artifact is written into a local, SSH, or devcontainer
workspace. This keeps app-state creation fast while retaining workspace
mutation controls.

## Pack integration

Packs declare `artifact-renderer` capability definitions that map media types or
file globs to trusted `nj.*` renderer IDs. See
[PACK_CAPABILITY_DEFS.md](PACK_CAPABILITY_DEFS.md). Unknown renderer IDs and
incompatible schema versions fall back to Markdown, text, or download.

## Recovery

If an artifact cannot render:

1. Open its provenance and revision details.
2. Select an earlier revision or use the fallback view.
3. Export the JSON payload for inspection.
4. Restore the backed-up artifact directory if the manifest is damaged.

The API treats malformed manifests as isolated corruption; one damaged artifact
does not prevent other artifacts from loading.
