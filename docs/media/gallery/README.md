# Gallery media (GitHub Pages)

Images here are published at **https://camronwood.github.io/neural-junkie/gallery/**

## Add new images

1. Drop PNG or JPG files into a folder:

| Folder | Use for |
|--------|---------|
| `ads/` | Marketing ads (1080×1080) |
| `screenshots/` | Desktop / product screenshots |
| `misc/` | Anything else (demos, Slack shots, etc.) |

2. Regenerate the gallery index:

```bash
./scripts/sync-gallery.sh
```

3. Commit the new files under `docs/media/gallery/` and the updated `docs/gallery/manifest.json`.

## Optional sidecar metadata

Add `my-shot.png.json` next to `my-shot.png`:

```json
{
  "title": "Slack Connect flow",
  "caption": "Settings → Connect Slack after beta.13",
  "tags": ["slack", "beta.13"]
}
```

## Sync from repo `assets/`

`sync-gallery.sh` also copies:

- `assets/neural-junkie-*-ad-1080.png` → `ads/`
- `assets/screenshots/*` → `screenshots/`

So you can keep composing ads in `assets/` and run sync before publishing.
