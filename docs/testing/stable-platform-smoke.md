# Stable platform install smoke

Operator checklist for **Gate 5** in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md).

Use installers from [GitHub Releases — v1.2.0-beta.7](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.7) (current soak build). Re-smoke on the tag you eventually ship as stable.

**Minimum before stable cut:** macOS arm64 (ad-hoc) on your machine + **one of** Windows x64 or Linux x64.

---

## macOS arm64 (ad-hoc)

1. Download [`Neural.Junkie_1.2.0-beta.7_aarch64.dmg`](https://github.com/camronwood/neural-junkie/releases/download/v1.2.0-beta.7/Neural.Junkie_1.2.0-beta.7_aarch64.dmg).
2. Quit any running Neural Junkie (`make stop` if a repo hub is up — packaged app should not fight `make start-all`).
3. Open `.dmg`, drag app to Applications (replace existing).
4. First launch: if Gatekeeper blocks, **Right-click → Open** (expected for ad-hoc builds).
5. Confirm **About** shows **1.2.0-beta.7**.
6. Complete setup wizard — bundled Ollama should start; pull default model if prompted.
7. DM smoke: message **Assistant** in DM — confirm a reply within ~2 min.
8. Optional soak of beta.7 hotfixes: open a workspace → Knowledge Graph → Ask agents on a node (no `workspace root not set`); hide main chat with IDE layout on (editor stays); confirm hub chip stays connected while typing.
9. Optional: Settings → About → **Check for updates** (beta channel should report current or no newer).

Record in checklist Gate 5 matrix: **PASS** / **FAIL** + notes.

---

## Windows x64

1. Download [`.msi`](https://github.com/camronwood/neural-junkie/releases/download/v1.2.0-beta.7/Neural.Junkie_1.2.0.7_x64_en-US.msi) or [setup `.exe`](https://github.com/camronwood/neural-junkie/releases/download/v1.2.0-beta.7/Neural.Junkie_1.2.0-7_x64-setup.exe).
2. Install; launch app.
3. Wizard should offer **Install Ollama** (internet required).
4. Confirm version **1.2.0-beta.7** (Windows WiX shows **1.2.0.7**).
5. DM smoke: message **Assistant** — confirm reply.

---

## Linux x64

1. Download [`neural-junkie_1.2.0-beta.7_amd64.deb`](https://github.com/camronwood/neural-junkie/releases/download/v1.2.0-beta.7/neural-junkie_1.2.0-beta.7_amd64.deb).
2. `sudo dpkg -i neural-junkie_1.2.0-beta.7_amd64.deb` (install deps if needed).
3. Launch from app menu or `neural-junkie`.
4. Wizard **Install Ollama** if not on PATH.
5. DM smoke: message **Assistant** — confirm reply.

**Homebrew (optional):** `brew upgrade --cask neural-junkie` (macOS) or `brew upgrade neural-junkie` (Linux) after `brew tap camronwood/tap`.

---

## Sign-off

When done, update the Gate 5 table in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md) and check the Gate 5 item in **Definition of ready to cut**. Close [#18](https://github.com/camronwood/neural-junkie/issues/18).
