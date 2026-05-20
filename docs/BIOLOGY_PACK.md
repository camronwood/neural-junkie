# Life Sciences / Biology Pack (v1)

Neural Junkie includes a **Life sciences** setup path with a domain-tuned model and **BiologyExpert** agent.

## What you get

| Piece | Description |
|-------|-------------|
| **OpenBioLLM 8B (chat)** | `koesn/llama3-openbiollm-8b:latest` — recommended Ollama Hub pull (Llama 3 template) |
| **Tool runner** | `qwen2.5:7b` — hub uses this for MCP `analyze_sequence` / `fold_protein` when the chat model has no native tools |
| **nj-bio:8b (optional)** | HF GGUF import with Llama 3 template (branded tag) |
| **BiologyExpert** | Preset agent with bio MCP tools |
| **analyze_sequence** | DNA/RNA/protein checks, length, reverse complement |
| **fold_protein** | ESMFold via Hugging Face Inference → PDB under `~/.neural-junkie/bio/` |
| **Sequence review runbook** | Import from Runbook templates |

## Enable the pack

**Settings → AI & providers → Domain packs** — toggle **Life sciences** on or off.

When enabled:

- **BiologyExpert** is added to configured specialists (hub restarts agents automatically).
- **Biology / Life sciences** appears in **New DM** and channel expert invite lists.
- `koesn/llama3-openbiollm-8b:latest`, `qwen2.5:7b`, and optional `nj-bio:8b` are merged into **models to ensure** for Ollama.

When disabled, pack-owned agents are stopped and removed from the hub. In-process engineering specialists are controlled by the separate [Software development pack](SOFTWARE_DEVELOPMENT_PACK.md); **Moderator**, **Assistant**, and CLI agents are always available.

You can also enable the pack via the **Life sciences** setup wizard track (same toggle in `config.json` under `packs.enabled["life-sciences"]`).

## Install models

### Recommended (Ollama Hub)

```bash
ollama pull koesn/llama3-openbiollm-8b:latest
ollama pull qwen2.5:7b
```

Or use **Model library** (⇧⌘M) → **Ollama** tab → **OpenBioLLM 8B (Llama 3)**.

### First-run wizard

1. Choose **Life sciences & lab work** on the Focus step.
2. Pick **Local Models** (Ollama) or **Cloud** (HF token for hosted OpenBioLLM).
3. Hub ensures **koesn** + **qwen** pulls when Ollama is running.

### Optional: HF GGUF import (`nj-bio:8b`)

1. Open **Hugging Face** tab → **Neural Junkie Bio 8B (GGUF)**.
2. Download the Q4_K_M file.
3. **Import to Ollama** → tag `nj-bio:8b` (imports with Llama 3 chat template).

## Clear a polluted DM thread

Open **channel info** (ℹ️ on the channel header) → **Clear message history**. This wipes hub history for that channel and broadcasts a resync so agents do not replay old errors on restart.

Use this after debugging bad `nj-bio` sessions or instruction-echo replies.

## Settings (no env vars required)

| Setting | Location |
|---------|----------|
| Life sciences pack | **Settings → AI & providers → Domain packs** |
| Hugging Face token (ESMFold + downloads) | **Settings → AI & providers → Hugging Face hub token** (or a `huggingface` provider) |
| Max fold/analyze length, ESMFold model, artifacts dir | **Settings → AI & providers → Life sciences tools** (when pack is on) |
| MCP master switch | Same section — **Enable MCP tool servers** |

Biology MCP starts automatically when the life-sciences pack is on and BiologyExpert is enabled. Stored in `~/.neural-junkie/config.json` under `mcp` and `hf`.

## Tools and Ollama

OpenBio chat models (`koesn/…`, `nj-bio:8b`) do not expose Ollama **tools** capability. BiologyExpert still runs MCP tools: the hub routes tool loops through **`qwen2.5:7b`** on the same Ollama endpoint while keeping **koesn** (or your configured bio tag) for normal chat replies.

Ensure `qwen2.5:7b` is pulled when using the life-sciences pack.

## Create BiologyExpert later

```
/create-expert biology
/create-expert biology MyBioCoach ollama koesn/llama3-openbiollm-8b:latest
```

## Disclaimers

- **Research and education only** — not for clinical diagnosis, treatment, or patient care.
- **In silico** structure predictions are not experimental structures.
- OpenBioLLM and ESMFold outputs may contain errors; validate in the lab.

## Smoke test checklist

1. Life sciences wizard → Ollama → enable BiologyExpert.
2. **Clear message history** on an old Biology DM if replies were echoing instructions.
3. DM with BiologyExpert: paste a short peptide → ask to analyze sequence.
4. Ask to fold the same sequence (HF hub token saved in Settings) → confirm PDB path in reply.
5. Runbook → **sequence-review** → instantiate with BiologyExpert → start execution.

## Out of scope (v1)

- SMILES validation (RDKit)
- scRNA / h5ad
- ESM3, ProtGPT2, GenSLMs
- In-app PDB viewer
