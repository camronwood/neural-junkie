# AWS domain pack

Official pack id: `aws`  
Repo: [neural-junkie-pack-aws](https://github.com/camronwood/neural-junkie-pack-aws)

## What it adds

- **AWSExpert** specialist agent
- Read-only AWS CLI MCP tools (port **8092**)
- **Settings → Integrations** panel for SSO profiles and default region

## Setup

1. Install the pack from **Settings → Domain packs → Pack store**.
2. Open **Settings → Integrations** and select your SSO profile from `~/.aws/config`.
3. Run `aws sso login --profile <profile>` in a terminal when needed.
4. Click **Test connection** to verify `sts get-caller-identity`.

## MCP tools

| Tool | Purpose |
|------|---------|
| `aws_get_caller_identity` | Verify active account/ARN |
| `aws_list_profiles` | List profiles from `~/.aws/config` |
| `aws_sso_login_hint` | Return SSO login command |
| `aws_cli_query` | Read-only CLI (describe/list/get) |

Mutating operations are blocked when `read_only` is enabled (default).

## Release

```bash
cd /Users/camronwood/development/projects/neural-junkie-pack-aws
make verify && make pack-zip
git tag v1.0.0 && git push origin v1.0.0
```

Update `packs/catalog.json` when bumping versions.
