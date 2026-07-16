# Gallery media (GitHub Pages)

Images here are published at **https://camronwood.github.io/neural-junkie/gallery/**

## Add new images

Prefer adding source creatives under `campaigns/<slug>/creatives/`, then sync.

For one-off gallery-only media, drop PNG or JPG files into:

| Folder | Use for |
|--------|---------|
| `ads/` | Marketing ads (usually synced from campaigns) |
| `screenshots/` | Desktop / product screenshots |
| `misc/` | Anything else (demos, Slack shots, etc.) |

Regenerate the gallery index:

```bash
./scripts/sync-gallery.sh
```

Commit the new files under `docs/media/gallery/` and the updated `docs/gallery/manifest.json`.

## Optional sidecar metadata

Add `my-shot.png.json` next to `my-shot.png`:

```json
{
  "title": "Slack Connect flow",
  "caption": "Settings → Connect Slack after beta.13",
  "tags": ["slack", "beta.13"]
}
```

## Sync from `campaigns/`

`sync-gallery.sh` copies:

- `campaigns/*/creatives/*-ad-*.png`, `*-1200.png`, `ide-v4-*`, `edge-ide-*` → `ads/`
- `assets/screenshots/*` → `screenshots/`

Keep composing ads in `campaigns/<slug>/creatives/` and run sync before publishing.
