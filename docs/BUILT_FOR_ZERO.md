# Built for $0

Neural Junkie is an experiment: ship a credible multi-agent desktop product using **only free and open-source tools** for build, distribution, and marketing.

**Marketing site:** [built-for-zero.html](https://camronwood.github.io/neural-junkie/built-for-zero.html)

## Thesis

| Layer | Free path |
|-------|-----------|
| **Users** | MIT license, local Ollama, no mandatory cloud |
| **Build** | Tauri, React, Go, GitHub Actions |
| **Ship** | GitHub Releases, Homebrew tap, GitHub Pages |
| **Market** | Static site, GoatCounter, articles, benchmarks |

## Honest platform taxes

| Gate | Paid fix | Free path |
|------|----------|-----------|
| macOS Gatekeeper | Apple notarization ($99/yr) | Homebrew, Right-click → Open, `scripts/open-macos-app.sh` |
| Windows SmartScreen | EV code signing | More info → Run anyway, SHA256 verify, source build |
| Custom domain | Registrar | `camronwood.github.io/neural-junkie` |

See [INSTALL_TRUST.md](INSTALL_TRUST.md) and [install-trust.html](https://camronwood.github.io/neural-junkie/install-trust.html).

## Reproduce

```bash
git clone https://github.com/camronwood/neural-junkie.git
cd neural-junkie
make gui-install && make gui
make site-seo-sync   # regenerate marketing SEO
```
