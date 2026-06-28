# IDE v4 — build the IDE you actually own

> If Elon Musk bought your favorite IDE, come join our open-source effort to build a better one.

I'm shipping **Neural Junkie IDE v4** — not as a reaction tweet, but as years of local-first, multi-agent workspace work reaching the depth developers expect from a serious editor.

## Why this moment matters

AI coding tools are consolidating fast. When a beloved IDE becomes an acquisition target, developers rightly ask:

- Who controls the roadmap?
- Where does my code go by default?
- Can I still run models locally?
- What happens to the workflow I built my team around?

Open source doesn't solve everything — but it answers the ownership question clearly.

## What v4 adds

### Full Monaco LSP (local workspaces)

We moved from **LSP-lite** (one-shot `gopls check`) to **persistent language servers** behind a hub WebSocket:

- Hover in the editor (live in Monaco)
- Foundation for completion, go-to-definition, rename
- Go, Rust, Python via `gopls`, `rust-analyzer`, `pyright-langserver`

Your hub spawns and owns the language server process. The desktop speaks JSON-RPC over WebSocket — same pattern as VS Code, without sending your repo to our cloud (there is no our cloud).

### Remote SSH via `nj-remote`

Workspaces are no longer "a path on the laptop only."

We added **`WorkspaceBackend`** — a pluggable FS/exec layer — and a small **`nj-remote` sidecar** you run on EC2, a dev box, or anywhere SSH reaches:

```bash
nj-remote -root ~/myapp -addr :19876 -token "$SECRET"
```

The desktop hub routes file operations, git, LSP, and implementation-session verify steps through the backend. Agents stay local; **your code stays where you put it**.

### Dev containers

Repos with `.devcontainer/devcontainer.json` get an attach plan from the hub. Run `nj-remote` inside the container after `devcontainer up` — same sidecar protocol, container as the host.

### tree-sitter symbols

Symbol search upgrades from regex to **tree-sitter** when the CLI is on PATH, with regex fallback. ⌘⇧O keeps working; nested methods stop hiding.

## What didn't change (on purpose)

- **Human approval on file changes** — proposals and diffs, not silent writes
- **Multi-agent specialists** — not one overworked chatbot
- **BYOM** — Ollama, Claude, GPT, LM Studio, Hugging Face — your keys, your machine
- **Local-first hub** — loopback by default

## Join the build

Neural Junkie is **open source** (see the repo license). v4 is the invitation:

1. **Download** the beta — macOS, Windows, Linux
2. **Try IDE layout** + Software development pack
3. **Open an issue** if remote/LSP breaks on your stack
4. **Send a PR** — `WorkspaceBackend`, LSP, sidecar, scenarios

If the IDE you loved now has a new owner, you don't have to wait for their roadmap.

**Come build the next one with us.**

https://github.com/camronwood/neural-junkie

**Download CTA:** https://camronwood.github.io/neural-junkie/download.html

**Suggested post date:** After `v1.2.0-beta.2` tag (CI + remote LSP dogfood).

---

Related: [IDE_V4.md](../IDE_V4.md) · [REMOTE_WORKSPACES.md](../REMOTE_WORKSPACES.md) · [assets/marketing/ide-v4-open-source-ad.md](../../assets/marketing/ide-v4-open-source-ad.md)
