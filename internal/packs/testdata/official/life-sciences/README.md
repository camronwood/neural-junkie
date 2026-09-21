# Neural Junkie — Life sciences pack (v2)

Official domain pack for [Neural Junkie](https://github.com/camronwood/neural-junkie).

Install via desktop **Settings → Domain packs → Pack store**, or sideload `dist/life-sciences-<version>.zip`.

## v2 features

- **GenomicsExpert**, **StructuralBiologyExpert**, **ChemInformaticsExpert** (+ BiologyExpert compat)
- Structure viewer workbench (PDB/mmCIF)
- Biology hub sidecar (optional local fold, BLAST, pathway, RDKit)
- Official **sequence-review** and **basic-qc** runbooks

## Develop

```bash
make verify
make setup-biology-sidecar   # optional: ~/.neural-junkie/biology/venv
make pack-smoke
make pack-zip                # dist/life-sciences-<version>.zip
```

See [assets/WORKSPACE.md](assets/WORKSPACE.md) and [Neural Junkie LIFE_SCIENCES_V2.md](https://github.com/camronwood/neural-junkie/blob/main/docs/LIFE_SCIENCES_V2.md).
