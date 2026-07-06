# Software development workspace

The **Software development** pack is the layout-owning orchestration hub for engineering workflows in Neural Junkie.

## Specialists

| Specialist | Focus |
|------------|--------|
| BackendEngineer | APIs, services, Go/backend code |
| FrontendEngineer | Web UI, TypeScript, CSS |
| PlatformEngineer | CI/CD, Kubernetes, deployment |
| SecurityReviewer | Auth, OWASP, dependency audits |
| SoftwareArchitect | Boundaries, migrations, system design |
| CodeReviewer | Correctness, tests, maintainability |
| DatabaseSpecialist | SQL, schema, query tuning |
| RustExpert | Cargo, async Rust, WASM |
| SREObservabilityEngineer | Prometheus, alerts, traces, on-call |
| MobileEngineer | React Native, iOS/Android builds |
| DataMLEngineer | Notebooks, datasets, ML pipelines |

## MCP sidecar

Domain MCP tools run in the pack-owned **`sd-mcp-server`** binary (`assets/mcp/bin/`). The hub starts it when this pack is enabled. Ports **8081–8090** and **8095–8097** match the historical matrix for external MCP clients.

Build locally:

```bash
make build-mcp
```

## Pack combos (orchestration)

| Goal | Packs |
|------|--------|
| Full-stack incident | software-development + incident-management + web-browser |
| Cloud + eng | software-development + aws |
| Frontend QA | software-development + web-browser |

### Role boundaries

- **PlatformEngineer** — repo, k8s manifests, CI pipelines
- **AWSExpert** — live AWS account truth (SSO, describe APIs)
- **IncidentManager** — Jira triage, reproduction, handoff
- **WebBrowserExpert** — fetch/preview, frontend verification

### Consult patterns

- IAM or live EC2 issues → `@AWSExpert`
- Bug ticket / stack trace → `@IncidentManager` then `@BackendEngineer`
- HTML preview / localhost QA → `@WebBrowserExpert` then `@FrontendEngineer`
- Production alert → `@SREObservabilityEngineer` with `@PlatformEngineer` for rollout

## Models

- **Chat:** `qwen3.5:27b` (default)
- **Utility / tools:** `qwen3.5:9b`
- Security LoRA `nj-security:14b` remains on 14b base until specialist-tuning ships Qwen-native adapters.

## Runbooks

Pack SOPs live in `assets/runbooks/`:

- `security-review.md`
- `migration-planning.md`
- `incident-handoff.md`

Use `/runbook` in collab or reference steps in channel prompts.
