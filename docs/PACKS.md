# Domain packs (install from GitHub)

Official domain packs are listed in [`packs/catalog.json`](../packs/catalog.json) on the `main` branch. Each pack is maintained in its own repository and published as a GitHub Release zip. The hub loads the catalog and installs pack bundles from **`download_url`** (catalog-only; no embedded fallback).

**v2 direction:** [PACKS_V2_ROADMAP.md](./PACKS_V2_ROADMAP.md) — fat packs, thin core.

## Install flow

1. Desktop **Settings → Domain packs → Pack store** → **Install**
2. Hub `POST /api/packs/{id}/install`
3. Hub fetches catalog (`GET` remote JSON)
4. Hub downloads the pack **zip** from `download_url` (HTTPS, GitHub hosts only)
5. Extracts to `~/.neural-junkie/packs/<pack-id>/` and validates `pack.yaml`
6. Requires network for first install; already-installed packs work offline

## Updates

When a newer version is published to `packs/catalog.json`, the Pack store shows an **Update** badge and button (or **Update all** banner). Hub compares semver of the installed `pack.yaml` version against the catalog.

- `GET /api/packs/updates` — packs with updates available
- `POST /api/packs/{id}/upgrade` — re-download catalog zip, preserve enable state, restart sidecars

Dev-linked and custom packs are excluded from automatic update prompts.

## URLs

| Item | Default |
|------|---------|
| Catalog JSON | `https://raw.githubusercontent.com/camronwood/neural-junkie/main/packs/catalog.json` |
| Override | `NEURAL_JUNKIE_PACKS_CATALOG_URL` |
| Pack zips | Per-pack repo releases (see `download_url` in catalog) |

## Pack repositories

| Pack ID | Repo |
|---------|------|
| `software-development` | `camronwood/neural-junkie-pack-software-development` |
| `life-sciences` | `camronwood/neural-junkie-pack-life-sciences` |
| `cad` | `camronwood/neural-junkie-pack-cad` |
| `specialist-tuning` | `camronwood/neural-junkie-pack-specialist-tuning` |
| `aws` | `camronwood/neural-junkie-pack-aws` |
| `incident-management` | `camronwood/neural-junkie-pack-incident-management` |
| `web-browser` | `camronwood/neural-junkie-pack-web-browser` |
| `music-creation` | `camronwood/neural-junkie-pack-music-creation` |

Build and release from each pack repo:

```bash
cd /Users/camronwood/development/projects/neural-junkie-pack-life-sciences
make verify
make pack-zip
git tag v1.0.0 && git push origin v1.0.0
```

Bump `version` in `pack.yaml`, update `packs/catalog.json` `download_url` / version when publishing.

**Customer / private packs:** sideload zip via Settings → Domain packs. See [PACKS_CUSTOM.md](./PACKS_CUSTOM.md).

**Capability tokens:** full registry of `capabilities:` flags and what they gate — [PACK_CAPABILITIES.md](./PACK_CAPABILITIES.md).

## API

- `GET /api/packs/catalog` — store rows + `catalog_url`
- `POST /api/packs/{id}/install` — download from catalog (does not enable)
- `POST /api/packs/install-zip` — install customer pack from base64 zip (see [PACKS_CUSTOM.md](./PACKS_CUSTOM.md))
- `GET /api/packs/customer-context` — enabled customer pack workspace guides
- `PUT /api/packs/{id}` — enable/disable
- `DELETE /api/packs/{id}` — uninstall
