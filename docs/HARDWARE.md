# Hardware requirements

**Last updated:** June 2026 · **Current beta:** [v1.0.0-beta.25](https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.25)

Neural Junkie is lightweight as an **app**; the hardware story is about **models**, **concurrency**, and what you expect local agents to do. This doc is the authoritative reference for disk, RAM, and tiered model choices. The setup wizard and **Settings → AI Providers** use the same tiers via `GET /api/system/hardware`.

---

## App footprint (small)

| Component | Typical size |
|-----------|----------------|
| Desktop installer (Tauri + hub sidecar) | ~10–15 MB app bundle |
| Bundled Ollama **runtime** (beta.22+) | ~1–2 GB per platform |
| Hub process | Modest CPU/RAM; listens on `http://localhost:18765` |
| Session / config under `~/.neural-junkie/` | Grows with history; bounded per channel |

Installers bundle the Ollama **runtime only** — not every model. First model pull is a one-time download (often several GB). See [DOWNLOAD.md](DOWNLOAD.md).

**From source:** Go 1.23+, Node 18+, Rust (desktop only). Downloaded builds need none of these.

---

## Default model footprint (local Ollama)

From [GETTING_STARTED.md](GETTING_STARTED.md) — software development track:

| Role | Default tag | Disk (approx) |
|------|-------------|---------------|
| Coding specialists | `qwen2.5-coder:14b` | ~9 GB |
| Moderator + Assistant | `qwen2.5:7b` | ~4.5 GB |

**Typical first-run disk:** installer + runtime + both models ≈ **15–20 GB**.

Lighter alternatives in the Model library: `qwen2.5-coder:7b` (~4.7 GB), `llama3.2:3b` (~2 GB). Life sciences and CAD tracks use different defaults — see [BIOLOGY_PACK.md](BIOLOGY_PACK.md) and [CAD_PACK.md](CAD_PACK.md).

---

## RAM tiers and recommended models

The hub classifies installed RAM into four tiers (`GET /api/system/hardware`). Boundaries use whole GB (floor from total bytes).

```mermaid
flowchart TB
  subgraph tiers [RAM tiers]
    Minimal["minimal: under 8 GB"]
    Light["light: 8 to 15 GB"]
    Recommended["recommended: 16 to 31 GB"]
    Heavy["heavy: 32 GB and up"]
  end
  Minimal --> Light --> Recommended --> Heavy
```

| Tier | RAM | Developer / CAD primary | Utility | Good for |
|------|-----|-------------------------|---------|----------|
| **minimal** | &lt; 8 GB | `llama3.2:3b` or `qwen2.5-coder:7b` | `qwen2.5:7b` | Light chat; collab/repo work may struggle — prefer cloud hybrid |
| **light** | 8–15 GB | `qwen2.5-coder:7b` | `qwen2.5:7b` | Safe local dev; skip 14B default |
| **recommended** | 16–31 GB | `qwen2.5-coder:14b` | `qwen2.5:7b` | Full software pack, repo agents, collaboration |
| **heavy** | 32 GB+ | `qwen2.5-coder:14b` (+ optional LoRA bases, 70B experiments) | `qwen2.5:7b` | Multi-model library, LoRA training with CUDA |

**Life sciences:** `koesn/llama3-openbiollm-8b:latest` + `qwen2.5:7b` for tools on most tiers; under 8 GB, prefer cloud Hugging Face in the wizard.

**Team chat / productivity:** `qwen2.5:7b` only — lowest local footprint.

### Estimated RAM for a model tag

For catalog models, the app derives:

```
estimated_ram_gb = ceil(model_disk_gb × 1.25 + 4)
```

The **4 GB** headroom covers OS, hub, and Ollama runtime. Example: `qwen2.5-coder:14b` (~9 GB disk) → **16 GB** suggested RAM. Lookup: `GET /api/ollama/library/lookup?name=qwen2.5-coder:14b`.

---

## Multi-agent ≠ multi-model

Specialists **share one Ollama backend**. You are not loading six separate 14B weights for six agents. You **do** pay for:

- **Sequential inference** when several agents reply in a thread or collaboration
- **Hub memory** for message history, repo indexes, session state
- **Extra disk** if you add LoRA bases (`llama3.1:8b`, `mistral:7b`, …) on top of Qwen

LoRA adapters (~tens of MB each) reduce “five full 14B downloads” — see [LORA_ADAPTERS.md](LORA_ADAPTERS.md).

---

## Optional workloads

| Feature | Extra requirements |
|---------|-------------------|
| **LoRA training** | Specialist tuning pack; Python `.venv-lora` (`make deps-lora`); **CUDA strongly recommended** — see [LORA_TRAINING.md](LORA_TRAINING.md) |
| **Repo agents** | Index capped at **2000** source file bodies in memory; large monorepos are trimmed |
| **Personal learning** | Ollama embeddings; no CUDA required — [PERSONAL_LEARNING.md](PERSONAL_LEARNING.md) |
| **Collaboration** | Local 7B/14B variance in plan quality — [KNOWN_ISSUES.md](KNOWN_ISSUES.md) (`collab-model-variance`) |
| **Slack bridge** | Hub must stay running locally (Socket Mode) |

---

## Hybrid strategy

- **Local (Ollama, LM Studio):** broad iteration, repo Q&A, specialist chat — no per-token bill
- **Cloud (Claude, OpenAI-compatible APIs):** hard reviews, long-context passes, when local models are not enough
- **Smart routing (optional):** execution tasks only — Settings → AI Providers

See [USER_VALUE_GUIDE.md](USER_VALUE_GUIDE.md).

---

## Platform matrix

| Platform | Installer | Notes |
|----------|-----------|-------|
| macOS Apple Silicon | `.dmg` (`aarch64`) | Ad-hoc signed, not notarized — Right-click → Open |
| macOS Intel | `.dmg` (`x64`) | Same Gatekeeper note |
| Windows | `.msi` / `.exe` | Bundled Ollama since beta.22 |
| Linux | `.AppImage` / `.deb` | Bundled Ollama since beta.22 |

Full install table: [DOWNLOAD.md](DOWNLOAD.md).

---

## Honest limits

Public tracker: [KNOWN_ISSUES.md](KNOWN_ISSUES.md) · [known-issues.html](known-issues.html)

| ID | Summary |
|----|---------|
| `ollama-model-pull` | Runtime bundled; models pulled on first use (several GB) |
| `collab-model-variance` | Local models vary in collaboration quality and timeouts |
| `lmstudio-tools` | MCP tool calling strongest on Ollama and Claude |
| `single-hub` | Single-server — no horizontal scale |
| `macos-notarized` | Gatekeeper may block first launch |
| `web-ui-thin` | Browser hub UI is chat-only — use desktop for full workspace |

---

## API reference

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/system/hardware` | Total RAM, tier, per-track model recommendations |
| `GET` | `/api/ollama/library/lookup?name=…` | Catalog `size_hint`, estimated disk/RAM |

Implementation: [`internal/hardware/`](../internal/hardware/).

---

## Related

- [GETTING_STARTED.md](GETTING_STARTED.md) — pull models, provider setup
- [DOWNLOAD.md](DOWNLOAD.md) — installers and troubleshooting
- [LORA_ADAPTERS.md](LORA_ADAPTERS.md) — disk-efficient specialists
- [marketing/HARDWARE-LINKEDIN.md](marketing/HARDWARE-LINKEDIN.md) — LinkedIn article publish copy
