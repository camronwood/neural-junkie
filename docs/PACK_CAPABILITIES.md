# Pack capability tokens

Domain packs declare **capability tokens** in `pack.yaml` under `capabilities:`. When a pack is **enabled**, the hub builds a **capability registry** (platform + pack-local defs) exposed via `GET /api/packs` as `capabilities`, `capability_registry`, and `short_id_collisions`.

## Two layers

| Layer | Owner | Declared in | Examples |
|-------|-------|-------------|----------|
| **Platform** | Neural Junkie | `capabilities:` only | `customer-pack`, `ide-v2`, `cad-workbench` |
| **Pack-local** | Customer pack | `capabilities:` + **`capability_defs`** | `phoenix-import`, `scan-summary-viewer` |

Pack-local capabilities are **defined in the pack**, not in NJ core. See [PACK_CAPABILITY_DEFS.md](./PACK_CAPABILITY_DEFS.md).

## Source of truth

| File | Purpose |
|------|---------|
| [`internal/packs/capabilities.go`](../internal/packs/capabilities.go) — `PlatformCapabilityTokens`, `KnownExtensionKinds` | Platform tokens + extension kinds |
| [`desktop/src/stores/packCapabilities.ts`](../desktop/src/stores/packCapabilities.ts) | `PACK_PLATFORM_CAP`, `PACK_LOCAL_CAP` (reference ids) |
| [`desktop/src/stores/packCapabilityRegistry.ts`](../desktop/src/stores/packCapabilityRegistry.ts) | Runtime registry dispatch |

Related docs: [PACKS.md](./PACKS.md), [PACKS_CUSTOM.md](./PACKS_CUSTOM.md), [PACK_CAPABILITY_DEFS.md](./PACK_CAPABILITY_DEFS.md).

---

## Software development (`software-development`)

| Token | Official pack | What it enables |
|-------|---------------|-----------------|
| `ide-v2` | software-development | IDE layout features: file explorer integration, symbols, Problems, pending hunks, LSP-lite, git SCM panel. Gated in `ChatWindow` and layout profile (`layout_profile: ide`). See [IDE_V2.md](./IDE_V2.md). |
| `ide-v3-composer` | software-development | IDE v3 composer mode in main chat (Ask/Agent/Export chips). See [IDE_V3.md](./IDE_V3.md). |
| `git-rest` | software-development | Hub REST git endpoints (`cmd/server/git_handlers.go`). Required for programmatic git operations from the IDE. |
| `inline-completion` | software-development | Declared by the pack; Monaco ghost-text completion is tied to **software-development pack enabled** + Settings toggle (not a separate `hasCapability` check today). See [IDE_V2.md](./IDE_V2.md). |

---

## Life sciences (`life-sciences`) — v2

| Token | What it enables |
|-------|-----------------|
| `biology-api` | Hub biology REST proxy to pack sidecar |
| `biology-workbench` | Structure viewer workbench for `.pdb` / `.cif` / `.mmcif` |
| `biology-sidecar` | Pack Python sidecar (`/api/biology/*`) for fold, BLAST, pathway, optional RDKit |

Pack-local `capability_defs`: `structure-viewer`, `biology-tools`. See [LIFE_SCIENCES_V2.md](./LIFE_SCIENCES_V2.md).

Lab scan/QC/Phoenix UI remains in **customer sideload packs** (below).

---

## Customer / lab packs (private sideload)

Lab workflow capabilities are **pack-local** — declare them in `capability_defs` (not NJ platform list). Typical ids:

| Token | Kind | What it enables |
|-------|------|-----------------|
| `customer-pack` | platform | Customer-context API, workspace guide |
| `phoenix-import` | hub-sidecar | PHX chip, Phoenix APIs via pack sidecar |
| `scan-summary-api` | hub-sidecar | Scan summary REST |
| `scan-summary-viewer` | file-viewer | Scan summary plate viewer |
| `scan-analysis-viewer` | file-viewer | Scan analysis plate viewer |
| `secondary-analysis-api` | hub-sidecar | 12-Plex QC, comparator jobs |
| `secondary-analysis-viewer` | file-viewer | Secondary analysis panel hooks |
| `secondary-analysis-python` | mcp-tools | Python job runner + MCP gating |

Full schema and Brightest Bio example: [PACK_CAPABILITY_DEFS.md](./PACK_CAPABILITY_DEFS.md). Reference fixture: `internal/packs/testdata/customer-lab-pack/`.

---

## CAD (`cad`)

| Token | What it enables |
|-------|-----------------|
| `cad-api` | Hub CAD REST/MCP endpoints (`cad_handlers.go`). **Domain packs** settings link for CAD tool configuration. |
| `cad-viewer` | Declared by the CAD pack; 3D preview is part of the CAD workbench pipeline. |
| `cad-workbench` | File explorer: open `.scad` files in the **CAD workbench** (OpenSCAD preview + export). See [CAD_PACK.md](./CAD_PACK.md). |

---

## Specialist tuning (`specialist-tuning`)

| Token | What it enables |
|-------|-----------------|
| `personal-learning` | Personal learning from chat/collabs; hub learning commands; **Settings → Memory & learning**. |
| `lora-training` | Train LoRA adapters from repo/chat; **My agents**, agent info modal, `/create-expert` LoRA paths. |
| `lora-compose` | Compose Hugging Face LoRAs in Ollama; model library and command form compose options. |
| `lora-adapters` | Pack-declared LoRA adapters (`lora_adapters` in manifest) downloaded and composed at hub startup. |

---

## AWS (`aws`)

