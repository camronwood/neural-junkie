# Stable release checklist

Repeat this checklist before tagging **`v1.0.0`** (stable channel) or any major stable bump. See [STABLE_SCOPE.md](STABLE_SCOPE.md) for in/out of scope.

**Last soak run:** 2026-06-09 (automated gates below; platform smoke pending operator sign-off)

**macOS notarization:** **Deferred** for initial v1.0.0 — ad-hoc signed builds with documented Gatekeeper workaround. Target **v1.0.1** for notarized macOS when Apple Developer credentials are available.

---

## Gate 1 — Known issues

- [x] Zero **Active** or **Investigating** items in [KNOWN_ISSUES.md](KNOWN_ISSUES.md) and [known-issues.html](known-issues.html)
- [x] Collab matrix [testing/collab-matrix.tsv](testing/collab-matrix.tsv) — all scenarios **PASS** (21/21 as of 2026-06-09)
- [ ] Rotate any API keys in local `env.local` if they were ever pasted into chat, docs, or shared channels (not tracked in git; operator responsibility)

---

## Gate 2 — Automated regression (hub running for live scenarios)

Run from repo root:

```bash
make test-go
make test-conversation-contract
make collab-preflight
make conversation-scenarios-regression
make test-parity-stable-restart   # recommended before stable
```

| Command | Purpose | Last result |
|---------|---------|-------------|
| `make test-go` | Unit + integration Go tests | **PASS** 2026-06-09 |
| `make test-conversation-contract` | Agent/hub/desktop chat wiring | **PASS** 2026-06-09 |
| `make collab-preflight` | Hub, Ollama, agents, scenario list | **PASS** 2026-06-09 |
| `make conversation-scenarios-regression` | 18 chat + 6 collab conversation scenarios | **PASS** 23/23 2026-06-09 |
| `make chat-scenarios-regression` | Tagged chat regressions | Run before tag |
| `make collab-scenarios-all` | Full collab sweep (~1–3h) | Optional |
| `make test-parity-stable-restart` | Implement scenarios 3× with hub restart | **PASS** 3/3 runs, 7/7 per sweep — [parity-stable-restart-2026-06-09-1723.log](testing/parity-stable-restart-2026-06-09-1723.log) |

---

## Gate 3 — Updater manifests

Fix merged: [scripts/publish-updater-manifests.sh](../scripts/publish-updater-manifests.sh) uses upload-only for existing `updater-beta` (avoids immutable-release tag recreate failures).

```bash
# Beta channel (after next beta tag + CI)
./scripts/verify-updater-manifest.sh v1.0.0-beta.33 beta

# After v1.0.0 tag + CI completes
./scripts/verify-updater-manifest.sh v1.0.0 stable
```

Verify on the tag you cut — beta.33 publish failed before the upload-only fix.

- [ ] Manifest version exactly matches the tagged bundle version on each platform
- [ ] `policy.channel`, rollout percentage, and enforcement deadline are correct
- [ ] Artifact URLs reference the immutable tag
- [ ] Release is public before `updater/beta/` advances
- [ ] `./scripts/verify-desktop-version-consistency.sh` passes

---

## Gate 4 — macOS notarization (deferred for v1.2.0 stable)

**Not required to cut v1.2.0 stable.** Target **v1.2.1** when Apple Developer credentials are available.

When Apple Developer account is available:

