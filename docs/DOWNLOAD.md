# Download and first run (v1.0.0-beta.32)

Install Neural Junkie from [GitHub Releases](https://github.com/camronwood/neural-junkie/releases) — no Go, Node, or Rust required for the desktop app.

**Auto-update:** After you install a release that includes in-app updates, Neural Junkie checks for newer builds on your channel (beta or stable) and can install them with one click. See [RELEASE_UPDATES.md](RELEASE_UPDATES.md). If you are on an older installer from before auto-update shipped, install one current release manually first.

## 1. Install

| Platform | Artifact |
|----------|----------|
| macOS (Apple Silicon) | `.dmg` with `aarch64` in the name |
| macOS (Intel) | `.dmg` with `x64` or `x86_64` in the name |
| Windows | `.msi` or setup `.exe` |
| Linux | `.deb` (no AppImage yet — see [known-issues.html](known-issues.html#linux-appimage-ci)) |

**macOS:** Builds are ad-hoc signed but not notarized. If Gatekeeper blocks the app, right-click → **Open**, or allow it in **System Settings → Privacy & Security**.

**Ollama:** **macOS** installers bundle the Ollama runtime. **Windows** and **Linux** use slim installers — the setup wizard **auto-installs Ollama** on first launch (internet required; Linux may prompt for password). All platforms need a **one-time model pull** on first run (internet required once). Cloud APIs remain optional in the wizard.

## 2. First launch

1. Open **Neural Junkie**.
2. Complete the **setup wizard**:
   - Pick a focus track or skip — install **domain packs** later from Settings (Software development, Life sciences, CAD, Specialist tuning). See [BIOLOGY_PACK.md](BIOLOGY_PACK.md) for life sciences.
   - Choose **Ollama (local)** — macOS starts bundled Ollama automatically; Linux/Windows use wizard **Install Ollama**; pull the suggested default model when prompted.
   - Or pick **cloud** and enter an API key (no local model pull).
3. Sign in on the login screen (pick a username and channel — local dev defaults are fine).

The bundled hub listens on **`http://localhost:18765`** (started by the desktop app).

## 3. Five-minute first win

1. In chat, ask **Moderator**:
   ```
   @Moderator What can Neural Junkie do?
   ```
2. Open the **command palette** with **Cmd+Shift+P** on macOS or **Ctrl+Shift+P** on Linux/Windows, then run **Help**.
3. Optional — index a repo and ask an expert:
   ```
   /create-repo-agent /path/to/your/repo MyRepoExpert
   @MyRepoExpert summarize the architecture and top risk areas
   ```

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
