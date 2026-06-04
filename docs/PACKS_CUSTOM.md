# Customer packs (sideload)

Customer packs are **private zip bundles** installed in-app. They complement official domain packs (e.g. **Life sciences**) with customer-specific workspace layout, SOPs, and optional hub setting defaults.

## Install (zip)

1. Build or obtain a pack zip (see below).
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
requires_packs:
  - life-sciences
settings_overlay:
  secondary_analysis_tools_path: assets/secondary-analysis-tools
  python_executable: python3
assets:
  workspace_guide: assets/WORKSPACE.md
  runbooks_glob: assets/runbooks/*.md
```

- **`requires_packs`**: must be installed **and enabled** before the customer pack can be enabled.
- **`settings_overlay`**: applied to **Settings → Life sciences tools** on enable; reverted on disable.
- Pack-relative paths in overlay values are resolved under the installed pack directory.

## Reference pack

**Brightest Bio Lab** — private repo (canonical source):

`/Users/camronwood/development/neural-junkie-brightest-bio-lab`

Build zip:

```bash
cd /Users/camronwood/development/neural-junkie-brightest-bio-lab
make pack-zip
# or from neural-junkie: scripts/build-customer-pack-zips.sh
```

Unit tests use a minimal fixture at `internal/packs/testdata/brightest-bio-lab/`.

## Customer-owned repos

Each customer pack should live in its **own private repository**. CI can run the same zip layout and distribute the artifact to analysts for sideload install.

Official catalog packs (`packs/catalog.json`) are unchanged; customer packs are not listed there unless you host a separate private catalog later.

## Phoenix import (toolbar **PHX** chip)

Customer packs with `phoenix-import` show a **PHX** chip in the toolbar after install. Click to sign in (Auth0 device code in-app), browse TIM analyses and scan results, and download into the active workspace. No `bbio` CLI required.

1. Install + enable **Life sciences** and the customer pack.
2. Click **PHX** → sign in with device code in the browser.
3. Pick an analysis or scan → **Download to workspace**.

The hub downloads `results.json`, `summary.zip`, and linked `scanResults` attachments and lays out:

```
{workspace}/{output_dir}/
  reports/results.json
  plots/          (from summary.zip)
  scan-export/    (from scan zip)
```

Pack overlay / hub config keys:

| Key | Purpose |
|-----|---------|
| `phoenix_environment` | `staging`, `dev`, or `prod` |
| `phoenix_credentials_path` | Optional path to `credentials-{env}.json` (default: bbio store) |
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
