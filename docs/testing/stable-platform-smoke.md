# Stable platform install smoke

Operator checklist for **Gate 5** in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md).

Use installers from [GitHub Releases](https://github.com/camronwood/neural-junkie/releases) (currently **v1.0.0-beta.33** until stable cut). Re-smoke on the tag you eventually ship.

**Minimum before v1.0.0 cut:** macOS arm64 (ad-hoc) on your machine + **one of** Windows x64 or Linux x64.

---

## macOS arm64 (ad-hoc)

1. Download `Neural.Junkie_1.0.0-beta.33_aarch64.dmg` (or current tag).
2. Open `.dmg`, drag app to Applications.
3. First launch: if Gatekeeper blocks, **Right-click → Open** (expected for ad-hoc builds).
4. Complete setup wizard — bundled Ollama should start; pull default model if prompted.
5. DM smoke: message **Assistant** in DM — confirm a reply within ~2 min.
6. Optional: Settings → About → **Check for updates** (may fail until next tag with fixed updater manifests).

Record in checklist Gate 5 matrix: **PASS** / **FAIL** + notes.

---

## Windows x64

1. Download `.msi` or setup `.exe` from Releases.
2. Install; launch app.
3. Wizard should offer **Install Ollama** (internet required).
4. DM smoke: message **Assistant** — confirm reply.

---

## Linux x64

1. Download `.deb` from Releases.
2. `sudo dpkg -i neural-junkie_*.deb` (install deps if needed).
3. Launch from app menu or `neural-junkie`.
4. Wizard **Install Ollama** if not on PATH.
5. DM smoke: message **Assistant** — confirm reply.

---

## Sign-off

When done, update the Gate 5 table in [STABLE_RELEASE_CHECKLIST.md](../STABLE_RELEASE_CHECKLIST.md) and check the Gate 5 item in **Definition of ready to cut**.
