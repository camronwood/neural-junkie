# CAD Pack (v2 summary)

Neural Junkie **CAD** domain pack v2: parametric OpenSCAD design through manufacturing, local-first.

**Pack repo:** [neural-junkie-pack-cad](https://github.com/camronwood/neural-junkie-pack-cad) — owns `WORKSPACE.md`, sidecar, runbooks, eval, and smoke scenarios.

## Enable the pack

**Settings → Domain packs → Pack store** — install and enable **CAD** (v2.0.0+).

When enabled:

- **CADExpert** and **ManufacturingExpert** are added to configured specialists.
- **CAD / Mechanical design** appears in New DM and `/create-expert cad`.
- Models `qwen3.5:27b`, `qwen3.5:9b`, and optional `nj-cad:27b` are merged into **models to ensure**.
- Open `.scad` files in the **CAD workbench** (editor, params, Three.js preview, printability panel, assembly panel).
- Hub starts the **CAD pack sidecar** for `/api/cad/*` when `cad-sidecar` capability is declared.

## Prerequisites

1. Install **OpenSCAD**: [https://openscad.org](https://openscad.org)
2. Run pack setup: `scripts/setup-cad-sidecar.sh` in the pack repo (or **Settings → CAD tools** after dev-link).
3. Optional **FreeCAD** for STEP import/export and drawings.

## Specialists

| Agent | Role |
|-------|------|
| **CADExpert** | OpenSCAD authoring, render, params, assembly layout |
| **ManufacturingExpert** | Printability, mesh repair, slicer presets, G-code sanity, STEP/2D export |

## Model stack

| Layer | Technology |
|-------|------------|
| Authoring (LLM) | `qwen3.5:27b` (optional `nj-cad:27b` LoRA) |
| Tool runner | `qwen3.5:9b` |
| Geometry | OpenSCAD CLI → STL (pack sidecar) |
| Manufacturing | Pack sidecar (trimesh, optional FreeCAD) |
| Viewport | Three.js in desktop workbench |

## MCP tools

**CADExpert:** `write_openscad`, `render_openscad`, `list_openscad_params`, `export_cad`

**ManufacturingExpert:** `check_printability`, `repair_mesh`, `export_slicer_preset`, `sanity_check_gcode`, `export_step`, `export_drawing`

## Model evaluation

Phase 0 compile benchmark lives in the **pack repo**: `scenarios/model-eval/prompts.json` and `scripts/eval-cad-models.sh`. See pack README for leaderboard.

## Related docs

- [PACKS.md](PACKS.md) — pack store install flow
- [PACKS_V2_ROADMAP.md](PACKS_V2_ROADMAP.md) — CAD v2 goals
- Pack `assets/WORKSPACE.md` — full workflow guide
