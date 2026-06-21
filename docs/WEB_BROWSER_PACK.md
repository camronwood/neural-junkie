# Web browser domain pack

Official pack id: `web-browser`  
Repo: [neural-junkie-pack-web-browser](https://github.com/camronwood/neural-junkie-pack-web-browser)

Requires: **software-development** pack.

## What it adds

- **WebBrowserExpert** specialist agent
- `fetch_url` and `web_search` MCP tools (port **8094**)
- **HTML browser workbench** — split editor + live preview for workspace `.html` files
- **Dev server URL** mode — point the preview iframe at `http://localhost:3000` (Vite, Next, etc.)

## HTML preview workbench

1. Install and enable the pack from **Domain packs** (⌘⇧K).
2. Open any `.html` / `.htm` file in the workspace — it opens in the HTML browser workbench automatically.
3. Edit HTML on the left; preview renders on the right via hub `/api/workspace-preview`.
4. Save (⌘S) and click **Reload preview** to refresh the file preview.
5. Switch to **Dev server URL** to preview a running local dev server instead.

Right-click an HTML file in the file explorer and choose **Open HTML browser** if you opened it as plain text first.

## MCP tools

| Tool | Purpose |
|------|---------|
| `fetch_url` | Fetch a URL and return readable page content |
| `web_search` | Search the web for references while building pages |

## Release

```bash
cd /Users/camronwood/development/projects/neural-junkie-pack-web-browser
make verify && make pack-zip
git tag v1.0.0 && git push origin v1.0.0
```

Update `packs/catalog.json` when bumping versions.
