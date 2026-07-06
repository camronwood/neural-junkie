# Sequence review

Standard operating procedure for GenomicsExpert / StructuralBiologyExpert (or BiologyExpert compat).

## Prerequisites

- Life sciences pack enabled
- Optional: Hugging Face token for cloud ESMFold, or biology sidecar for local fold

## Steps

### 1. Interpret sequence

- Confirm sequence type (DNA, RNA, protein) and length
- Call `analyze_sequence` when raw sequence or FASTA is provided
- Note motifs, reading frames, or mutation context

### 2. Structure prediction (protein only)

- If appropriate length and type, call `fold_protein`
- Record PDB path under `~/.neural-junkie/bio/`
- Open in structure workbench to review confidence (pLDDT if present)
- Skip with explicit reason if sequence is too long or not protein

### 3. Wet-lab next steps

- Suggest controls, replicates, and assays
- Separate in silico predictions from experimental validation
- Include research-only disclaimer — not clinical advice
