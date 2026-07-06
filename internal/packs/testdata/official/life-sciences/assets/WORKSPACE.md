# Life sciences workspace

Research and education workflows with GenomicsExpert, StructuralBiologyExpert, and the structure viewer workbench.

## Prerequisites

1. Enable the **Life sciences** pack in **Domain packs** (⌘⇧K).
2. Pull models: `koesn/llama3-openbiollm-8b:latest` and `qwen3.5:9b`.
3. Optional: Hugging Face token in **Settings → AI & providers** for cloud ESMFold.

## Sequence workflow

1. DM **@GenomicsExpert** (or **@BiologyExpert** for v1 compat).
2. Paste a DNA/RNA/protein sequence or FASTA.
3. Ask to analyze — agent calls `analyze_sequence`.

## Structure workflow

1. DM **@StructuralBiologyExpert** with a protein sequence.
2. Ask to fold — agent calls `fold_protein` → PDB under `~/.neural-junkie/bio/`.
3. Open the PDB in the **structure workbench** (double-click `.pdb` in file explorer).

## Sidecar setup (optional local fold)

```bash
./scripts/setup-biology-sidecar.sh
```

Set **biology_fold_backend** in Domain packs settings:

| Backend | Requires |
|---------|----------|
| `hf` (default) | HF token |
| `local-esmfold` | GPU/CPU, fair-esm in sidecar venv |
| `colabfold` | Local ColabFold install |

Confirm readiness: hub proxies `GET /api/biology/status` when the pack is enabled.

## Runbooks

Import from the pack zip:

- **sequence-review** — interpret sequence, optional fold, wet-lab next steps
- **basic-qc** — generic in-silico QC (not plate assays)

## Customer lab packs

Phoenix import, 12-Plex QC, scan viewers, and comparator workflows ship in your organization's **customer sideload pack** — not in this official pack.

## Disclaimers

- Research and education only — not for clinical diagnosis or patient care.
- In silico structure predictions are not experimental structures.
