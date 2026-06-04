# Customer pack examples

The **Brightest Bio Lab** customer pack lives in its own repository:

**[/Users/camronwood/development/neural-junkie-brightest-bio-lab](file:///Users/camronwood/development/neural-junkie-brightest-bio-lab)**

Build and sideload:

```bash
cd /Users/camronwood/development/neural-junkie-brightest-bio-lab
make pack-zip
# dist/brightest-bio-lab-<version>.zip → Settings → Domain packs → Install custom pack
```

Unit tests use a minimal copy under `internal/packs/testdata/brightest-bio-lab/`.

See [docs/PACKS_CUSTOM.md](../docs/PACKS_CUSTOM.md).
