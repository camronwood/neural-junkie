# Slack OAuth relay setup (Cloudflare Workers)

One-time maintainer checklist for public **Connect Slack** (multi-workspace OAuth).

## 1. Deploy Worker

**First-time Cloudflare:** register a `workers.dev` subdomain before deploy succeeds. When `wrangler deploy` asks *"register a workers.dev subdomain now?"* answer **yes**, or use [Workers onboarding](https://dash.cloudflare.com/?to=/:account/workers/onboarding) in the dashboard.

```bash
cd /Users/camronwood/development/projects/neural-junkie
cd workers/slack-oauth-relay
npm install
npx wrangler login          # browser — personal Cloudflare account
npm run deploy              # complete subdomain registration if prompted
```

Or from repo root: `make slack-oauth-relay-deploy-cf`

Note the deployed URL, e.g. `https://nj-slack-oauth-relay.<subdomain>.workers.dev`.

```bash
SLACK_OAUTH_RELAY_BASE=https://nj-slack-oauth-relay.<subdomain>.workers.dev ./scripts/verify-slack-oauth-relay.sh
```

## 2. Slack app console

[api.slack.com/apps](https://api.slack.com/apps) → **Neural Junkie**:

| Step | Action |
|------|--------|
| OAuth redirect URLs | Add `{relay_base}/api/slack/oauth/callback` and `{relay_base}/api/slack/oauth/user-dm/callback` |
| Manage Distribution | Complete HTTPS checklist → **Activate Public Distribution** |
| App Directory | Do **not** submit (Socket Mode apps are not eligible) |
| Socket Mode | Leave enabled |

Optional: keep `http://localhost:18765/...` redirect URLs for local dev.

## 3. GitHub Actions + local vendor

```bash
export SLACK_VENDOR_OAUTH_RELAY_BASE=https://nj-slack-oauth-relay.<subdomain>.workers.dev
gh secret set SLACK_VENDOR_OAUTH_RELAY_BASE --repo camronwood/neural-junkie

# Local release build
export SLACK_VENDOR_CLIENT_ID=...
export SLACK_VENDOR_CLIENT_SECRET=...
export SLACK_VENDOR_APP_TOKEN=xapp-...
./scripts/ci-write-slack-vendor-oauth.sh
go build -tags slackvendor -o bin/server ./cmd/server
```

Deployed relay: `https://nj-slack-oauth-relay.neuraljunkie.workers.dev`

## 4. Verify end-to-end

1. Run NJ desktop with `slackvendor` hub
2. **Settings → Integrations → Slack → Connect Slack**
3. Approve in a **second** Slack workspace
4. Confirm success page and `GET /api/slack/connection` shows `uses_oauth_relay: true`

See [SLACK_INTEGRATION.md](SLACK_INTEGRATION.md) for full integration docs.
