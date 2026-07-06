# Life sciences pack v2 — implementation plan

Detailed plan for [neural-junkie-pack-life-sciences](https://github.com/camronwood/neural-junkie-pack-life-sciences) v2.

**Parent:** [PACKS_V2_ROADMAP.md](./PACKS_V2_ROADMAP.md) · **v1 doc:** [BIOLOGY_PACK.md](./BIOLOGY_PACK.md) · **Customer boundary:** [PACKS_CUSTOM.md](./PACKS_CUSTOM.md)

**Last updated:** July 2026  
**Target pack version:** `2.0.0`

---

## North star

| v1 | v2 |
|----|-----|
| One **BiologyExpert** + two MCP tools (`analyze_sequence`, `fold_protein`) | **Workbench + runbooks + optional cheminformatics** |
| HF token required for every fold | **Local fold path** (sidecar) with HF as fallback |
| PDB written to disk; open externally | **In-app structure viewer** (PDB/mmCIF) |
| Sequence-review runbook in customer packs only | **Official runbook tier** in the pack zip |
| Scan/QC/Phoenix in customer sideload packs | **Unchanged** — org LIMS stays customer-owned |
| Biology MCP + agent in core (`internal/mcp/biology`, `builtin/biology`) | **Pack-owned sidecar** + assets; core keeps thin plumbing only |

Research and education only — not for clinical diagnosis or patient care.

---

## Current state (v1)

### Pack repo (thin manifest)

```
pack.yaml          # agents, models, mcp_agents — no capabilities, no assets
Makefile
scripts/verify-pack.sh
scripts/build-pack-zip.sh
```

### Core-owned domain logic

| Piece | Location |
|-------|----------|
| BiologyExpert agent | `internal/agent/specialized_agents.go` → `builtin/biology` |
| MCP server (port 8091) | `internal/mcp/biology/*` |
| `analyze_sequence`, `fold_protein` | Always available when life-sciences pack on |
| Scan/QC tools (`summarize_scan_*`, `run_12plex_qc`, …) | Registered in core; **gated** by customer pack `capability_defs` |
| Settings UI | **Settings → Life sciences tools** |
| Models | `koesn/llama3-openbiollm-8b`, `qwen3.5:9b` tool runner, optional `nj-biology:8b` LoRA |

### v1 explicit out-of-scope (becomes v2 backlog)

- SMILES / RDKit
- scRNA / h5ad
- ESM3, ProtGPT2, GenSLMs
- In-app PDB viewer

---

## v2 target layout

Follow the [standard v2 pack layout](./PACKS_V2_ROADMAP.md#standard-v2-pack-layout) (pattern: [web-browser](https://github.com/camronwood/neural-junkie-pack-web-browser), [aws](https://github.com/camronwood/neural-junkie-pack-aws), [incident-management](https://github.com/camronwood/neural-junkie-pack-incident-management)).

```
pack.yaml
assets/
  WORKSPACE.md
  severity-rubric.md              # optional — research-risk disclaimers, not incident P0–P4
  hub/
    server.py
    biology_common.py
    routes/
      __init__.py
      fold.py                     # local ESMFold / ColabFold
      cheminformatics.py          # RDKit (optional)
      lookup.py                   # BLAST, pathway DB
  runbooks/
    sequence-review.md
    basic-qc.md                   # generic in-silico QC, not 12-Plex SOP
  runbook-templates/
    sequence-review.json          # migrate from customer-lab fixture
    basic-qc.json
  eval/
    biology.yaml                  # keyword + tool-call benchmarks
scenarios/
  sequence-fold-viewer-smoke.json
  collab/structure-review-smoke.json
scripts/
  setup-biology-sidecar.sh
  verify-pack.sh
  pack-smoke.sh
  eval-bio-models.sh
```

---

## Specialist roster

Split **sequence interpretation / genomics reasoning** from **structure / cheminformatics** so each agent has a tight tool surface and consult triggers.

### Recommended default (two specialists)

| Agent | `type` | Owns | MCP tools |
|-------|--------|------|-----------|
| **GenomicsExpert** | `genomics` | FASTA/DNA/RNA/protein sequence analysis, mutation interpretation, pathway *discussion*, official QC runbooks | `analyze_sequence`, `blast_search`, `pathway_lookup` |
| **StructuralBiologyExpert** | `structural-biology` | Folding, PDB/mmCIF, structure confidence, viewer workbench | `fold_protein`, `structure_metadata` (pLDDT summary from PDB) |

**BiologyExpert** remains as a **compat alias** for v1 channels and `/create-expert biology` — routes to GenomicsExpert compose defaults, or a thin wrapper that consults both. Deprecate in pack `2.1.0` docs, remove in `3.0.0`.

### Optional third specialist (cheminformatics tier)

| Agent | When enabled | Tools |
|-------|--------------|-------|
| **ChemInformaticsExpert** | User runs `setup-biology-sidecar.sh` with RDKit; pack setting `biology_rdkit_enabled` | `validate_smiles`, `mol_descriptors`, `substructure_match` |

Gate ChemInformaticsExpert behind `biology-tools` sidecar health — same pattern as Playwright in web-browser v2.

### Consult / handoff wiring

Document in `assets/WORKSPACE.md`:

```
User asks about peptide mutation impact
  → GenomicsExpert (analyze_sequence, pathway_lookup)

User asks to fold and inspect structure
  → StructuralBiologyExpert (fold_protein → open PDB in workbench)

User asks about small-molecule liability on a target
  → GenomicsExpert consults ChemInformaticsExpert (if RDKit enabled)

Customer lab 12-Plex QC
  → NOT official pack — customer sideload pack + BiologyExpert/GenomicsExpert with scan tools
```

Cross-pack: no `requires_packs` on life-sciences (unlike web-browser → SD). Customer lab packs keep `requires_packs: [life-sciences]`.

---

## Capability tokens and `capability_defs`

### Platform tokens (new — require thin core PR)

Add to `internal/packs/capabilities.go` and desktop registry:

| Token | Enables |
|-------|---------|
| `biology-api` | Hub biology REST/MCP proxy to pack sidecar |
| `biology-workbench` | Structure viewer workbench tab (like `cad-workbench`, `html-preview`) |
| `biology-sidecar` | Pack hub sidecar process |

### Pack manifest sketch

```yaml
id: life-sciences
version: "2.0.0"
capabilities:
  - biology-api
  - biology-workbench
  - biology-sidecar
capability_defs:
  biology-sidecar:
    kind: hub-sidecar
    routes:
      - /api/biology
    sidecar:
      module: assets/hub/server.py
    settings:
      - biology_artifacts_dir
      - biology_fold_backend        # hf | local-esmfold | colabfold
      - biology_fold_local_url
      - biology_blast_db_dir
      - biology_rdkit_enabled
      - python_executable
  structure-viewer:
    kind: file-viewer
    match_glob: "**/*.{pdb,cif,mmcif}"
    viewer: nj.structure
  biology-tools:
    kind: mcp-tools
    mcp_tools:
      - validate_smiles
      - mol_descriptors
      - blast_search
      - pathway_lookup
    mcp_tools_path: assets/hub/routes/cheminformatics.py
settings_overlay:
  biology_artifacts_dir: ~/.neural-junkie/bio
  biology_fold_backend: hf
  biology_rdkit_enabled: false
  python_executable: ~/.neural-junkie/biology/venv/bin/python3
assets:
  workspace_guide: assets/WORKSPACE.md
  runbooks_glob: assets/runbooks/*.md
  runbook_templates_glob: assets/runbook-templates/*.json
```

**Note:** Until core ships `nj.structure` viewer id, use interim platform token `biology-structure-viewer` with a core-registered component — then migrate to pack-declared `file-viewer` per [PACKS_V2_ROADMAP core work #2](./PACKS_V2_ROADMAP.md#core-work-required-thin-additions-only).

---

## Structure viewer workbench

Parallel to CAD (`.scad` → Three.js STL) and web-browser (`.html` → live preview).

### Desktop behavior

1. Double-click `*.pdb`, `*.cif`, `*.mmcif` in file explorer → open **Structure workbench** tab.
2. Split layout: Monaco (text) + **3D viewport** (Mol* or 3Dmol.js — match bundle size / license constraints).
3. Toolbar: cartoon / surface / stick, chain picker, measure distance (v2.1), export screenshot.
4. **Open from fold** — when `fold_protein` returns a path under `biology_artifacts_dir`, hub pushes `editorStore.openStructureViewer(path)` (same pattern as CAD post-render).

### Chat context

Outbound metadata includes active structure tab (chain count, residue range) for `@StructuralBiologyExpert` — mirror `outboundChatMetadata` CAD workbench hook.

### Core dependency

| Work | Owner | Notes |
|------|-------|-------|
| `EditorTabViewMode: 'structure-workbench'` | Core desktop | One-time addition |
| `nj.structure` viewer component | Core desktop (thin) | Pack declares `file-viewer`; core hosts WebGL shell |
| Pack capability gating | Core | `hasCapability('biology-workbench')` |

---

## Biology sidecar

### Fold backends (`biology_fold_backend`)

| Mode | Requires | Behavior |
|------|----------|----------|
| `hf` (default) | HF token in Settings | Current behavior — HF Inference API |
| `local-esmfold` | GPU/CPU, `setup-biology-sidecar.sh` | `fair-esm` or compatible local weights |
| `colabfold` | Local ColabFold install | Longer sequences, MSA; document RAM/GPU mins |

Sidecar route: `POST /api/biology/fold` → writes PDB to `biology_artifacts_dir`, returns path + metadata.

MCP `fold_protein` calls sidecar when pack enabled; falls back to in-process core handler during migration window.

### Cheminformatics (optional)

`setup-biology-sidecar.sh` installs RDKit in `~/.neural-junkie/biology/venv`.

| Tool | Description |
|------|-------------|
| `validate_smiles` | Parse SMILES, return canonical form or error |
| `mol_descriptors` | MW, logP, HBD/HBA, TPSA (research triage only) |
| `substructure_match` | SMARTS query against SMILES |

### Lookup tools

| Tool | Backend |
|------|---------|
| `blast_search` | Local `blast+` against user DB **or** NCBI REST (rate-limited; document) |
| `pathway_lookup` | KEGG / Reactome read-only REST (gene ID or name → pathway summary) |

Keep network calls read-only; no API keys required for public endpoints (document rate limits in WORKSPACE).

### Migration from core MCP

**Phase 2** (after viewer + runbooks ship):

1. Move `analyze_sequence` implementation to sidecar (pure Python — port from `internal/mcp/biology/sequence.go`).
2. Move `fold_protein` to sidecar; delete HF path from core after one release overlap.
3. Leave scan/QC tool **registration** in core **or** move to customer pack repos only — preferred: customer packs own scan MCP via `mcp-tools` capability; core removes `summarize_scan_*` from biology MCP in v2.1.

---

## Official runbook tier

Ship in pack assets; importable from Runbook library without a customer pack.

### `sequence-review` (promote from customer fixture)

Source: `internal/packs/testdata/customer-lab-pack/assets/runbook-templates/sequence-review.json`

Tasks unchanged:

1. Interpret sequence (`analyze_sequence`)
2. Structure prediction (`fold_protein` + open in viewer)
3. Wet-lab next steps (controls, replicates — research disclaimer)

Add matching `assets/runbooks/sequence-review.md` SOP (incident-pack pattern).

### `basic-qc` (new — official tier)

Generic **in-silico** QC, not plate assays:

1. Validate input FASTA/sequence
2. Flag length/composition anomalies
3. If protein: optional fold + structure confidence review in viewer
4. Document limitations vs experimental QC

**Explicitly not in official pack:** 12-Plex SOP, Phoenix import, comparator, cumulative QC — remain in customer `brightest-bio-lab` and similar repos.

---

## Model refresh and eval

Pattern: CAD `scenarios/cad/model-eval/prompts.json` + `scripts/eval-cad-models.sh`.

### Candidate models (evaluate; pick winner for `pack.yaml` defaults)

| Role | Candidates |
|------|------------|
| Chat | `koesn/llama3-openbiollm-8b` (baseline), newer OpenBio/med LLM pulls, `nj-biology:8b` LoRA on `llama3:8b` |
| Tool runner | `qwen3.5:9b` (current), `qwen3.5:27b` for harder tool loops |
| Optional import | `nj-bio:8b` HF GGUF (branded tag) |

### Benchmark suite (`assets/eval/biology.yaml`)

Expand specialist-tuning's minimal two-question file:

| Category | Examples |
|----------|----------|
| Knowledge | central dogma, reading frames, enzyme classes |
| Tool routing | "analyze this DNA", "fold this peptide" → expect tool call |
| Structure literacy | interpret pLDDT bands, when *not* to fold |
| Safety | refuse clinical diagnosis requests |

`scripts/eval-bio-models.sh` — score keyword hits + optional tool-call assertions via hub smoke harness.

Publish Phase 0 leaderboard in pack README; data-driven default for compose block in `pack.yaml`.

---

## Scenarios and smoke

| Scenario | Gate |
|----------|------|
| `sequence-fold-viewer-smoke.json` | `analyze_sequence` → `fold_protein` → viewer opens PDB |
| `collab/structure-review-smoke.json` | Runbook `sequence-review` end-to-end with GenomicsExpert |
| `make pack-smoke` | Requires Ollama + optional HF token or local fold |

`make verify` — pack.yaml schema, capability_defs, asset globs, sidecar module paths.

---

## Core platform work (blocking)

Ordered by dependency:

1. **Structure workbench view mode** — desktop tab type + `nj.structure` viewer shell.
2. **Pack-declared biology platform tokens** — `biology-api`, `biology-workbench`, `biology-sidecar`.
3. **Sidecar MCP bridge** — hub proxies biology MCP tools to pack sidecar (same as AWS typed tools migration).
4. **Agent implementation indirection** — `implementation: pack/genomics` or multi-agent manifest without new `builtin/*` per specialist (stretch; can ship v2 with `builtin/genomics` + `builtin/structural-biology` if needed).
5. **Post-open hook** — fold result auto-opens structure workbench.

No new domain features in core beyond viewer shell and routing.

---

## Phased delivery

Aligned with [PACKS_V2_ROADMAP sequencing](./PACKS_V2_ROADMAP.md#suggested-sequencing) (life sciences = priority 3, Phase 1 viewer).

### Phase 0 — pack skeleton (1 PR)

- [ ] `assets/WORKSPACE.md`, `Makefile` targets (`verify`, `pack-smoke`)
- [ ] Promote `sequence-review` runbook template + markdown
- [ ] `pack.yaml` v2.0.0 manifest stub (capabilities declared; sidecar optional)
- [ ] Link from BIOLOGY_PACK.md → this doc

### Phase 1 — structure viewer (visible win)

- [ ] Core: `structure-workbench` + `nj.structure` viewer
- [ ] Pack: `structure-viewer` `file-viewer` capability_def
- [ ] Open PDB from `~/.neural-junkie/bio/` and workspace
- [ ] Scenario: `sequence-fold-viewer-smoke.json`
- [ ] Update BIOLOGY_PACK.md smoke checklist (in-app viewer step)

### Phase 2 — sidecar + local fold

- [ ] `setup-biology-sidecar.sh`, `assets/hub/server.py`
- [ ] `biology_fold_backend` setting (hf | local-esmfold | colabfold)
- [ ] Migrate `fold_protein` to sidecar; HF fallback
- [ ] `GET /api/biology/status` health check

### Phase 3 — specialists + tools

- [ ] GenomicsExpert + StructuralBiologyExpert in `pack.yaml`
- [ ] BiologyExpert compat shim
- [ ] `analyze_sequence` in sidecar
- [ ] `blast_search`, `pathway_lookup`
- [ ] Optional RDKit tools + ChemInformaticsExpert

### Phase 4 — eval + model defaults

- [ ] `assets/eval/biology.yaml` + `eval-bio-models.sh`
- [ ] Refresh `models_to_ensure` and compose defaults from benchmark
- [ ] `basic-qc` runbook

### Phase 5 — core slim-down

- [ ] Remove duplicated biology MCP from core (keep scan gating hooks until customer packs absorb)
- [ ] Core BIOLOGY_PACK.md becomes summary; pack repo owns WORKSPACE + release notes

---

## Customer pack boundary (unchanged)

| Official life-sciences v2 | Customer sideload pack |
|---------------------------|-------------------------|
| Sequence analysis, folding, structure viewer | Phoenix TIM import (`phoenix-import`) |
| Generic sequence-review + basic in-silico QC runbooks | 12-Plex QC SOP, `run_12plex_qc` |
| Pathway *lookup* (public DBs) | Org-specific analyte panels, LIMS paths |
| Research disclaimers | `cumulative_qc_dir`, org auth |

Customer packs continue to declare `requires_packs: [life-sciences]` and gate scan MCP via `capability_defs` (`scan-summary-api`, `secondary-analysis-python`, etc.) — see [PACKS_CUSTOM.md](./PACKS_CUSTOM.md).

---

## Out of scope (v2)

Carry forward to v3 or never:

- scRNA / h5ad / AnnData workflows
- ESM3, ProtGPT2, GenSLMs
- LIMS integration, ELN writeback
- Clinical decision support
- Org-specific Phoenix/12-Plex in official pack

---

## Success criteria

- [ ] `make verify` + `make pack-smoke` green in pack repo without core biology code changes per release
- [ ] Fresh install: enable life-sciences → fold peptide → **view PDB in workbench** without PyMOL
- [ ] Fold works with **local backend** and no HF token when sidecar configured
- [ ] Runbook library imports `sequence-review` from official pack zip
- [ ] Customer lab pack enables alongside v2 with no capability collision
- [ ] Eval script produces reproducible scores for ≥2 chat model candidates

---

## See also

| Doc | Purpose |
|-----|---------|
| [BIOLOGY_PACK.md](./BIOLOGY_PACK.md) | v1 user guide (update summary when v2 ships) |
| [PACK_CAPABILITY_DEFS.md](./PACK_CAPABILITY_DEFS.md) | `capability_defs` schema |
| [PACKS_CUSTOM.md](./PACKS_CUSTOM.md) | Customer lab pack boundary |
| [CAD_PACK.md](./CAD_PACK.md) | Workbench + model-eval pattern |
| [neural-junkie-pack-web-browser](https://github.com/camronwood/neural-junkie-pack-web-browser) | Sidecar + workbench reference |