| Token | What it enables |
|-------|-----------------|
| `aws-api` | Declared by the AWS pack; AWSExpert agent + read-only AWS CLI MCP tools (port 8092). |
| `aws-sso` | **Settings → Integrations** AWS SSO profile picker and connection test. See [AWS_PACK.md](./AWS_PACK.md). |

Default overlay: `aws_default_region` (e.g. `us-east-2`).

---

## Incident management (`incident-management`)

Requires **software-development** pack.

| Token | What it enables |
|-------|-----------------|
| `incident-api` | Declared by the pack; IncidentManager agent + incident MCP tools (port 8093). |
| `jira-integration` | **Settings → Integrations** Jira URL, email, API token, default project. See [INCIDENT_PACK.md](./INCIDENT_PACK.md). |
| `incident-triage` | Declared by the pack; Jira triage MCP tools (`jira_get_issue`, `jira_search_issues`, etc.). |

---

## Web browser (`web-browser`)

Requires **software-development** pack.

| Token | What it enables |
|-------|-----------------|
| `web-browser` | Declared by the pack; **WebBrowserExpert** agent + browser MCP (`fetch_url`, web search). |
| `web-preview` | Declared by the pack; HTML/URL preview behavior in the browser workbench. |
| `web-browser-workbench` | File explorer: open workspace `.html` / `.htm` in the **HTML preview workbench**; context menu **Open in browser workbench**. See [WEB_BROWSER_PACK.md](./WEB_BROWSER_PACK.md). |

---

## Music creation (`music-creation`)

| Token | What it enables |
|-------|-----------------|
| `music-generation` | **MusicExpert** agent, `generate_music` tool, `/generate-music`. See [MUSIC_CREATION_PACK.md](./MUSIC_CREATION_PACK.md). |
| `music-workbench` | File explorer: open `.wav`, `.mp3`, or `project.nj-music.json` in the **music workbench** (waveform, loop regions, A/B compare, stems). |
| `music-sidecar` | Pack-local hub sidecar; ACE-Step inference at `/api/music/*`. |

---

## Capability → UI quick reference

| UI / behavior | Capability token(s) |
|---------------|---------------------|
| IDE layout + editor panels | `ide-v2` (+ software-development enabled) |
| Ask/Agent/Export composer chips | `ide-v3-composer` |
| PHX toolbar chip | `phoenix-import` |
| Scan summary plate viewer | `scan-summary-viewer` |
| Scan analysis plate viewer | `scan-analysis-viewer` |
| Secondary analysis panel | `secondary-analysis-viewer` (+ often `secondary-analysis-api`) |
| Life sciences tools settings (Python path, panel profile) | `secondary-analysis-api` or `secondary-analysis-python` |
| CAD workbench (`.scad`) | `cad-workbench` |
| HTML preview workbench | `web-browser-workbench` |
| Inline song player in chat | `music-generation` |
| Music workbench (audio / project) | `music-workbench` |
| AWS integrations settings | `aws-sso` |
| Jira integrations settings | `jira-integration` |
| Memory / LoRA training settings | `personal-learning`, `lora-training` |
| Pack store LoRA adapter section | `lora-adapters` |

---

## Known `settings_overlay` keys

Applied by the hub when a customer pack is **enabled**; reverted on disable. Pack-relative paths are resolved under the installed pack directory.

| Key | Applies to | Purpose |
|-----|------------|---------|
| `secondary_analysis_tools_path` | Biology MCP | Python secondary-analysis scripts directory |
| `python_executable` | Biology MCP | Python binary for secondary analysis |
| `cumulative_qc_dir` | Biology MCP | Cumulative QC output directory |
| `default_panel_profile` | Biology MCP | Default 12-plex panel profile id |
| `artifacts_dir` | Biology MCP | Artifacts directory |
| `phoenix_environment` / `environment` | Phoenix | TIM environment (`dev`, `staging`, …) |
| `phoenix_credentials_path` / `credentials_path` | Phoenix | Stored credentials file |
| `phoenix_auth_config_path` / `auth_config_path` | Phoenix | Auth0 / bbio CLI config for token refresh |
| `aws_default_region` | AWS | Default AWS region |
| `aws_profile` | AWS | Default SSO profile name |
| `aws_sso_start_url` | AWS | SSO start URL override |
| `jira_base_url` | Jira | Jira Cloud site URL |
| `jira_email` | Jira | Jira API user email |
| `jira_api_token` | Jira | Jira API token |
| `jira_default_project_key` | Jira | Default project key for searches |

Keys may be prefixed with `mcp.biology.` (e.g. `mcp.biology.secondary_analysis_tools_path`).

Full list: `KnownOverlayKeys` in [`internal/packs/validate.go`](../internal/packs/validate.go).

---

## Adding a new capability

1. Implement the UI and/or hub behavior in the main `neural-junkie` repo.
2. Add the token to `KnownCapabilityTokens` (`validate.go`) and `PACK_CAP` (`packCapabilities.ts`).
3. Gate with `hasCapability()` (desktop) and/or `AnyPackCapability()` (hub).
4. Declare the token in the relevant official `pack.yaml` (or document it for customer packs).
5. Update this file.

Pack authors **cannot** introduce new UI by declaring an unknown token alone — a hub/desktop release is required first.

---

## Validate and test

- **Pack dev studio** → live YAML validation warns on unknown capabilities and overlay keys.
- **Test panel** checklist maps common customer capabilities to expected UI (PHX chip, viewers, etc.).
- Hub: `POST /api/packs/validate` with pack zip or dev-linked folder.

```bash
# From a pack repo
make verify   # runs scripts/verify-pack.sh against hub validate rules
```
