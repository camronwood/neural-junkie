# Basic in-silico QC

Generic sequence QC for research workflows — **not** plate assays or 12-Plex SOP.

## Prerequisites

- Life sciences pack enabled
- Optional: structure workbench for PDB review

## Steps

### 1. Validate input

- Run `analyze_sequence` on FASTA or raw sequence
- Record type, length, invalid characters

### 2. Structure check (protein only)

- If appropriate, run `fold_protein` and note PDB path
- Open structure in workbench; use `structure_metadata` for chain/B-factor summary

### 3. Document limitations

- In silico checks do not replace experimental QC
- Org-specific LIMS/Phoenix QC remains in customer lab packs
