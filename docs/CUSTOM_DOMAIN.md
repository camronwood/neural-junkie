# Custom domain (optional)

The live site is **https://camronwood.github.io/neural-junkie/** — no domain registrar required.

Canonical URLs, sitemap, and Open Graph tags all use that GitHub Pages URL.

## If you add a custom domain later

`neuraljunkie.com` currently resolves to WordPress.com (not this repo). To use it with GitHub Pages you would need:

1. DNS control at a registrar (or transfer the domain away from WordPress)
2. `docs/CNAME` with your chosen hostname (e.g. `www.neuraljunkie.com`)
3. Update `SITE_BASE_URL` in `scripts/site_nav.py` to match
4. Run `make site-seo-sync` and `make github-metadata-sync`
5. GitHub **Settings → Pages → Custom domain** + Enforce HTTPS
6. Google Search Console with the new property + sitemap

### DNS records (when ready)

| Type  | Name | Value                |
|-------|------|----------------------|
| CNAME | www  | camronwood.github.io |

Optional apex (`@`) A records: `185.199.108.153`, `185.199.109.153`, `185.199.110.153`, `185.199.111.153`

## Google Search Console (GitHub Pages URL)

1. Add property **https://camronwood.github.io/neural-junkie/**
2. Verify via HTML file or GitHub OAuth
3. Submit sitemap: **https://camronwood.github.io/neural-junkie/sitemap.xml**

Regenerate sitemap after adding pages: `make site-seo-sync`
