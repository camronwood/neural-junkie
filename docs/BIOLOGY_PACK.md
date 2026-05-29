# Life Sciences / Biology Pack (v1)

Neural Junkie includes a **Life sciences** setup path with a domain-tuned model and **BiologyExpert** agent.

## One pack at a time

Install and enable this pack from **Settings → Domain packs → Pack store**. Multiple packs can be enabled together; layout is determined by the first pack you turn on.

## What you get

| Piece | Description |
|-------|-------------|
| **OpenBioLLM 8B (chat)** | `koesn/llama3-openbiollm-8b:latest` — recommended Ollama Hub pull (Llama 3 template) |
| **Tool runner** | `qwen2.5:7b` — hub uses this for MCP `analyze_sequence` / `fold_protein` when the chat model has no native tools |
| **nj-bio:8b (optional)** | HF GGUF import with Llama 3 template (branded tag) |
| **BiologyExpert** | Preset agent with bio MCP tools |
| **analyze_sequence** | DNA/RNA/protein checks, length, reverse complement |
| **fold_protein** | ESMFold via Hugging Face Inference → PDB under `~/.neural-junkie/bio/` |
| **Sequence review runbook** | Import from Runbook templates |
| **Scan summary viewer** | Open Phoenix-style exports (`imageMetadata.json` + well TIFFs A1–H12) from the file explorer |
| **summarize_scan_summary** | BiologyExpert MCP tool for QC stats on a scan summary folder |
| **Scan analysis viewer** | Open Phoenix-style analysis exports (`reports/results.json`, plots, per-analyte CSVs) with plate heat maps and QC tables |
| **summarize_scan_analysis** | BiologyExpert MCP tool for analysis QC (LOQ counts, dilution factor, analyte list) |

## Enable the pack

**Settings → AI & providers → Domain packs** — toggle **Life sciences** on or off.

When enabled:

- **BiologyExpert** is added to configured specialists (hub restarts agents automatically).
- **Biology / Life sciences** appears in **New DM** and channel expert invite lists.
- `koesn/llama3-openbiollm-8b:latest`, `qwen2.5:7b`, and optional `nj-bio:8b` are merged into **models to ensure** for Ollama.
- **Scan summary viewer** — add a workspace folder containing `imageMetadata.json` and extensionless well TIFFs; open the metadata file or use **Open scan summary** in the file explorer context menu.
- **Scan analysis viewer** — add a workspace folder containing `reports/results.json` (and optional `plots/`); open results or use **Open scan analysis** in the file explorer. Link to a scan folder to jump from concentration data to well TIFF images.

When disabled, pack-owned agents are stopped and removed from the hub. In-process engineering specialists are controlled by the separate [Software development pack](SOFTWARE_DEVELOPMENT_PACK.md); **Moderator**, **Assistant**, and CLI agents are always available.

You can also enable the pack via the **Life sciences** setup wizard track (same toggle in `config.json` under `packs.enabled["life-sciences"]`).

## Install models

### Recommended (Ollama Hub)

```bash
ollama pull koesn/llama3-openbiollm-8b:latest
ollama pull qwen2.5:7b
```

Or use **Model library** (⇧⌘M) → **Ollama** tab → **OpenBioLLM 8B (Llama 3)**.

### First-run wizard

1. Choose **Life sciences & lab work** on the Focus step.
2. Pick **Local Models** (Ollama) or **Cloud** (HF token for hosted OpenBioLLM).
3. Hub ensures **koesn** + **qwen** pulls when Ollama is running.

### Optional: HF GGUF import (`nj-bio:8b`)

1. Open **Hugging Face** tab → **Neural Junkie Bio 8B (GGUF)**.
2. Download the Q4_K_M file.
3. **Import to Ollama** → tag `nj-bio:8b` (imports with Llama 3 chat template).

## Clear a polluted DM thread

Open **channel info** (ℹ️ on the channel header) → **Clear message history**. This wipes hub history for that channel and broadcasts a resync so agents do not replay old errors on restart.

Use this after debugging bad `nj-bio` sessions or instruction-echo replies.

## Settings (no env vars required)

| Setting | Location |
|---------|----------|
| Life sciences pack | **Settings → AI & providers → Domain packs** |
| Hugging Face token (ESMFold + downloads) | **Settings → AI & providers → Hugging Face hub token** (or a `huggingface` provider) |
| Max fold/analyze length, ESMFold model, artifacts dir | **Settings → AI & providers → Life sciences tools** (when pack is on) |
| MCP master switch | Same section — **Enable MCP tool servers** |

