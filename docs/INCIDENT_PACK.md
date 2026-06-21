# Incident management domain pack

Official pack id: `incident-management`  
Repo: [neural-junkie-pack-incident-management](https://github.com/camronwood/neural-junkie-pack-incident-management)

Requires the **software-development** pack.

## What it adds

- **IncidentManager** specialist agent
- Jira Cloud MCP tools (port **8093**)
- **Settings → Integrations** panel for Jira credentials

## Setup

1. Install **software-development**, then install **incident-management**.
2. Open **Settings → Integrations** and enter Jira site URL, email, and API token.
3. Set a default project key (optional).
4. Click **Test connection** to verify API access.

Create an API token at [Atlassian account security](https://id.atlassian.com/manage-profile/security/api-tokens).

## MCP tools

| Tool | Purpose |
|------|---------|
| `jira_get_issue` | Fetch issue by key |
| `jira_search_issues` | JQL search |
| `jira_add_comment` | Add triage comment |
| `jira_summarize_issue` | Structured triage summary |

## Release

```bash
cd /Users/camronwood/development/projects/neural-junkie-pack-incident-management
make verify && make pack-zip
git tag v1.0.0 && git push origin v1.0.0
```

Update `packs/catalog.json` when bumping versions.
