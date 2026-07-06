# AWS domain pack

Official pack id: `aws`  
Repo: [neural-junkie-pack-aws](https://github.com/camronwood/neural-junkie-pack-aws)

## What it adds (v2)

- **AWSExpert** specialist agent with workspace read for IaC correlation
- **Typed boto3 sidecar** (`/api/aws/*`) — paginated describe/list tools
- **IaC awareness** — scan Terraform/CFN/CDK and correlate vs live state
- **Cost and security lenses** — Cost Explorer, Security Hub, GuardDuty, IAM policy analyzer
- **Multi-account** — Organizations list with optional account allowlists
- **Gated write mode** — opt-in mutating tools with `confirm_token` + audit log
- **Settings → Integrations** for SSO profiles, allowlists, and write toggle

Detailed setup: pack `assets/WORKSPACE.md` after install.

## MCP tools (highlights)

| Category | Tools |
|----------|-------|
| Identity | `aws_get_caller_identity`, `aws_list_profiles`, `aws_sso_login_hint` |
| Compute / storage | `describe_ec2_instances`, `list_s3_buckets`, `get_lambda_config`, `list_lambda_functions` |
| IAM / CFN | `describe_iam_role`, `describe_cloudformation_stack`, `analyze_iam_policy` |
| IaC | `scan_iac_workspace`, `correlate_iac_resource` |
| Lenses | `get_cost_summary`, `list_security_hub_findings`, `list_guardduty_findings` |
| Multi-account | `list_organization_accounts`, `assume_account_context` |
| Write (gated) | `ec2_stop_instance`, `lambda_update_function_configuration` |
| Legacy | `aws_cli_query` (deprecated fallback) |

## Role boundaries

- **AWSExpert** — live account truth, alarms, SSO, cost/security read ops
- **PlatformEngineer** (SD pack) — repo CI/CD, k8s, Helm, IaC authoring
- **IncidentManager** — Jira triage; defer resource trace to AWSExpert

## Pack smoke

```bash
cd neural-junkie-pack-aws
make setup && make verify && make pack-smoke
```

Collab fixture: `scenarios/collab/aws-alarm-incident-handoff.json` (requires incident-management pack).

## Release

```bash
make verify && make pack-zip
git tag v2.0.0 && git push origin v2.0.0
```

Update `packs/catalog.json` when bumping versions.
