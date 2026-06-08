# Customer packs (sideload)

Customer packs are **private zip bundles** installed in-app. They complement official domain packs (e.g. **Life sciences**) with customer-specific workspace layout, SOPs, and optional hub setting defaults.

## Pack dev studio (in-app)

Desktop **Settings → Domain packs → Pack dev studio** provides an end-to-end workflow for authors:

1. **Create** — scaffold wizard writes `pack.yaml`, `assets/WORKSPACE.md`, and optional `assets/runbooks/`.
2. **Edit** — YAML manifest editor with live validation (capabilities, overlays, assets, requires_packs).
3. **Dev link** — point at a local folder; **Link & sync** copies into `~/.neural-junkie/packs/<id>/` without rebuilding a zip. Use **Reload** after edits.
4. **Test** — enable/disable toggle, customer-context workspace guide preview, capability checklist.
5. **Release** — **Validate zip** or **Build release zip & smoke install** before distributing the artifact.

Hub APIs (hub access required): `POST /api/packs/validate`, `POST /api/packs/dev-link`, `POST /api/packs/dev-reload`, `POST /api/packs/dev-unlink`.

Use **dev link** while iterating; use **zip validate + sideload** for the final artifact you ship to analysts.

Pack dev studio scaffolds **generic** customer packs (workspace guide, `customer-pack`). Advanced capabilities (**Phoenix import**, **secondary-analysis-api/viewer/python**, `secondary_analysis_tools_path`, `cumulative_qc_dir`, etc.) are authored in the customer’s **private pack repository**, not in the generic scaffold wizard.

## Install (zip)

1. Build or obtain a pack zip (see below), or use Pack dev studio **Build release zip**.
2. Desktop **Settings → Domain packs → Install custom pack (zip…)**.
3. Enable **Life sciences** (or other required packs listed in `requires_packs`).
4. Enable the customer pack.

Hub API: `POST /api/packs/install-zip` with JSON `{ "pack_zip_base64": "..." }`.

Installed to `~/.neural-junkie/packs/<id>/`. Custom packs appear in the pack list with a **Custom** badge.

## pack.yaml (customer)

```yaml
id: acme-lab
version: "1.0.0"
title: Acme Lab data pack
publisher: Acme Corp
pack_kind: customer
capabilities:
  - customer-pack
  - phoenix-import
  - scan-summary-api
  - scan-summary-viewer
  - scan-analysis-viewer
  - secondary-analysis-api
  - secondary-analysis-viewer
  - secondary-analysis-python
requires_packs:
  - life-sciences
settings_overlay:
  secondary_analysis_tools_path: assets/secondary-analysis-tools
  python_executable: python3
  default_panel_profile: human-inflammatory-12plex-v1
assets:
  workspace_guide: assets/WORKSPACE.md
  runbooks_glob: assets/runbooks/*.md
```

- **`requires_packs`**: must be installed **and enabled** before the customer pack can be enabled.
- **`settings_overlay`**: applied to hub biology MCP settings on enable; reverted on disable. When the pack declares **`secondary-analysis-api`** or **`secondary-analysis-python`**, the desktop shows secondary-analysis fields under **Settings → Life sciences tools** (tools path, Python, panel profile, cumulative QC).
- Pack-relative paths in overlay values are resolved under the installed pack directory.

## Test fixture

Unit tests use a minimal fixture at `internal/packs/testdata/customer-lab-pack/`.

## Customer-owned repos

Each customer pack should live in its **own private repository**. CI can run the same zip layout and distribute the artifact to analysts for sideload install.

Official catalog packs (`packs/catalog.json`) are unchanged; customer packs are not listed there unless you host a separate private catalog later.

## Phoenix import (toolbar **PHX** chip)

When a customer pack declares **`phoenix-import`**, a **PHX** chip appears in the toolbar when that pack is installed and enabled. Generic customer packs created in Pack dev studio do not include this by default; add `phoenix-import` in the customer’s private `pack.yaml` when needed. Click to sign in (Auth0 device code in-app), browse TIM analyses and scan results, and download into the active workspace.

1. Install + enable **Life sciences** and the customer pack.
2. Click **PHX** → sign in with device code in the browser.
3. Pick an analysis or scan → **Download to workspace**.

The hub downloads `results.json`, `summary.zip`, `validation.zip` (when present), and linked `scanResults` attachments and lays out:

```
{workspace}/{plate-barcode}-summary/
  reports/results.json
  plots/          (from summary.zip)
  scan-export/    (from scan zip)

{workspace}/{plate-barcode}-validation/
  reports/validation_report.csv
  reports/allspots.csv
  images/         (annotated well JPGs from validation.zip)
  plots/          (from validation.zip)
```

Default import folder names use the plate barcode plus `-summary` / `-validation` suffixes (comparator convention). When TIM has no `validation.zip`, the hub synthesizes the CSVs from `results.json`.

Pack overlay / hub config keys:

| Key | Purpose |
|-----|---------|
| `phoenix_environment` | `staging`, `dev`, or `prod` |
| `phoenix_credentials_path` | Optional path to `credentials-{env}.json` (default: vendor CLI store) |
| `phoenix_auth_config_path` | Auth0 app creds file (client_id for token refresh), e.g. `.phoenix-customer-cli-creds` |

Hub `config.json`: `phoenix.environment`, `phoenix.credentials_path`, `phoenix.auth_config_path`.

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/packs/install-zip` | Install from base64 zip |
| GET | `/api/packs/customer-context` | Enabled customer packs + workspace guides |
| GET | `/api/phoenix/status` | TIM credentials and login state |
| POST | `/api/phoenix/login/start` | Start device code login |
| GET | `/api/phoenix/login/poll?session_id=` | Poll device authorization |
| POST | `/api/phoenix/logout` | Clear stored TIM credentials |
| GET | `/api/phoenix/analyses` | Recent analyses for browse UI |
| GET | `/api/phoenix/scan-results` | Recent scanResults for browse UI |
| POST | `/api/phoenix/import` | Download analysis into workspace NJ layout |
| POST | `/api/phoenix/import-scan` | Download scanResults zip into workspace |