- [ ] GitHub secrets configured per [DEVELOPMENT_NOTES.md](DEVELOPMENT_NOTES.md#macos-notarization-release-ci)
- [ ] Beta tag CI produced stapled `.dmg` (check `xcrun stapler validate` on downloaded artifact)
- [ ] Clean Mac: double-click install — no Gatekeeper block, no Right-click → Open
- [ ] Remove `macos-adhoc-sign` from [KNOWN_ISSUES.md](KNOWN_ISSUES.md) and [known-issues.html](known-issues.html)

---

## Gate 5 — Platform install smoke matrix

Record PASS/FAIL and notes. Use existing release installers until you cut a new tag.

| Platform | Install artifact | Wizard Ollama | DM + reply | Updater check | PASS |
|----------|------------------|---------------|------------|---------------|------|
| macOS arm64 | `.dmg` aarch64 (ad-hoc) | bundled | Right-click → Open if blocked | Auto-download + safe restart | **PENDING** operator |
| macOS x64 | `.dmg` x64 (ad-hoc) | bundled | same | | optional |
| Windows x64 | `.msi` | wizard winget/silent | | | **PENDING** operator |
| Linux x64 | `.deb` | wizard apt/curl | | Manual-update notice | **PENDING** operator |

**Minimum before stable cut:** macOS arm64 (your machine) + **one of** Windows or Linux.

**Smoke steps:** See [testing/stable-platform-smoke.md](testing/stable-platform-smoke.md) and [testing/platform-smoke-beta26.md](testing/platform-smoke-beta26.md).

**Linux AppImage:** CI builds `.deb` only for stable releases. AppImage is not promised on the download page.

Updater smoke must cover optional, rollout-ineligible, critical grace, mandatory-after-deadline, offline fail-open, interrupted download, signature rejection, and active-work restart blocking.

---

## Gate 6 — Documentation

- [ ] [STATUS.md](STATUS.md) — current tag and stable/beta intent *(update at cut time)*
- [ ] [download.html](download.html) — asset links match tag *(update at cut time)*
- [x] [CHANGELOG.md](CHANGELOG.md) + [release-notes.html](release-notes.html) — stable section (ad-hoc macOS, v1.0.1 notarization note)
- [x] [STABLE_SCOPE.md](STABLE_SCOPE.md) — accurate for ad-hoc macOS v1.0.0

---

## Definition of ready to cut (no tag until all checked)

- [x] Gate 1 green
- [x] Gate 2: `test-parity-stable-restart` PASS (2026-06-09, [log](testing/parity-stable-restart-2026-06-09-1723.log))
- [x] Gate 3: updater publish fix merged *(verify on tag at cut time)*
- [x] Gate 4: explicitly deferred; `macos-adhoc-sign` documented
- [ ] Gate 5: minimum platform smoke signed off by operator — run [testing/stable-platform-smoke.md](testing/stable-platform-smoke.md)
- [x] Gate 6: scope/changelog/release-notes prep merged *(download.html asset URLs at cut time)*
- [ ] CHANGELOG `[1.0.0]` section dated when you cut (remove TBD)

**Operator decision:** When Gate 5 and the CHANGELOG date are done, run cut procedure below. **No tag until then.**

---

## Cut stable release

When ready checklist passes:

```bash
./scripts/cut-stable-release.sh              # dry-run
./scripts/cut-stable-release.sh --execute    # tag v1.2.0 (or v1.3.0), push, then refresh site after CI
./scripts/update-website-release.sh v1.2.0 --bump-site v1.2.0-beta.N
./scripts/verify-updater-manifest.sh v1.2.0 stable
```

**Website bump:** `update-website-release.sh --bump-site` only rewrites `index.html`, `features/index.html`, `features/hardware-requirements.html`, and root `README.md`. It does **not** rewrite historical entries in `release-notes.html` (append-only).

**Beta / stable tag checklist:**

1. Add section to [CHANGELOG.md](CHANGELOG.md) and append [release-notes.html](release-notes.html)
2. `./scripts/update-website-release.sh vX.Y.Z --bump-site vPREV`
3. `python3 scripts/sync-site-nav.py`
4. Verify `./scripts/extract-changelog-section.sh vX.Y.Z` matches the new CHANGELOG section (used by release CI for GitHub release body)

**Linux:** CI publishes `.deb` for stable; AppImage is not on the download page.

**Beta users:** stay on beta channel until they install a stable build manually. See [RELEASE_UPDATES.md](RELEASE_UPDATES.md).

---

## Soak policy

With few or no public users, soak is **operator validation**, not a calendar wait:

1. **Feature freeze** — bugfixes and docs only on `main` until stable cut
2. Re-run Gates 2–3 before tagging
3. **v1.0.0** does not require Gate 4 (notarization) — Gate 5 minimum smoke required
4. Gate 4 required for **v1.0.1** notarization patch only

---

## Post-stable — v1.0.1 notarization (when Apple account available)

1. Add Apple GitHub secrets per [DEVELOPMENT_NOTES.md](DEVELOPMENT_NOTES.md#macos-notarization-release-ci)
2. Tag `v1.0.1` (or beta) and verify notarized `.dmg` on clean Mac
3. Remove `macos-adhoc-sign` limitation; update [DOWNLOAD.md](DOWNLOAD.md)
