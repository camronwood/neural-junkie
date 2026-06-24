# Custom packs (sideload)

Custom packs are **private zip bundles** installed in-app. They complement official domain packs (e.g. **Life sciences**) with organization-specific workspace layout, SOPs, optional toolbar chips, and hub setting defaults.

## Pack dev studio (in-app)

Desktop **Settings → Domain packs → Pack dev studio** provides an end-to-end workflow for authors:

1. **Create** — scaffold wizard writes `pack.yaml`, `assets/WORKSPACE.md`, optional `assets/runbooks/`, and optional sidebar chip (`capability_defs.pack-toolbar`).
2. **Edit** — YAML manifest editor with live validation (capabilities, overlays, assets, requires_packs).
3. **Dev link** — point at a local folder; **Link & sync** copies into `~/.neural-junkie/packs/<id>/` without rebuilding a zip. Use **Reload** after edits.
4. **Test** — enable/disable toggle, custom-pack workspace guide preview, capability checklist.
5. **Release** — **Validate zip** or **Build release zip & smoke install** before distributing the artifact.

Hub APIs (hub access required): `POST /api/packs/validate`, `POST /api/packs/dev-link`, `POST /api/packs/dev-reload`, `POST /api/packs/dev-unlink`.

Use **dev link** while iterating; use **zip validate + sideload** for the final artifact you ship to analysts.

Pack dev studio scaffolds **generic** custom packs (workspace guide, `customer-pack`, optional toolbar chip). Advanced capabilities (**Phoenix import**, **secondary-analysis-api/viewer/python**, `secondary_analysis_tools_path`, `cumulative_qc_dir`, etc.) are authored in your org's **private pack repository** (e.g. `neural-junkie-brightest-bio-lab`) with **`capability_defs`** — see [PACK_CAPABILITY_DEFS.md](./PACK_CAPABILITY_DEFS.md).

## Install (zip)

1. Build or obtain a pack zip (see below), or use Pack dev studio **Build release zip**.
2. Desktop **Settings → Domain packs → Install custom pack (zip…)**.
3. Enable **Life sciences** (or other required packs listed in `requires_packs`).
4. Enable the custom pack.

Hub API: `POST /api/packs/install-zip` with JSON `{ "pack_zip_base64": "..." }`.

Installed to `~/.neural-junkie/packs/<id>/`. Custom packs appear in the pack list with a **Custom** badge.

## pack.yaml (custom)

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
assets:
  workspace_guide: assets/WORKSPACE.md
  runbooks_glob: assets/runbooks/*.md
capability_defs:
  pack-toolbar:
    kind: toolbar-chip
    ui:
      toolbar:
        id: acme-chip
        label: ACM
        # icon: assets/icons/acme.png
```

Lab-specific capabilities (`phoenix-import`, scan viewers, secondary analysis) live in org packs such as **brightest-bio-lab** — see that repository's `pack.yaml`.

- **`requires_packs`**: must be installed **and enabled** before the custom pack can be enabled.
- **`settings_overlay`**: applied to hub biology MCP settings on enable; reverted on disable.
- **`capability_defs.*.ui.toolbar`**: declares a chip in the **Custom pack tools** toolbar section (≤3 letter label and/or pack-relative icon path).

## Test fixture

Unit tests use a minimal fixture at `internal/packs/testdata/customer-lab-pack/`.

## Org-owned repos

Each custom pack should live in its **own private repository**. CI can run the same zip layout and distribute the artifact to analysts for sideload install.

Official catalog packs (`packs/catalog.json`) are unchanged; custom packs are not listed there unless you host a separate private catalog later.

## Toolbar chips

When a custom pack declares `capability_defs.<id>.ui.toolbar`, a chip appears in the **Custom pack tools** section of the chat toolbar when that pack is enabled.

Example (Brightest Bio **brightest-bio-lab** pack):

```yaml
capability_defs:
  phoenix-import:
    kind: hub-sidecar
    ui:
      toolbar: { id: phx, label: PHX }
      modal: phoenix-import
```

Click **PHX** → sign in with device code → browse TIM analyses and scan results → download into the active workspace.

Pack assets (icons) are served at `GET /api/packs/<id>/asset?path=assets/icons/chip.png`.

## API reference

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/packs/install-zip` | Install custom pack from base64 zip |
| GET | `/api/packs/customer-context` | Enabled custom packs + workspace guides |
| GET | `/api/packs/<id>/asset?path=…` | Pack-relative asset (toolbar icon, etc.) |
