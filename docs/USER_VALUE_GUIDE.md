# Neural Junkie: Why You Want This

## What It Is

Neural Junkie is your AI engineering team in one app.

Instead of one assistant trying to do everything, you get a coordinated group of specialized agents (backend, frontend, security, DevOps, repo experts, docs experts, and more) that can collaborate in real time.

## Why It Exists

Most AI tooling has three common problems:

- Context gets lost between chats.
- One model is forced to act like every role.
- Cloud usage becomes expensive without clear control.

Neural Junkie fixes that by giving you:

- **Persistent project context** through repo and documentation agents.
- **Role-based specialists** for higher-quality answers and reviews.
- **Local-first options** (Ollama, LM Studio, CLI agents) to reduce cloud token spend.
- **Human approval workflow** for risky actions like file edits and tool calls.

## Who It Is For

- Developers shipping production code
- Technical leads coordinating complex changes
- Teams that want AI acceleration without losing control
- Anyone trying to reduce cloud AI costs with local inference

## How It Works (Simple Version)

1. Open a channel or DM.
2. Ask a question or run a slash command.
3. Mention the right specialist (`@GoExpert`, `@RustExpert`, etc.) or start a collaboration.
4. Review proposals and approve changes when needed.
5. Track provider usage in Settings to understand local vs cloud activity.

## Why Users Like It

- **Faster delivery:** parallel specialist reasoning beats single-agent back-and-forth.
- **Better quality:** domain experts catch more edge cases.
- **Safer automation:** approval gates for tools and file changes.
- **Lower cost control:** clear separation between local and cloud provider activity.
- **Fits real workflows:** channels, DMs, threads, and commands feel like a team chat.

## 5-Minute First Win

1. Start the stack (`make start-all` — hub with in-process specialists + desktop).
2. Open `Settings > AI Providers` and confirm local and/or cloud providers.
3. Create a repo expert:
   - `/create-repo-agent /path/to/repo MyRepoExpert`
4. Ask:
   - `@MyRepoExpert summarize the architecture and top risk areas`
5. Kick off a collaboration for a real task:
   - `/collaborate @GoExpert @SecurityExpert harden auth middleware`

You should immediately get a structured plan, ownership by specialist, and an execution path.

## Cost Positioning (Plain Language)

- **Local providers** (Ollama, LM Studio, CLI-based local models): usually no per-token bill.
- **Cloud providers** (Anthropic / OpenAI-compatible APIs): billed by token usage.
- **Practical strategy:** use local for broad iteration, switch to cloud for final high-accuracy passes.

Neural Junkie is designed for that hybrid workflow by default.

