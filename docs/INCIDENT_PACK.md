# Incident management domain pack (v2)

Official pack id: `incident-management` v2.0.0  
Repo: [neural-junkie-pack-incident-management](https://github.com/camronwood/neural-junkie-pack-incident-management)

Requires the **software-development** pack.

## What v2 adds

- **IncidentManager** with P0–P4 severity rubric (pack assets)
- **Multi-provider ticketing:** Jira, GitHub Issues, Linear (+ unified `ticket_*` tools)
- **PagerDuty / Sentry** read-only alert linking
- **Write mode + approval gates** for create/assign/transition/comment
- **Handoff runbooks** (markdown + JSON templates)
- **Stack trace ingestion** via pack hub sidecar
- **Postmortem workflow** (timeline + template draft)

Detailed setup: see pack `assets/WORKSPACE.md` after install.

## MCP tools (selection)

| Tool | Purpose |
|------|---------|
| `jira_*` / `ticket_*` | Get, search, comment, create, assign, transition |
| `pagerduty_*` | List/get incidents |
| `sentry_*` | Get issue/event |
| `incident_parse_stack_trace` | Parse trace → suspect files + repro |
| `incident_generate_postmortem` | Draft postmortem from template |

Port **8093** when MCP enabled.

## Settings

**Settings → Integrations:** Jira, GitHub Issues, Linear, PagerDuty, Sentry, and **Incident** (default provider, write mode).

## Release

```bash
cd neural-junkie-pack-incident-management
make verify && make pack-smoke && make pack-zip
git tag v2.0.0 && git push origin v2.0.0
```

Update `packs/catalog.json` when bumping versions.
