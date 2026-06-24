# Pack capability definitions (`capability_defs`)

Custom packs declare **pack-local capabilities** in `pack.yaml` under `capability_defs`. Each entry in `capabilities:` that is not an NJ **platform** token must have a matching definition.

Official domain packs continue to use platform tokens only (`ide-v2`, `cad-workbench`, etc.) — see [PACK_CAPABILITIES.md](./PACK_CAPABILITIES.md).

## Schema

```yaml
capabilities:
  - customer-pack          # platform
  - phoenix-import         # pack-local (requires def below)

capability_defs:
  <cap-id>:
    kind: <extension-kind>
    # kind-specific fields…
```

### Extension kinds

| Kind | Purpose | Key fields |
|------|---------|------------|
| `hub-sidecar` | Hub REST routes served by pack Python sidecar | `routes`, `sidecar.module` |
| `file-viewer` | Desktop opens matching workspace files in NJ viewer | `match_glob`, `viewer` |
| `toolbar-chip` | Toolbar button (often combined with `ui` on hub-sidecar) | `ui.toolbar`, `ui.modal` |
| `mcp-tools` | Biology MCP tools gated to this pack | `mcp_tools`, `mcp_tools_path` |
| `settings-schema` | Settings overlay keys surfaced in Domain packs UI | `settings` |

### Example (Brightest Bio Lab)

```yaml
capability_defs:
  phoenix-import:
    kind: hub-sidecar
    routes: [/api/phoenix]
    sidecar:
      module: assets/hub/routes/phoenix.py
    ui:
      toolbar: { id: phx, label: PHX, icon: assets/icons/phx.png }
      modal: phoenix-import
    settings:
      - phoenix_environment
      - phoenix_auth_config_path

  scan-summary-viewer:
    kind: file-viewer
    match_glob: "**/scan-export/imageMetadata.json"
    viewer: nj.scan-summary
```

## Naming: short vs qualified IDs

| Form | Example | When to use |
|------|---------|-------------|
| Short | `phoenix-import` | Default; works when only one enabled pack defines that id |
| Qualified | `brightest-bio-lab/phoenix-import` | Multi-pack installs or collision avoidance |

Hub exposes both in `GET /api/packs` → `capabilities` and `capability_registry`.

## Pack hub sidecar

Customer packs with `hub-sidecar` capabilities ship:

```
assets/hub/server.py          # entrypoint (stdlib or FastAPI)
assets/hub/routes/*.py        # route modules
assets/secondary-analysis-tools/   # optional Python tools
```

Hub starts the sidecar on pack **enable**, passes overlay settings via `NJ_PACK_SETTINGS_JSON`, and proxies matching `/api/*` routes.

Health check: `GET /health` → `200 {"ok": true}`.

## NJ built-in viewers

Pack `viewer` values reference desktop components (not pack code):

| Viewer id | Component |
|-----------|-----------|
| `nj.scan-summary` | Scan summary plate viewer |
| `nj.scan-analysis` | Scan analysis plate-map viewer |

## Validation

- `make verify` / `POST /api/packs/validate` — pack-local caps without `capability_defs` → **error**
- Enable-time collision — two enabled customer packs with the same short cap id → **error**

## Migration from legacy lab tokens

Previously, lab tokens (`phoenix-import`, `scan-summary-*`, …) were listed in NJ core. They are now **owned by the customer pack** via `capability_defs`. Update your `pack.yaml` using [internal/packs/testdata/customer-lab-pack/pack.yaml](../internal/packs/testdata/customer-lab-pack/pack.yaml) as reference.
