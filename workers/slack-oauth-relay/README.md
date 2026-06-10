# NJ Slack OAuth relay (Cloudflare Worker)

Public HTTPS redirect relay for Neural Junkie **Connect Slack**. Slack requires HTTPS `redirect_uri` values for public distribution; this Worker forwards the browser to the user's local hub on loopback.

No secrets are stored here. Token exchange happens on the local NJ hub.

## Deploy

```bash
cd workers/slack-oauth-relay
npm install
npx wrangler login    # once per machine
npm run deploy
```

Or from repo root:

```bash
make slack-oauth-relay-deploy-cf
```

Register the printed URLs in your Slack app (**OAuth & Permissions → Redirect URLs**).

## Test

```bash
npm test
curl -s "https://YOUR_WORKER.workers.dev/healthz"
```

## Local dev

```bash
npm run dev
```
