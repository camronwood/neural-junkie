#!/usr/bin/env bash
#
# Refresh docs/download.html with direct GitHub release asset links.
# Optionally bump version strings across the marketing site.
#
# Usage:
#   ./scripts/update-website-release.sh v1.0.0-beta.27
#   ./scripts/update-website-release.sh v1.0.0-beta.27 --bump-site v1.0.0-beta.25
#
# Requires: gh, jq

set -euo pipefail

TAG="${1:?Usage: $0 <tag> [--bump-site <old-tag>]}"
REPO="${WEBSITE_REPO:-camronwood/neural-junkie}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUMP_FROM=""

shift || true
while [[ $# -gt 0 ]]; do
  case "$1" in
    --bump-site)
      BUMP_FROM="${2:?--bump-site requires a tag}"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

DISPLAY_VERSION="${TAG#v}"
RELEASE_BASE="https://github.com/${REPO}/releases/download/${TAG}"
RELEASE_PAGE="https://github.com/${REPO}/releases/tag/${TAG}"
PUB_DATE="$(gh release view "${TAG}" --repo "${REPO}" --json publishedAt,createdAt -q '.publishedAt // .createdAt' 2>/dev/null || true)"
PUB_DATE="${PUB_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
HUMAN_DATE="$(date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "${PUB_DATE}" "+%B %-d, %Y" 2>/dev/null || date -u -d "${PUB_DATE}" "+%B %d, %Y" 2>/dev/null || date -u +"%B %d, %Y")"

ASSETS=()
while IFS= read -r line; do
  [[ -n "${line}" ]] && ASSETS+=("${line}")
done < <(gh release view "${TAG}" --repo "${REPO}" --json assets -q '.assets[].name' | sort)

pick_asset() {
  local pattern="$1"
  local name
  for name in "${ASSETS[@]}"; do
    if [[ "${name}" =~ ${pattern} ]]; then
      echo "${name}"
      return 0
    fi
  done
  return 1
}

link_or_pending() {
  local pattern="$1"
  local label="$2"
  local asset
  if asset="$(pick_asset "${pattern}")"; then
    printf '            <a class="btn btn-primary download-card__btn" href="%s/%s" download>%s</a>\n' "${RELEASE_BASE}" "${asset}" "${label}"
  else
    printf '            <p class="download-card__pending">Building — check <a href="%s">GitHub Releases</a> shortly.</p>\n' "${RELEASE_PAGE}"
  fi
}

MAC_ARM="$(pick_asset 'aarch64\.dmg$' || true)"
MAC_INTEL="$(pick_asset '(^|_)x64\.dmg$' || true)"
WIN_MSI="$(pick_asset '\.msi$' || true)"
WIN_EXE="$(pick_asset '\.exe$' || true)"
LINUX_APPIMAGE="$(pick_asset '\.AppImage$' || true)"
LINUX_DEB="$(pick_asset '\.deb$' || true)"

cat > "${ROOT}/docs/download.html" <<HTML
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Download ${DISPLAY_VERSION} — Neural Junkie</title>
  <meta name="description" content="Download Neural Junkie ${DISPLAY_VERSION} for macOS, Windows, and Linux — direct installer links from GitHub Releases." />
  <link rel="icon" type="image/png" sizes="32x32" href="assets/icon/favicon-32.png" />
  <link rel="apple-touch-icon" href="assets/icon/apple-touch-icon.png" />
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=DM+Sans:ital,opsz,wght@0,9..40,400;0,9..40,600;0,9..40,800;1,9..40,400&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet" />
  <link rel="stylesheet" href="css/landing.css" />
</head>
<body>
  <a class="skip-link" href="#main">Skip to content</a>
  <header class="site-header">
    <div class="wrap">
      <a class="logo" href="index.html" aria-label="Neural Junkie home">
        <span class="logo-mark" aria-hidden="true"><img src="assets/icon/favicon-32.png" alt="" width="32" height="32" decoding="async" /></span>
        Neural Junkie
      </a>
      <nav class="nav-actions" aria-label="Primary">
        <a class="btn btn-ghost" href="index.html">Landing</a>
        <a class="btn btn-ghost" href="features/index.html">Feature guides</a>
        <a class="btn btn-ghost" href="release-notes.html">Release notes</a>
        <a class="btn btn-primary" href="download.html">Download ${DISPLAY_VERSION}</a>
      </nav>
    </div>
  </header>

  <main id="main" class="wrap feature-deep" style="padding-top: clamp(2rem, 5vw, 3rem);">
    <nav class="breadcrumb" aria-label="Breadcrumb">
      <a href="index.html">Home</a> · Download
    </nav>
    <header class="feature-deep-hero">
      <h1>Download Neural Junkie</h1>
      <p class="feature-deep-lead">
        Latest beta: <strong>${DISPLAY_VERSION}</strong>
        <time datetime="${PUB_DATE}">${HUMAN_DATE}</time>.
        Pick your platform below for a direct installer link, or browse
        <a href="${RELEASE_PAGE}">all assets on GitHub</a>.
        Installers bundle Ollama — first launch pulls a default model once (internet required).
        In-app auto-updates ship in this release and later betas.
      </p>
    </header>

    <div class="download-grid">
      <section class="download-card" aria-labelledby="dl-macos-arm">
        <h2 id="dl-macos-arm">macOS · Apple Silicon</h2>
        <p class="download-card__meta">M1 / M2 / M3 / M4 · <code>.dmg</code></p>
        <p class="download-card__file">${MAC_ARM:-Pending CI upload}</p>
$(link_or_pending 'aarch64\.dmg$' 'Download .dmg')
        <p class="download-card__note">First launch: right-click → <strong>Open</strong> if Gatekeeper blocks (ad-hoc signed).</p>
      </section>

      <section class="download-card" aria-labelledby="dl-macos-intel">
        <h2 id="dl-macos-intel">macOS · Intel</h2>
        <p class="download-card__meta">x64 Macs · <code>.dmg</code></p>
        <p class="download-card__file">${MAC_INTEL:-Pending CI upload}</p>
$(link_or_pending '(^|_)x64\.dmg$' 'Download .dmg')
        <p class="download-card__note">Same Gatekeeper note as Apple Silicon builds.</p>
      </section>

      <section class="download-card" aria-labelledby="dl-windows">
        <h2 id="dl-windows">Windows</h2>
        <p class="download-card__meta">x64 · <code>.msi</code> or setup <code>.exe</code></p>
        <p class="download-card__file">${WIN_MSI:-${WIN_EXE:-Pending CI upload}}</p>
$(link_or_pending '\.msi$' 'Download .msi')
$(if [[ -n "${WIN_EXE}" ]]; then printf '            <a class="btn btn-ghost download-card__btn-secondary" href="%s/%s" download>Download setup .exe</a>\n' "${RELEASE_BASE}" "${WIN_EXE}"; fi)
      </section>

      <section class="download-card" aria-labelledby="dl-linux">
        <h2 id="dl-linux">Linux</h2>
        <p class="download-card__meta">x86_64 · <code>.AppImage</code> or <code>.deb</code></p>
        <p class="download-card__file">${LINUX_APPIMAGE:-${LINUX_DEB:-Pending CI upload}}</p>
$(link_or_pending '\.AppImage$' 'Download AppImage')
$(if [[ -n "${LINUX_DEB}" ]]; then printf '            <a class="btn btn-ghost download-card__btn-secondary" href="%s/%s" download>Download .deb</a>\n' "${RELEASE_BASE}" "${LINUX_DEB}"; fi)
        <p class="download-card__note">AppImage: <code>chmod +x</code> then run. Deb: install with your package manager.</p>
      </section>
    </div>

    <article class="feature-prose" style="margin-top: 2rem;">
      <h2>After install</h2>
      <ol>
        <li>Open Neural Junkie and complete the setup wizard (Ollama local or cloud API).</li>
        <li>See <a href="https://github.com/${REPO}/blob/main/docs/DOWNLOAD.md">DOWNLOAD.md</a> for a five-minute first win.</li>
        <li>Updates: Settings → About → <strong>Check for updates</strong> (beta channel). See <a href="https://github.com/${REPO}/blob/main/docs/RELEASE_UPDATES.md">RELEASE_UPDATES.md</a>.</li>
      </ol>
      <p>
        Prefer source? <a href="https://github.com/${REPO}#quick-start">Build from the repo</a>.
        Problems? <a href="known-issues.html">Known issues</a> ·
        <a href="https://github.com/${REPO}/issues">GitHub Issues</a>
      </p>
    </article>
  </main>

  <footer class="site-footer">
    <div class="wrap">
      <p>
        <a href="https://github.com/${REPO}">Neural Junkie</a>
        — open source · ${DISPLAY_VERSION} open beta
      </p>
    </div>
  </footer>
</body>
</html>
HTML

echo "Wrote docs/download.html for ${TAG}"
echo "  macOS arm:  ${MAC_ARM:-pending}"
echo "  macOS x64:  ${MAC_INTEL:-pending}"
echo "  Windows:    ${WIN_MSI:-${WIN_EXE:-pending}}"
echo "  Linux:      ${LINUX_APPIMAGE:-${LINUX_DEB:-pending}}"

if [[ -n "${BUMP_FROM}" ]]; then
  OLD="${BUMP_FROM#v}"
  while IFS= read -r file; do
    [[ -z "${file}" ]] && continue
    [[ "${file}" == *download.html ]] && continue
    sed -i.bak \
      -e "s/${BUMP_FROM}/${TAG}/g" \
      -e "s/v${OLD}/${DISPLAY_VERSION}/g" \
      "${file}"
    rm -f "${file}.bak"
    echo "Bumped version in ${file#${ROOT}/}"
  done < <(rg -l "${BUMP_FROM}" "${ROOT}/docs" --glob '*.html' 2>/dev/null || true)
  if [[ -f "${ROOT}/README.md" ]]; then
    sed -i.bak "s/${BUMP_FROM}/${TAG}/g" "${ROOT}/README.md" && rm -f "${ROOT}/README.md.bak"
    echo "Bumped version in README.md"
  fi
fi
