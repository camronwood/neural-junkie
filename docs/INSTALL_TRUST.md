# Install trust (no paid certs)

Install Neural Junkie without Apple notarization or Windows EV signing.

**Marketing site:** [install-trust.html](https://camronwood.github.io/neural-junkie/install-trust.html)

## Quick picks

| Platform | Recommended |
|----------|-------------|
| macOS | `brew tap camronwood/tap && brew install --cask neural-junkie` |
| macOS (.dmg) | Right-click app → **Open** (first launch) |
| Windows | SmartScreen → **More info** → **Run anyway** |
| Linux | `sudo dpkg -i neural-junkie_*.deb` |
| Any | `./scripts/verify-release-checksums.sh <tag> <file>` or build from source |

## macOS quarantine helper

```bash
./scripts/open-macos-app.sh "/Applications/Neural Junkie.app"
```

## Verify SHA256

Releases publish `SHA256SUMS` (see `scripts/publish-release-checksums.sh`).

```bash
./scripts/verify-release-checksums.sh v1.2.0-beta.5 ~/Downloads/Neural.Junkie_1.2.0-beta.5_aarch64.dmg
```

## Build from source

```bash
make gui-install
make gui-build
make gui
```

## Community packaging

- **Scoop:** `packaging/scoop/neural-junkie.json` — volunteer PR to [ScoopInstaller/Extras](https://github.com/ScoopInstaller/Extras)
- **WinGet:** submit manifest when stable (no cert required for community bucket)

See [BUILT_FOR_ZERO.md](BUILT_FOR_ZERO.md) for why we skip paid signing.
