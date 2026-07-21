# Neural Canvas

Neural Canvas is Neural Junkie's native surface for durable agent artifacts.
Artifacts are app-managed, revisioned documents that open beside chat without
writing generated files into a workspace.

## Core renderers

- `nj.markdown`, `nj.mermaid`, and `nj.code`
- `nj.table`, `nj.chart`, and `nj.timeline`
- `nj.image` and `nj.graph`
- Host-owned adapters for knowledge graph, runbook, CAD, structure, music,
  Model Arena, scan, and comparator workbenches

Renderers consume validated declarative data. Neural Canvas never evaluates
artifact JavaScript or pack-provided React components.

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
