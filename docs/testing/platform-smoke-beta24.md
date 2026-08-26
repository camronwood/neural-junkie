# Platform smoke — v1.2.0-beta.24

Operator matrix for **Gate 5** ([#18](https://github.com/camronwood/neural-junkie/issues/18)) after P0–P4 polish.

**Installer:** [v1.2.0-beta.24](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.24) (publish GitHub Release assets before running this matrix).

**Checklist:** [stable-platform-smoke.md](stable-platform-smoke.md) — steps below add P0–P4-specific verification.

---

## Automated gates (pre-installer)

| Gate | Command / artifact | Result | Date |
|------|-------------------|--------|------|
| Go tests | `make test-go` | **PASS** | 2026-08-25 |
| Desktop tests | `cd desktop && npm test -- --run` | **PASS** (599) | 2026-08-25 |
| Desktop build | `cd desktop && npm run build` | **PASS** — main ~794 KB gzip | 2026-08-25 |
| Collab core ×2 | [p4-collab-proof-2026-08-25.md](p4-collab-proof-2026-08-25.md) | **PASS** (8/8 ×2) | 2026-08-25 |

---

## Operator install matrix

Record **PASS** / **FAIL** and notes. Minimum before stable cut: **macOS arm64** + **one of** Windows x64 or Linux x64.

| Platform | Install | Wizard / Ollama | DM smoke | P0–P4 extras | Updater N→N+1 | Result | Operator | Date |
|----------|---------|-----------------|----------|--------------|---------------|--------|----------|------|
| macOS arm64 | `.dmg` aarch64 | bundled | Assistant reply ~2 min | Lazy panels (Monaco, Domain packs); Settings → Domain packs deep-link; collab gen-error banner; load-failure toasts | Auto-download + safe restart | **PENDING** | — | — |
| Windows x64 | `.msi` or setup `.exe` | wizard | Assistant reply | Same lazy-panel spot check | N→N+1 MSI | **PENDING** | — | — |
| Linux x64 | `.deb` | wizard if needed | Assistant reply | Manual-update notice in About | N/A (manual) | **PENDING** | — | — |

### P0–P4 macOS spot checks (when installer available)

1. Open **Collaboration** panel during a planning collab — confirm gen-error / file-awaiting banners render when triggered (or after forced error in dev hub).
2. Open **Settings → Domain packs** — confirm deep-link opens Domain Packs modal.
3. Open editor / Mermaid panel — confirm lazy load (brief spinner, no white screen); trigger a panel load error and confirm toast + ErrorBoundary recovery.
4. **About** shows **1.2.0-beta.24**.

---

## Sign-off

When macOS arm64 **and** one of Windows/Linux rows are **PASS**:

1. Update Gate 5 table in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md).
2. Close [#18](https://github.com/camronwood/neural-junkie/issues/18) with links to this matrix.

Until then, #18 remains open; automated gates above are green.
