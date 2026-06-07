# Release auto-updates

Neural Junkie uses the [Tauri v1 updater](https://v1.tauri.app/v1/guides/distribution/updater/) with minisign signatures. Beta and stable builds use separate update channels.

## Channels

| Channel | Build tag | Manifest URL |
|---------|-----------|--------------|
| **Stable** | `v1.0.0`, `v1.1.0`, … | `https://github.com/camronwood/neural-junkie/releases/latest/download/update-{target}-{arch}.json` |
| **Beta** | `v1.0.0-beta.N` | `https://github.com/camronwood/neural-junkie/releases/download/updater-beta/update-{target}-{arch}.json` |

Beta builds only receive beta updates. Stable builds only receive stable updates.

## Signing keys (one-time setup)

Generate a keypair locally:

```bash
cd desktop
npx tauri signer generate -w ~/.tauri/neural-junkie.key --force --ci
```

- **Public key** (`~/.tauri/neural-junkie.key.pub`) is committed in `desktop/src-tauri/tauri.conf.json` → `tauri.updater.pubkey`.
- **Private key** must never be committed. Add it as GitHub Actions secrets:

```bash
gh secret set TAURI_PRIVATE_KEY --repo camronwood/neural-junkie --body "$(cat ~/.tauri/neural-junkie.key)"
gh secret set TAURI_KEY_PASSWORD --repo camronwood/neural-junkie --body ""
```

If you regenerate keys, every installed app must be manually reinstalled once (old signatures will not match).

## CI flow

On each `v*` tag push:

1. `scripts/ci-prepare-release-build.sh` sets `package.version` from the tag and selects the updater channel.
2. Each platform build runs `tauri build` with `TAURI_PRIVATE_KEY` set, producing signed updater bundles:
   - macOS: `*.app.tar.gz` + `.sig`
   - Linux: `*.AppImage.tar.gz` + `.sig`
   - Windows: `*.msi.zip` + `.sig`
3. `scripts/publish-updater-manifests.sh` generates `update-*.json` manifests and uploads them to the versioned release (and to `updater-beta` for beta tags).

## Version alignment

Beta tags use **`1.0.0-beta.N`** in macOS/Linux bundles. **Windows** maps to WiX-safe semver **`1.0.0-N`** (numeric prerelease only). Updater manifests use the same version string as each platform's embedded bundle version.

## Verify a release

After CI completes:

```bash
# Beta channel manifests
./scripts/verify-updater-manifest.sh v1.0.0-beta.25 beta

# Stable channel (after v1.0.0)
./scripts/verify-updater-manifest.sh v1.0.0 stable
```

## Manual test checklist

1. Install a build **with** updater enabled at version N.
2. Publish version N+1 on the same channel.
3. Launch the app — update banner should appear within one launch (or use Settings → About → **Check for updates**).
4. Click **Update Now** — download progress, then relaunch into N+1.

Users on installers from **before** auto-update was enabled must install one updater-enabled build manually from GitHub Releases.

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/ci-prepare-release-build.sh` | Version + channel prep for CI |
| `scripts/generate-update-manifests.sh` | Build platform manifest JSON from release assets |
| `scripts/publish-updater-manifests.sh` | Upload manifests to GitHub Releases |
| `scripts/verify-updater-artifacts.sh` | Fail CI if signed bundles are missing |
| `scripts/verify-updater-manifest.sh` | Validate published manifest URLs and signatures |
