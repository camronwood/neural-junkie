# Release auto-updates

Neural Junkie uses the [Tauri v2 updater](https://v2.tauri.app/plugin/updater/) with mandatory minisign verification. macOS and Windows check and download updates automatically; installation waits for a safe restart. Linux `.deb` updates remain manual until installed-package upgrade testing is complete.

## Channels

| Channel | Build tag | Manifest URL |
|---------|-----------|--------------|
| **Stable** | `v1.0.0`, `v1.1.0`, … | `https://github.com/camronwood/neural-junkie/releases/latest/download/update-{target}-{arch}.json` |
| **Beta** | `v1.0.0-beta.N` | `https://raw.githubusercontent.com/camronwood/neural-junkie/main/updater/beta/update-{target}-{arch}.json` |

Rolling beta manifests are git-backed under [`updater/beta/`](../updater/beta/). CI uploads immutable versioned manifests while the release is a draft, publishes the release, verifies success, and only then runs `promote-updater-channel.sh` to advance the beta pointer. A failed release therefore leaves clients on the previous beta.

Beta installers (beta.21+) point **only** at the git-backed URL. The legacy `updater-beta` GitHub release endpoint is retired (404) and must not be listed first — a failing primary endpoint previously made in-app checks look broken even when the fallback was healthy.

Beta builds only receive beta updates. Stable builds only receive stable updates.

The first Tauri v2 releases generate `v1Compatible` updater artifacts so installed Tauri v1 clients can migrate without a manual reinstall. Keep this mode for multiple release cycles.

## Client behavior

1. Launch continues while a ten-second update check runs.
2. Eligible signed updates download in the background.
3. The app shows **Restart to update** after verification.
4. Restart saves editor and runbook changes and refuses to interrupt drafts, active agents, collaborations, training/analysis jobs, or terminal foreground work.
5. Network, manifest, or download failures fail open.
6. Critical mandatory updates block normal use only after their configured grace deadline and only while a verified bundle is immediately installable.

After a signed bundle is verified, the client caches its accepted update metadata for offline continuity. Downloaded bytes remain in the running Tauri process only. If the process exits before installation, the next launch warns about the accepted update and downloads it again; cached metadata alone never blocks normal use.

## Release policy

Each updater manifest contains `policy.schema_version: 1`, explicit channel and platform availability, plus:

- `severity`: `normal` or `critical`
- `enforcement`: `optional` or `mandatory`
- `mandatory_after`: ISO-8601 grace deadline
- `minimum_supported_version`: oldest supported client
- `rollout.percentage`: deterministic client cohort from 0–100
- `rollout.seed`: sticky release cohort seed

Release automation reads these optional environment variables:

```bash
UPDATE_SEVERITY=critical \
UPDATE_ENFORCEMENT=mandatory \
UPDATE_MANDATORY_AFTER=2026-07-24T12:00:00Z \
UPDATE_MINIMUM_SUPPORTED_VERSION=1.2.0-beta.7 \
UPDATE_ROLLOUT_PERCENTAGE=25 \
./scripts/publish-updater-manifests.sh v1.2.0-beta.8 camronwood/neural-junkie
```

Normal releases default to optional enforcement and a 100% rollout. Increase rollout percentages monotonically. Mandatory releases override cohort rollout after their deadline.

## Signing keys (one-time setup)

Generate a keypair locally:

```bash
cd desktop
npx tauri signer generate -w ~/.tauri/neural-junkie.key --force --ci
```

- **Public key** (`~/.tauri/neural-junkie.key.pub`) is committed in `desktop/src-tauri/tauri.conf.json` → `plugins.updater.pubkey`.
- **Private key** must never be committed. Add it as GitHub Actions secrets:

```bash
gh secret set TAURI_PRIVATE_KEY --repo camronwood/neural-junkie --body "$(cat ~/.tauri/neural-junkie.key)"
gh secret set TAURI_KEY_PASSWORD --repo camronwood/neural-junkie --body ""
```

If you regenerate keys, every installed app must be manually reinstalled once (old signatures will not match).

## CI flow

On each `v*` tag push:

1. `scripts/ci-prepare-release-build.sh` aligns all desktop versions and selects the updater channel.
2. Each platform build runs `tauri build` with `TAURI_SIGNING_PRIVATE_KEY` set, producing signed updater bundles:
   - macOS: `*.app.tar.gz` + `.sig`
   - Windows: `*.msi.zip` + `.sig`
   - Linux: `.deb` installer only; no automatic updater manifest yet
3. `scripts/publish-updater-manifests.sh` generates policy-bearing manifests and uploads them to the immutable release.
4. Checksums complete and CI publishes the release.
5. `scripts/promote-updater-channel.sh` atomically advances the beta pointer.

## Version alignment

Beta tags use **`1.0.0-beta.N`** in macOS/Linux bundles. **Windows** maps to WiX-safe semver **`1.0.0-N`** (numeric prerelease only). Updater manifests use the same version string as each platform's embedded bundle version.

## Verify a release

After CI completes:

```bash
# Beta channel manifests
./scripts/verify-updater-manifest.sh v1.0.0-beta.31 beta

# Stable channel (after v1.0.0)
./scripts/verify-updater-manifest.sh v1.0.0 stable
```

## Manual test checklist

1. Install a build **with** updater enabled at version N.
2. Publish version N+1 on the same channel.
3. Launch the app — it should download N+1 automatically.
4. Confirm **Restart to update** appears and active work prevents restart.
5. Restart and verify N+1, preserved settings, and a healthy Hub.
6. Repeat once from a Tauri v1 build into the first Tauri v2 build.

Users on installers from **before** auto-update was enabled must install one updater-enabled build manually from GitHub Releases.

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/ci-prepare-release-build.sh` | Version + channel prep for CI |
| `scripts/generate-update-manifests.sh` | Build platform manifest JSON from release assets |
| `scripts/publish-updater-manifests.sh` | Upload manifests to GitHub Releases |
| `scripts/promote-updater-channel.sh` | Advance beta pointer after release publication |
| `scripts/verify-desktop-version-consistency.sh` | Reject mismatched desktop versions |
| `scripts/bootstrap-beta-updater-channel.sh` | Sync `updater/beta/` manifests from a release tag |
| `scripts/verify-updater-artifacts.sh` | Fail CI if signed bundles are missing |
| `scripts/verify-updater-manifest.sh` | Validate published manifest URLs and signatures |

## Rollback

Do not advance a channel pointer when smoke tests fail. For beta, regenerate or copy the prior known-good versioned manifests into `updater/beta/` and push that change. For stable, mark the bad release non-latest and publish a higher-version hotfix; Tauri will not accept a lower version by default. Never rotate the updater key during rollback.
