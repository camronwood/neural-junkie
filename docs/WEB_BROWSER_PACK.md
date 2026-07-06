# Web browser domain pack

Official pack id: `web-browser`  
Repo: [neural-junkie-pack-web-browser](https://github.com/camronwood/neural-junkie-pack-web-browser)

Requires: **software-development** pack.

## What it adds

- **WebBrowserExpert** specialist agent
- `fetch_url` and `web_search` MCP tools (port **8094**)
- **Playwright sidecar** — real browser automation via `/api/browser/*`
- **HTML browser workbench** — split editor + live preview for workspace `.html` files
- **Dev server URL** mode — point the preview iframe at `http://localhost:3000` (Vite, Next, etc.)
- **QA panels** (v2) — responsive presets, a11y audit, performance metrics, visual diff, DOM picker

## Setup (v2)

```bash
cd neural-junkie-pack-web-browser
./scripts/setup-playwright.sh
```

Enable the pack in **Domain packs** (⌘⇧K). The hub starts the Playwright sidecar automatically.

## HTML preview workbench

1. Install and enable the pack from **Domain packs** (⌘⇧K).
2. Open any `.html` / `.htm` file in the workspace — it opens in the HTML browser workbench automatically.
3. Edit HTML on the left; preview renders on the right via hub `/api/workspace-preview`.
4. Save (⌘S) and click **Reload preview** to refresh the file preview.
5. Switch to **Dev server URL** to preview a running local dev server instead.
6. Use **Mobile / Tablet / Desktop** presets and **A11y / Perf / Visual** panels when the sidecar is ready.

## MCP tools

| Tool | Purpose |
|------|---------|
| `fetch_url` | Fetch a URL and return readable page content |
| `web_search` | Search the web for references while building pages |
| `browser_screenshot` | Capture a PNG via Playwright sidecar |
| `browser_navigate` | Open URL and return session id |
| `browser_click` | Click element in browser session |
| `browser_fill` | Fill form field in browser session |
| `browser_a11y_audit` | Run axe-core WCAG audit |
| `browser_metrics` | Lighthouse-lite performance metrics |

## Release

```bash
cd /Users/camronwood/development/projects/neural-junkie-pack-web-browser
make verify && make pack-smoke && make pack-zip
git tag v2.0.0 && git push origin v2.0.0
```

Update `packs/catalog.json` when bumping versions.