Biology MCP starts automatically when the life-sciences pack is on and BiologyExpert is enabled. Stored in `~/.neural-junkie/config.json` under `mcp` and `hf`.

## Tools and Ollama

OpenBio chat models (`koesn/…`, `nj-bio:8b`) do not expose Ollama **tools** capability. BiologyExpert still runs MCP tools: the hub routes tool loops through **`qwen2.5:7b`** on the same Ollama endpoint while keeping **koesn** (or your configured bio tag) for normal chat replies.

With **[Agent delegation](DELEGATION.md)** enabled, **any** hub agent can consult BiologyExpert (or OpenBio via model consult) for bio-heavy questions and synthesize one reply.

Ensure `qwen2.5:7b` is pulled when using the life-sciences pack.

## Create BiologyExpert later

```
/create-expert biology
/create-expert biology MyBioCoach ollama koesn/llama3-openbiollm-8b:latest
```

## Disclaimers

- **Research and education only** — not for clinical diagnosis, treatment, or patient care.
- **In silico** structure predictions are not experimental structures.
- OpenBioLLM and ESMFold outputs may contain errors; validate in the lab.

## Discovering tools

Open **BiologyExpert** via agent **ℹ️** (Tools & models), check the **tool count** badge in the sidebar, or run **`/tools-list`** in a channel where BiologyExpert is a member. See [MCP_INTEGRATION.md](MCP_INTEGRATION.md).

## Scan summary viewer

Phoenix-style **scan summary** exports are folders with:

- `imageMetadata.json` — per-well stage position, FOV, and spot layout (analyte, grid row/column, pixel coordinates)
- Extensionless well files `A1` … `H12` — 32-bit grayscale TIFF images

With the Life sciences pack enabled:

1. Add the unzipped summary folder as a **workspace** in the file explorer.
2. Click `imageMetadata.json` or right-click the folder → **Open scan summary**.
3. Use the plate grid to select wells; the viewer decodes TIFF → PNG and overlays spots by analyte.

BiologyExpert can run **`summarize_scan_summary`** with the folder path (or path to `imageMetadata.json`) for QC markdown (well counts, analyte distribution, wells missing spots).

## Scan analysis viewer

Phoenix-style **scan analysis** exports are folders with:

- `reports/results.json` — canonical analysis data (concentrations, standard/unknown QC, spot intensities, LOQ, fit parameters)
- `reports/process_report.txt` — human-readable analysis log
- `reports/{analyte}_summary_report.csv` — per-analyte CSV exports
- `plots/{analyte}_calibration_curve.jpg` and `plots/{analyte}_heat_map.jpg` — visualization plots

With the Life sciences pack enabled:

1. Add the analysis export folder as a **workspace** (or a combined folder with both scan TIFFs and `reports/`).
2. Click `reports/results.json` or right-click the folder → **Open scan analysis**.
3. Use analyte tabs and plate grid modes (concentration, intensity, LOQ, well type).
4. **Link scan folder** (or auto-link when scan metadata is co-located) → **View scan image** jumps to the TIFF viewer for the same well.
5. From the scan summary viewer, **Analysis at well** shows concentrations when analysis is linked.

BiologyExpert can run **`summarize_scan_analysis`** with the analysis folder path, `reports/results.json`, or a `reports/{analyte}_summary_report.csv` file for QC markdown (LOQ pass/fail counts, dilution factor reminder, process report excerpt). When a sibling **`scan-export/`** folder exists, the summary includes the linked scan path for **`summarize_scan_summary`**.

## Smoke test checklist

1. Life sciences wizard → Ollama → enable BiologyExpert.
2. **Clear message history** on an old Biology DM if replies were echoing instructions.
3. DM with BiologyExpert: paste a short peptide → ask to analyze sequence.
4. Ask to fold the same sequence (HF hub token saved in Settings) → confirm PDB path in reply.
5. Runbook → **sequence-review** → instantiate with BiologyExpert → start execution.
6. Add a scan summary folder as a workspace → open viewer → confirm well A1 image and spot overlay.
7. Ask BiologyExpert to summarize the same folder path → confirm `summarize_scan_summary` output.
8. Add a scan analysis folder (or `summary (5)`-style export) → open analysis viewer → confirm IL-6 plate map and QC tables.
9. Link scan + analysis → well click → **View scan image** opens TIFF; scan sidebar shows concentrations.
10. Ask BiologyExpert to summarize analysis path → confirm `summarize_scan_analysis` output with dilution note.

## Out of scope (v1)

- SMILES validation (RDKit)
- scRNA / h5ad
- ESM3, ProtGPT2, GenSLMs
- In-app PDB viewer
