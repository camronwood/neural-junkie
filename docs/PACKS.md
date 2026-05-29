# Domain packs (install from GitHub)

Official domain packs are listed in [`packs/catalog.json`](../packs/catalog.json) on the `main` branch. The hub loads that catalog (with embedded fallback) and installs pack bundles from **GitHub Releases** when `download_url` is set.

## Install flow

1. Desktop **Settings → Domain packs → Pack store** → **Install**
2. Hub `POST /api/packs/{id}/install`
3. Hub reads catalog (`GET` remote JSON, merged with embedded listing)
4. Hub downloads the pack **zip** from `download_url` (HTTPS, GitHub hosts only)
5. Extracts to `~/.neural-junkie/packs/<pack-id>/` and validates `pack.yaml`
6. If download fails (offline, release missing), **embedded builtin** copy is used for official packs

## URLs

| Item | Default |
|------|---------|
| Catalog JSON | `https://raw.githubusercontent.com/camronwood/neural-junkie/main/packs/catalog.json` |
| Override | `NEURAL_JUNKIE_PACKS_CATALOG_URL` |
| Pack zips | `https://github.com/camronwood/neural-junkie/releases/download/packs-v1.0.0/<pack-id>-1.0.0.zip` |

## Publishing pack zips

```bash
/Users/camronwood/development/sandbox/neural-junkie/scripts/build-pack-zips.sh
```

Upload `dist/packs/*.zip` to a GitHub Release tagged **`packs-v1.0.0`** (or update `download_url` / tag in `packs/catalog.json` when you bump versions).

Until that release exists, installs still work via the **builtin** fallback shipped in the hub binary.

## API

- `GET /api/packs/catalog` — store rows + `catalog_url`
- `POST /api/packs/{id}/install` — download or builtin install (does not enable)
- `PUT /api/packs/{id}` — enable/disable
- `DELETE /api/packs/{id}` — uninstall
