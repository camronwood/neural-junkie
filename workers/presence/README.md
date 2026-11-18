# NJ presence (anonymous install ping)

Free Cloudflare Worker that counts **unique desktop installs** that open Neural Junkie.

Clients send **only** `{ "id": "<uuid>" }` once per UTC day. No version, OS, features, or chat content.

## Deploy

```bash
cd workers/presence
npm install
npx wrangler login    # once per machine
npx wrangler kv namespace create PRESENCE_KV
# put the returned id into wrangler.toml [[kv_namespaces]].id
npx wrangler secret put PRESENCE_STATS_TOKEN
npm run deploy
```

Or from repo root:

```bash
make presence-deploy-cf
```

## Smoke

```bash
curl -s https://nj-presence.YOUR_SUBDOMAIN.workers.dev/healthz
curl -s -X POST https://nj-presence.YOUR_SUBDOMAIN.workers.dev/v1/ping \
  -H 'content-type: application/json' \
  -d '{"id":"550e8400-e29b-41d4-a716-446655440000"}'
curl -s -H "Authorization: Bearer $PRESENCE_STATS_TOKEN" \
  https://nj-presence.YOUR_SUBDOMAIN.workers.dev/v1/stats
```

## Local

```bash
npm test
npm run dev
```

Production URL: `https://nj-presence.camronwood.workers.dev` (Cloudflare account `camronwood@gmail.com`; baked into release CI as `VITE_PRESENCE_URL`).

Admin stats token (local only): `~/.config/neural-junkie/presence-stats-token`
