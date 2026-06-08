# Customer packs

Customer packs are **private zip bundles** sideloaded in-app. Each customer maintains their own repository with `pack.yaml`, workspace guides, runbooks, and optional tool assets.

Build and sideload from the customer repo:

```bash
cd /path/to/customer-pack-repo
make pack-zip
# dist/<pack-id>-<version>.zip → Settings → Domain packs → Install custom pack
```

Unit tests use a minimal fixture at `internal/packs/testdata/customer-lab-pack/`.

See [docs/PACKS_CUSTOM.md](../docs/PACKS_CUSTOM.md).
