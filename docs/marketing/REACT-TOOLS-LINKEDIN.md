# LinkedIn article — ReAct Tool Wrapper (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-react-tools-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-react-tools-article.sh`

**Suggested title (pick one):**
- Gemma Can't Call Tools. We Taught It Anyway.
- Tools Without Native Tool Calling in Neural Junkie
- ReAct MCP Loops for Local Models That Lack Function Calling

**Feed post teaser:**
> Strong local models like Gemma 3 12B reason well but lack native function calling. Neural Junkie's ReAct wrapper runs MCP tools on the same model — with Qwen swap as a safety net when parsing fails.

**Hashtags:** `#AI #LocalAI #Ollama #DeveloperTools #OpenSource #MultiAgent`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/react-tools.html

**Download CTA:** https://camronwood.github.io/neural-junkie/download.html

**Related articles:** [Model layering](MODEL-LAYERING-LINKEDIN.md) · [Loop stack](LOOP-STACK-LINKEDIN.md) · [Inference layer](INFERENCE-LAYER-LINKEDIN.md)

---

## PASTE START

You've found a local model that's great at chat — clear reasoning, solid code explanations, maybe even vision — and then you ask it to **read a file** or **run an MCP tool**.

Nothing happens. Or worse: the hub silently swaps in a second model for the tool loop, and you lose the persona you picked.

**Gemma 3 12B** is in that bucket for many Ollama installs: strong mid-tier model, no native `tools` capability in `/api/show`. Until now, Neural Junkie handled that the same way we handle biology chat models — route the tool loop through **Qwen**.

That works. It also means **two models in RAM**, split reasoning, and routing telemetry that says `tool_fallback` instead of one coherent brain.

## The problem in one sentence

**Good chat model + no native tool calling = either no tools, or a different model runs the loop.**

## What we shipped: three-tier tool routing

Neural Junkie now picks a tool path in order:

1. **Native tools** — Ollama `tool_calls` or Claude `tool_use` when the model supports it (Qwen, coder models).
2. **ReAct on the same model** — for allowlisted tags like `gemma3:12b`, the hub wraps the chat provider and runs an MCP loop via structured text.
3. **Qwen swap fallback** — if ReAct hits the iteration cap or can't parse a tool call, we fall back to the existing tool-loop model.

You see which path ran on the typing indicator: `chat: gemma3:12b (react_tools)` or `(react_fallback_swap)` when the safety net kicks in.

## The contract: tagged JSON, not free-form Yao ReAct

We tried to keep parsing reliable. The model emits **one block per tool step**:

```
<tool_call>{"name":"read_file","arguments":{"path":"src/App.tsx"}}</tool_call>
```

When it's done gathering data, it replies normally — no `<tool_call>` block.

The hub executes the MCP tool, feeds the observation back into the loop, and continues until the model answers or hits the configured iteration cap (same guardrails as native tool loops).

## Why Gemma 3 12B first

- **Single ~8 GB model** for chat + light MCP (read, grep, list) without pulling Qwen.
- **Benchmarked well for Assistant chat** in our release profiles — it was already the optional upgrade; now it can *act*, not just talk.
- **Honest limits:** heavy implementation loops (`search_replace`, multi-file edits) are still harder than native Qwen tool calling. That's why tier 3 exists.

Biology and CAD domain LoRAs stay on the Qwen swap by default — different use case, different default list.

## Configuration

In `config.json` under `ollama`:

```json
"react_tools_enabled": true,
"react_tool_models": ["gemma3:12b"]
```

Add other non-native tags when you've validated them. Turn off `react_tools_enabled` to restore swap-only behavior.

Agent info in the desktop app shows **react** on the tool loop badge when this path is active.

## What this is not

- Not a replacement for native tool calling when your model already supports it.
- Not LM Studio / OpenAI-compat yet — same `ReActToolProvider` pattern, follow-up PR.
- Not magic: smaller models will still drift format under pressure. The fallback exists because production agents need a floor.

## Try it

1. Pull `gemma3:12b` in Ollama.
2. Point an agent or Assistant at that tag.
3. Ask it to read a file in a shared workspace.
4. Watch routing show `(react_tools)` and the tool execute without loading Qwen.

**Neural Junkie** is local-first, open source, and built for specialists that share tools, approvals, and routing policy — not one monolithic planner guessing in a vacuum.

Repo: https://github.com/camronwood/neural-junkie

## PASTE END
