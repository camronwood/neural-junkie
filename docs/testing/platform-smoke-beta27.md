# Platform smoke — v1.2.0-beta.27

Operator matrix for **Gate 5** ([#18](https://github.com/camronwood/neural-junkie/issues/18)) after P7 structure (Slack/Assistant chatAPI domains + ChatWindow command-palette extract).

**Installer:** [v1.2.0-beta.27](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.27) — **published** 2026-09-04 (Release CI [33883905491](https://github.com/camronwood/neural-junkie/actions/runs/33883905491)).

**Checklist:** [stable-platform-smoke.md](stable-platform-smoke.md) — steps below add P0–P7-specific verification.

---

## Automated gates (pre-installer)

| Gate | Command / artifact | Result | Date |
|------|-------------------|--------|------|
| Go tests | `make test-go` | **PASS** | 2026-09-09 |
| Desktop tests | `cd desktop && npm test -- --run` | **PASS** (627 as of P8) | 2026-09-09 |
| Desktop build | `cd desktop && npm run build` | **PASS** — main ~794 KB gzip | 2026-08-25 |
| Collab core ×2 | [p4-collab-proof-2026-08-25.md](p4-collab-proof-2026-08-25.md) | **PASS** (8/8 ×2) | 2026-08-25 |
| GitHub Release publish | tag `v1.2.0-beta.27` assets + SHA256SUMS | **PASS** | 2026-09-04 |

---

## Package verification (agent)

| Check | Result | Notes | Date |
|-------|--------|-------|------|
| Assets present | **PASS** | `.dmg` aarch64/x64, `.msi`, `.deb`, updater tarballs + sigs | 2026-09-04 |
| Updater beta promote | **PASS** | CI commit `dbc99804` synced `updater/beta/update-*.json` to beta.27 | 2026-09-04 |
| macOS aarch64 DMG checksum + mount | **PASS** | SHA256 `4e05402e4cf63e939141e9e6620966d9a16abd300d2cdadb4080196b91a6f854` matches; CFBundleShortVersionString=`1.2.0-beta.27`; ad-hoc signed arm64 | 2026-09-09 |

---

## Operator install matrix

Record **PASS** / **FAIL** and notes. Minimum before stable cut: **macOS arm64** + **one of** Windows x64 or Linux x64.

| Platform | Install | Wizard / Ollama | DM smoke | P0–P7 extras | Updater N→N+1 | Result | Operator | Date |
|----------|---------|-----------------|----------|--------------|---------------|--------|----------|------|
| macOS arm64 | `.dmg` aarch64 | bundled | Assistant reply ~2 min | Lazy panels; Settings → Domain packs; collab banners; About = beta.27 | Auto-download + safe restart | **PENDING** interactive | — | — |
| Windows x64 | `.msi` | wizard | Assistant reply | Lazy-panel spot check | N→N+1 MSI | **PENDING** | — | — |
| Linux x64 | `.deb` | wizard if needed | Assistant reply | Manual-update notice in About | N/A (manual) | **PENDING** | — | — |

### P0–P7 macOS spot checks (when installer available)

1. Open **Collaboration** panel during a planning collab — confirm gen-error / file-awaiting banners render when triggered (or after forced error in dev hub).
2. Open **Settings → Domain packs** — confirm deep-link opens Domain Packs modal.
3. Open editor / Mermaid panel — confirm lazy load (brief spinner, no white screen); trigger a panel load error and confirm toast + ErrorBoundary recovery.
4. **About** shows **1.2.0-beta.27**.
5. Command palette (`⌘K` / slash) opens and local `/nj-open-*` shortcuts still work.

---

## Sign-off

When macOS arm64 **and** one of Windows/Linux rows are **PASS**:

1. Update Gate 5 table in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md).
2. Close [#18](https://github.com/camronwood/neural-junkie/issues/18) with links to this matrix.

Until then, #18 remains open; automated gates and release publish are green. Interactive install smoke is the remaining Gate 5 gap.
