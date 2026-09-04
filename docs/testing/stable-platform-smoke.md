# Stable platform install smoke

Operator checklist for **Gate 5** in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md).

Use installers from [GitHub Releases — v1.2.0-beta.27](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.27) (current soak build). Re-smoke on the tag you eventually ship as stable.

Record results in [platform-smoke-beta24.md](platform-smoke-beta24.md).

**Minimum before stable cut:** macOS arm64 (ad-hoc) on your machine + **one of** Windows x64 or Linux x64.

---

## macOS arm64 (ad-hoc)

1. Download [`Neural.Junkie_1.2.0-beta.27_aarch64.dmg`](https://github.com/camronwood/neural-junkie/releases/download/v1.2.0-beta.27/Neural.Junkie_1.2.0-beta.27_aarch64.dmg).
2. Quit any running Neural Junkie (`make stop` if a repo hub is up — packaged app should not fight `make start-all`).
3. Open `.dmg`, drag app to Applications (replace existing).
4. First launch: if Gatekeeper blocks, **Right-click → Open** (expected for ad-hoc builds).
5. Confirm **About** shows **1.2.0-beta.27**.
6. Complete setup wizard — bundled Ollama should start; pull default model if prompted.
7. DM smoke: message **Assistant** in DM — confirm a reply within ~2 min.
8. **P0–P4:** lazy panels (editor, Domain packs) load on demand; Settings → **Domain packs** deep-link works; collab gen-error banner visible when applicable; load-failure toasts on panel errors.
9. Launch N with N+1 available; confirm the check does not block startup and download begins automatically.
10. Edit a file and start an agent response; confirm **Restart to update** refuses to interrupt active work.
11. Save/finish work, restart, and confirm N+1 launches with settings preserved and the Hub healthy.
12. Repeat with network unavailable (launch must continue), an invalid signature (installation must fail), and a partial/interrupted download (N remains runnable).

Record in checklist Gate 5 matrix: **PASS** / **FAIL** + notes.

---

## Windows x64

1. Download [`.msi`](https://github.com/camronwood/neural-junkie/releases/download/v1.2.0-beta.27/Neural.Junkie_1.2.0-27_x64_en-US.msi) (WiX product version shows **1.2.0.27**).
2. Install; launch app.
3. Wizard should offer **Install Ollama** (internet required).
4. Confirm version **1.2.0-beta.27** (Windows WiX shows **1.2.0.27**).
5. DM smoke: message **Assistant** — confirm reply.
6. Validate N → N+1 using the MSI updater artifact, including automatic download and updater-driven process exit.
7. Validate one installed Tauri v1 build → the first Tauri v2 build while `v1Compatible` artifacts are enabled.

---

## Linux x64

1. Download [`Neural.Junkie_1.2.0-beta.27_amd64.deb`](https://github.com/camronwood/neural-junkie/releases/download/v1.2.0-beta.27/Neural.Junkie_1.2.0-beta.27_amd64.deb).
2. `sudo dpkg -i Neural.Junkie_1.2.0-beta.27_amd64.deb` (install deps if needed).
3. Launch from app menu or `neural-junkie`.
4. Wizard **Install Ollama** if not on PATH.
5. DM smoke: message **Assistant** — confirm reply.
6. Confirm Settings → About identifies Linux as a manual-update platform and does not attempt automatic installation.

## Policy cases

For one macOS or Windows candidate, publish manifests against a test channel and verify:

1. `rollout.percentage: 0` does not download for an optional update.
2. A critical update before `mandatory_after` remains deferrable.
3. The same update after the deadline blocks normal use only after its signed bundle is verified and ready; it still permits retry, diagnostics, and quit.
4. Offline launch during the grace period remains usable.
5. Verify a mandatory update, quit before installation, then relaunch offline: the app must warn that the update is required without blocking normal use.
6. Relaunch online after the previous case: the bundle downloads again and blocking begins only when it is ready to install.

**Homebrew (optional):** `brew upgrade --cask neural-junkie` (macOS) or `brew upgrade neural-junkie` (Linux) after `brew tap camronwood/tap`.

---

## Sign-off

When done, update the Gate 5 table in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md) and [platform-smoke-beta26.md](platform-smoke-beta26.md) (or a new matrix for this tag). Close [#18](https://github.com/camronwood/neural-junkie/issues/18) when macOS arm64 + one secondary platform pass.
