# Custom domain — www.neuraljunkie.com

Neural Junkie marketing site canonical URLs use **https://www.neuraljunkie.com**. GitHub Pages also serves the site from `https://camronwood.github.io/neural-junkie/` until DNS is live.

## 1. GitHub Pages settings

1. Open **Settings → Pages** on [camronwood/neural-junkie](https://github.com/camronwood/neural-junkie/settings/pages).
2. Confirm **Source** is `main` / `/docs`.
3. Under **Custom domain**, enter `www.neuraljunkie.com` and save.
4. Wait for DNS check, then enable **Enforce HTTPS**.

The repo includes `docs/CNAME` with `www.neuraljunkie.com` so merges keep the domain configured.

## 2. DNS (at your registrar)

Add a **CNAME** record:

| Type  | Name | Value                    | TTL  |
|-------|------|--------------------------|------|
| CNAME | www  | camronwood.github.io     | 3600 |

Optional apex redirect (`neuraljunkie.com` → `www`):

| Type | Name | Value              |
|------|------|--------------------|
| A    | @    | 185.199.108.153    |
| A    | @    | 185.199.109.153    |
| A    | @    | 185.199.110.153    |
| A    | @    | 185.199.111.153    |

Then in GitHub Pages, add `neuraljunkie.com` as a second custom domain or configure registrar forwarding to `https://www.neuraljunkie.com`.

## 3. Verify

```bash
# DNS
dig +short www.neuraljunkie.com CNAME

# HTTPS + redirects
curl -sI https://www.neuraljunkie.com/ | head -5
curl -sI https://camronwood.github.io/neural-junkie/ | head -5
```

## 4. Google Search Console

1. Add property **https://www.neuraljunkie.com**
2. Verify via DNS TXT or HTML file
3. Submit sitemap: **https://www.neuraljunkie.com/sitemap.xml**

Regenerate sitemap after adding pages:

```bash
make site-seo-sync
```

## 5. Related files

| File | Purpose |
|------|---------|
| `docs/CNAME` | GitHub Pages custom domain |
| `docs/sitemap.xml` | Generated URL list |
| `docs/robots.txt` | Crawler rules + sitemap pointer |
| `scripts/site_nav.py` | `SITE_BASE_URL`, canonical + OG tags |
| `scripts/generate-sitemap.py` | Sitemap + robots generator |
| `scripts/sync-github-repo-metadata.sh` | GitHub topics + descriptions |
