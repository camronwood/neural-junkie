# Download and first run (v1.0.0-beta.8)

Install Neural Junkie from [GitHub Releases](https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.8) — no Go, Node, or Rust required for the desktop app.

## 1. Install

| Platform | Artifact |
|----------|----------|
| macOS (Apple Silicon) | `.dmg` with `aarch64` in the name |
| macOS (Intel) | `.dmg` with `x64` or `x86_64` in the name |
| Windows | `.msi` or setup `.exe` |
| Linux | `.AppImage` and/or `.deb` |

**macOS:** If Gatekeeper blocks the app (unsigned build), right-click → **Open**, or allow it in **System Settings → Privacy & Security**.

**Windows:** Install [Ollama](https://ollama.com) yourself, or choose a cloud provider in the wizard. The app cannot auto-install Ollama on Windows.

## 2. First launch

1. Open **Neural Junkie**.
2. Complete the **setup wizard**:
   - **macOS / Linux:** choose **Ollama (local)** — the wizard can install or start Ollama and pull a default model.
   - **Windows:** install Ollama from [ollama.com](https://ollama.com) first, or pick **cloud** and enter an API key.
3. Sign in on the login screen (pick a username and channel — local dev defaults are fine).

The bundled hub listens on **`http://localhost:18765`** (started by the desktop app).

## 3. Five-minute first win

1. In chat, ask **Moderator**:
   ```
   @Moderator What can Neural Junkie do?
   ```
2. Open the **command palette** (toolbar or type `/`) and run `/help`.
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
| Windows + Ollama | Install Ollama manually; ensure `ollama` is on your PATH |

**Build from source instead?** See [GETTING_STARTED.md](GETTING_STARTED.md).

**Issues:** [github.com/camronwood/neural-junkie/issues](https://github.com/camronwood/neural-junkie/issues)
