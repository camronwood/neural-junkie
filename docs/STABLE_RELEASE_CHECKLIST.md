# Stable release checklist

Repeat this checklist before tagging **`v1.0.0`** (stable channel) or any major stable bump. See [STABLE_SCOPE.md](STABLE_SCOPE.md) for in/out of scope.

**Last soak run:** 2026-06-08 (automated gates below; manual platform matrix pending operator sign-off)

---

## Gate 1 — Known issues

- [ ] Zero **Active** or **Investigating** items in [KNOWN_ISSUES.md](KNOWN_ISSUES.md) and [known-issues.html](known-issues.html)
- [ ] Collab matrix [testing/collab-matrix.tsv](testing/collab-matrix.tsv) — all scenarios **PASS**

---

## Gate 2 — Automated regression (hub running for live scenarios)

Run from repo root:

```bash
make test-go
make collab-preflight
make chat-scenarios-regression
make test-parity-stable-restart   # optional but recommended before stable
```

| Command | Purpose | Last result |
|---------|---------|-------------|
| `make test-go` | Unit + integration Go tests | **PASS** 2026-06-08 |
| `make collab-preflight` | Hub, Ollama, agents, scenario list | Run before tag |
| `make chat-scenarios-regression` | Workspace, closure, echo regressions | Run before tag |
| `make collab-scenarios-all` | Full collab sweep (~1–3h) | Weekly during soak |
| `make test-parity-stable-restart` | Implement scenarios 3× with hub restart | Before stable cut |

---

## Gate 3 — Updater manifests

```bash
# Beta channel (current soak build)
./scripts/verify-updater-manifest.sh v1.0.0-beta.33 beta

# After v1.0.0 tag + CI completes
./scripts/verify-updater-manifest.sh v1.0.0 stable
```

---

## Gate 4 — macOS notarization

- [ ] GitHub secrets configured per [DEVELOPMENT_NOTES.md](DEVELOPMENT_NOTES.md#macos-notarization-release-ci)
- [ ] Beta tag CI produced stapled `.dmg` (check `xcrun stapler validate` on downloaded artifact)
- [ ] Clean Mac: double-click install — no Gatekeeper block, no Right-click → Open

---

## Gate 5 — Platform install smoke matrix

Record PASS/FAIL and notes. Use clean VM or spare machine where possible.

| Platform | Install artifact | Wizard Ollama | DM + reply | Updater check | PASS |
|----------|------------------|---------------|------------|---------------|------|
| macOS arm64 | `.dmg` aarch64 | bundled | | Settings → About | |
| macOS x64 | `.dmg` x64 | bundled | | | |
| Windows x64 | `.msi` | wizard winget/silent | | | |
| Linux x64 | `.deb` | wizard apt/curl | | | |

**Linux AppImage:** CI builds `.deb` only for stable releases. AppImage is not promised on the download page.

---

## Gate 6 — Documentation

- [ ] [STATUS.md](STATUS.md) — current tag and stable/beta intent
- [ ] [download.html](download.html) — asset links match tag
- [ ] [CHANGELOG.md](CHANGELOG.md) + [release-notes.html](release-notes.html) — stable section
- [ ] [STABLE_SCOPE.md](STABLE_SCOPE.md) — accurate

---

## Cut stable release

When all gates pass:

```bash
./scripts/cut-stable-release.sh              # dry-run
./scripts/cut-stable-release.sh --execute    # tag v1.0.0, push, then refresh site after CI
./scripts/update-website-release.sh v1.0.0 --bump-site v1.0.0-beta.33
```

**Beta users:** stay on beta channel until they install a stable build manually. See [RELEASE_UPDATES.md](RELEASE_UPDATES.md).

---

## Soak policy (2–4 weeks)

During soak:

1. **Feature freeze** — bugfixes and docs only on `main`
2. Ship beta tags (`v1.0.0-beta.N`) for fixes; re-run Gates 2–4
3. No stable tag until Gate 4 (notarization) and Gate 5 (platform matrix) are signed off
