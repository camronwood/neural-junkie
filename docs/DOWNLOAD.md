# Download and first run (v1.2.0-beta.6)

Install Neural Junkie from [GitHub Releases — v1.2.0-beta.6](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.6) ([latest](https://github.com/camronwood/neural-junkie/releases/latest)) — no Go, Node, or Rust required for the desktop app.

**Auto-update:** After you install a release that includes in-app updates, Neural Junkie checks for newer builds on your channel (beta or stable) and can install them with one click. See [RELEASE_UPDATES.md](RELEASE_UPDATES.md). If you are on an older installer from before auto-update shipped, install one current release manually first.

## 1. Install

**macOS (Homebrew):**

```bash
brew tap camronwood/tap
brew install --cask neural-junkie
```

Details: [HOMEBREW.md](HOMEBREW.md). The cask installs the same signed `.dmg` as GitHub Releases.

| Platform | Artifact |
|----------|----------|
| macOS (Apple Silicon) | `.dmg` with `aarch64` in the name |
| macOS (Intel) | `.dmg` with `x64` or `x86_64` in the name |
| Windows | `.msi` or setup `.exe` |
| Linux | `.deb` (x86_64) |

**macOS:** Official GitHub Release builds are **ad-hoc signed** at v1.0.0 until Apple Developer credentials are available (**v1.0.1** targets notarization). If Gatekeeper blocks first launch, right-click → **Open**. Local builds from source use ad-hoc signing as well.

**Linux:** Stable releases ship **`.deb`** (x86_64). AppImage is not published on the download page (CI may build it best-effort on beta tags only).

**Ollama:** **macOS** installers bundle the Ollama runtime. **Windows** and **Linux** use slim installers — the setup wizard **auto-installs Ollama** on first launch (internet required; Linux may prompt for password). All platforms need a **one-time model pull** on first run (internet required once). Cloud APIs remain optional in the wizard.

## 2. First launch

1. Open **Neural Junkie**.
2. Complete the **setup wizard**:
   - Pick a focus track or skip — install **domain packs** later from Settings (Software development, Life sciences, CAD, Specialist tuning). See [BIOLOGY_PACK.md](BIOLOGY_PACK.md) for life sciences.
   - Choose **Ollama (local)** — macOS starts bundled Ollama automatically; Linux/Windows use wizard **Install Ollama**; pull the suggested default model when prompted.
   - Or pick **cloud** and enter an API key (no local model pull).
3. Sign in on the login screen (pick a username and channel — local dev defaults are fine).

The bundled hub listens on **`http://localhost:18765`** (started by the desktop app).

## 3. Next steps

For a guided five-minute path (first chat, command palette, repo expert, collaboration), see the marketing site [Start here guide](https://camronwood.github.io/neural-junkie/start-here.html) or [GETTING_STARTED.md](GETTING_STARTED.md) for source developers.

## Troubleshooting

| Issue | What to try |
|-------|-------------|
| App won’t open (macOS) | Right-click → **Open**; check **Privacy & Security** |
| No AI responses | Settings → **AI Providers** — confirm Ollama is running or cloud key is set |
| Hub unreachable | Quit and relaunch the app; check nothing else is using port **18765** |
| Model pull fails | Check internet; retry from setup wizard or **Settings → Ollama** |
| Large download size | Installers include Ollama GPU runtimes (~1–2 GB per platform) |

**Build from source instead?** See [GETTING_STARTED.md](GETTING_STARTED.md).

**Issues:** [github.com/camronwood/neural-junkie/issues](https://github.com/camronwood/neural-junkie/issues)
