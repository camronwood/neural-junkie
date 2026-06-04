# CAD Pack (v1)

Neural Junkie includes a **CAD** domain pack with OpenSCAD parametric modeling, **CADExpert**, and a desktop CAD workbench.

## Enable the pack

**Settings → Domain packs → Pack store** — install and enable **CAD**.

When enabled:

- **CADExpert** is added to configured specialists.
- **CAD / Mechanical design** appears in New DM and `/create-expert cad`.
- Models `qwen2.5-coder:14b`, `qwen2.5-coder:7b`, `qwen2.5:7b`, and optional `nj-cad:27b` are merged into **models to ensure**.
- Open `.scad` files in the **CAD workbench** (Monaco editor, param sliders, Three.js STL preview, version history).

## Prerequisites

Install **OpenSCAD** on your machine: [https://openscad.org](https://openscad.org)

Test from **Settings → Domain packs → CAD tools → Test OpenSCAD**.

## Model stack (three layers)

| Layer | Technology |
|-------|------------|
| Authoring (LLM) | CADExpert chat model (default `qwen2.5-coder:14b`; optional `nj-cad:27b`) |
| Geometry | OpenSCAD CLI → STL |
| Viewport | Three.js in desktop workbench |

**Tool runner:** `qwen2.5:7b` when the chat model lacks native Ollama tool calling (same pattern as Life sciences).

Change chat/tool models in **Settings → Domain packs → CAD tools**.

## MCP tools (CADExpert)

| Tool | Purpose |
|------|---------|
| `write_openscad` | Save/update `.scad` |
| `render_openscad` | OpenSCAD → STL |
| `list_openscad_params` | Parse Customizer variables |
| `export_cad` | Copy SCAD/STL to workspace |

## Workbench

- Open `.scad` from the file explorer (double-click or **Open CAD workbench**).
- Edit SCAD, click **Render**, tune **Parameters** (uses OpenSCAD `-D` overrides).
- **Save version** / restore from sidebar history.
- **Export:** STL via Render; SCAD from editor; STEP requires optional FreeCAD path in Settings (OpenSCAD does not export STEP natively).

## Model evaluation

See `scenarios/cad/model-eval/prompts.json` and `scripts/eval-cad-models.sh` for the Phase 0 OpenSCAD compile benchmark suite.

## Related docs

- [PACKS.md](PACKS.md) — pack store install flow
- [BIOLOGY_PACK.md](BIOLOGY_PACK.md) — parallel domain pack pattern
