# Marketing campaigns

Canonical home for Neural Junkie marketing **copy** and **creatives**, organized by campaign.

```text
campaigns/<slug>/
  *.md                 # briefs, ad paste copy, LinkedIn article sources
  creatives/           # PNGs (ads, covers, carousel frames)
```

## Rules

1. **Edit here** — never hand-edit published copies under `docs/media/` or generated `docs/articles/`.
2. **One campaign folder** owns both the brief and its art.
3. After changing creatives or LinkedIn sources, run:

```bash
make gallery-sync
make articles-sync
```

## Pipeline

| Source | Command | Output |
|--------|---------|--------|
| `campaigns/*/creatives/*.png` | `make gallery-sync` | `docs/media/gallery/ads/` + gallery manifest |
| `campaigns/*/*-LINKEDIN.md` (etc.) | `make articles-sync` | `docs/articles/*.html` + covers |
| `scripts/compose-*.sh` | run per campaign | writes into `campaigns/<slug>/creatives/` |

Product screenshots stay in `assets/screenshots/` (synced into the gallery as screenshots, not ads).

## Campaigns

| Slug | Focus |
|------|--------|
| `edge-ide` | IDE for AI at the Edge |
| `ide-v4` | IDE v4 open-source |
| `nondev` | Non-developer audience + second opinion |
| `beta12` / `beta13` / `beta21` / `beta25` | Release-wave ads |
| `launch` | v1 beta launch + download creative |
| `inference-layer` / `modular-ai` / `hub` / `model-layering` / `loop-stack` / `hardware` | Architecture article series |
| `lora` / `lora-v2` / `two-tier-lora` / `mcp-lora` / `personal-learning` / `conversation-memory` | Learning / LoRA |
| `collaboration` / `collab-craft` / `solo-vs-collab-parity` / `stream-subscriptions` | Collab + streams |
| `byom` / `community` / `test-harness` / `try-local-ai` / `product` | Product feature ads |
