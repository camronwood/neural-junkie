# Homebrew distribution

Neural Junkie ships on macOS via a **custom Homebrew tap** today. **Official `homebrew-cask`** is the planned next step once the app is stable and meets Homebrew notability guidelines.

## Install (macOS, custom tap)

```bash
brew tap camronwood/tap
brew install --cask neural-junkie
```

Upgrade:

```bash
brew upgrade --cask neural-junkie
```

Uninstall (including `~/.neural-junkie`):

```bash
brew uninstall --cask --zap neural-junkie
```

The cask installs the same signed `.dmg` artifacts from [GitHub Releases](https://github.com/camronwood/neural-junkie/releases). Ollama is bundled on macOS; first launch runs the setup wizard and a one-time model pull.

**Tap repo:** [github.com/camronwood/homebrew-tap](https://github.com/camronwood/homebrew-tap)

---

## Maintainer setup (one time)

1. **Create the tap repo** on GitHub: `camronwood/homebrew-tap` (local scaffold lives at `../homebrew-tap` next to this repo).
2. Push the initial contents (`Casks/neural-junkie.rb`, `README.md`, `.github/workflows/audit.yml`).
3. **Configure the deploy key** (recommended — see [Secrets](#secrets) below).
4. Publish a release — [`.github/workflows/bump-homebrew-tap.yml`](../.github/workflows/bump-homebrew-tap.yml) regenerates the cask and pushes to the tap.

### Secrets

Public open-source repos **never store tokens in git**. All credentials are **GitHub Actions secrets** (Settings → Secrets and variables → Actions on `camronwood/neural-junkie`). Forks and pull requests from outsiders **do not receive** those secrets.

#### Homebrew tap bump (recommended: deploy key)

Scope is limited to **one repo** (`homebrew-tap`) — not your Apple certs, Slack OAuth, or personal account.

1. On your machine, generate an SSH key pair used only for this automation:

   ```bash
   ssh-keygen -t ed25519 -f ./homebrew-tap-deploy -N "" -C "neural-junkie-release-bump"
   ```

2. On **camronwood/homebrew-tap** → Settings → Deploy keys → **Add deploy key**
   - Title: `neural-junkie-release-bump`
   - Key: contents of `homebrew-tap-deploy.pub`
   - **Allow write access** ✓

3. On **camronwood/neural-junkie** → Settings → Secrets → Actions → **New repository secret**
   - Name: `HOMEBREW_TAP_DEPLOY_KEY`
   - Value: full contents of `homebrew-tap-deploy` (private key, including `BEGIN`/`END` lines)

4. Delete the local key files after copying, or store the private key in a password manager.

The release workflow checks out `homebrew-tap` over SSH and pushes the updated cask. If the secret is missing, the job **warns and skips** — releases still publish; bump manually with `make bump-homebrew-cask`.

#### Alternative: fine-grained PAT

If you prefer a PAT over a deploy key:

- Create a fine-grained token scoped **only** to `camronwood/homebrew-tap`
- Permission: **Contents** read and write
- Set expiration (e.g. 90 days) and rotate
- Store as `HOMEBREW_TAP_TOKEN` and change the workflow checkout to use `token:` instead of `ssh-key:`

Deploy keys are preferable: they cannot access other repositories or your GitHub account.

#### Other release secrets (unchanged)

Signed installers and bundled Slack/Google OAuth use separate CI secrets (`APPLE_*`, `SLACK_VENDOR_*`, `GOOGLE_VENDOR_*`, `TAURI_*`). Those are injected at **build time** into release artifacts — they are not in the public source tree. Maintainers document names in `docs/DEVELOPMENT_NOTES.md`; values stay in GitHub Secrets.

#### Who can trigger secret use?

| Event | Gets secrets? |
|-------|----------------|
| Tag release on `camronwood/neural-junkie` | Yes (maintainer-controlled) |
| PR from a fork | No |
| PR from a branch in the main repo | Yes, but use `pull_request` guards on sensitive workflows |
| Random user forks + runs Actions | No secrets on fork (default) |

Only maintainers with release permission can publish tags that run the bump workflow on the canonical repo.

### Manual cask bump

```bash
make bump-homebrew-cask TAG=v1.2.0-beta.5 TAP_DIR=../homebrew-tap
# or
./scripts/bump-homebrew-cask.sh v1.2.0-beta.5 ../homebrew-tap
```

Then commit and push `homebrew-tap`.

### Local validation

```bash
brew tap camronwood/tap /path/to/homebrew-tap
brew audit --cask camronwood/tap/neural-junkie
brew style --cask camronwood/tap/neural-junkie
# Optional full install smoke:
brew install --cask camronwood/tap/neural-junkie
```

`brew audit --cask --online --strict` currently fails on **beta prereleases** (expected). Re-enable strict online audit in the tap CI after the first **stable** non-prerelease tag.

---

## Phase 2 — official `homebrew-cask` (future)

Submit a PR to [Homebrew/homebrew-cask](https://github.com/Homebrew/homebrew-cask) so users can run:

```bash
brew install --cask neural-junkie
```

without tapping.

### Prerequisites

| Item | Status | Notes |
|------|--------|-------|
| Stable release (no `-beta`) | Pending | e.g. `v1.2.0` or `v1.0.0` |
| SPDX `LICENSE` at repo root + on GitHub | Pending | Required for acceptance |
| Notability (stars, forks, press) | Low today | Aim for Homebrew’s informal threshold (~30–75+ stars) |
| Signed macOS binaries | Done | Developer ID + notarization in release CI |
| `brew audit --cask --online --strict` | Blocked on beta | Add `livecheck` when stable; prerelease check will pass |
| Responsive maintainer | — | Watch PR reviews on homebrew-cask |

### Stable cask additions

When cutting stable, extend the cask (in tap first, then upstream):

```ruby
livecheck do
  url "https://github.com/camronwood/neural-junkie/releases/latest"
  strategy :github_latest
end
```

Re-enable in tap CI:

```yaml
brew audit --cask --online --strict Casks/*
```

### Upstream maintenance

After acceptance, bump with:

```bash
brew bump-cask-pr neural-junkie --version 1.2.0
```

Or keep the release workflow and open a PR to homebrew-cask mirroring the tap file.

### What not to upstream as a formula

Do **not** submit a `homebrew-core` **formula** that builds the full app from source (Go + Node + Rust/Tauri + Ollama + vendor OAuth). The desktop product is a **cask** only.

Optional later: separate formulae for `nj-remote` or `cmd/cli` if there is CLI-only demand.

---

## Related docs

- [DOWNLOAD.md](DOWNLOAD.md) — all install paths
- [RELEASE_UPDATES.md](RELEASE_UPDATES.md) — in-app updater vs Homebrew upgrades
- [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md) — stable cut gates
