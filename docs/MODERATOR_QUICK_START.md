# Moderator Agent - Quick Start Guide

## What is the Moderator Agent?

The **Chat Moderator** is an always-on system agent that helps users understand and use the Neural Junkie effectively. Think of it as your helpful guide for navigating the chat system.

## Automatic Startup

The moderator automatically starts when you run the server:

```bash
make server
```

You'll see this in the logs:
```
🤖 Initializing moderator agent...
✅ Claude provider initialized for moderator (model: claude-3-5-sonnet-20241022)
✅ Moderator agent started successfully
```

In the chat, you'll see:
```
Chat Moderator joined general
👋 Chat Moderator online! I'm here to help with chat features and commands.
```

## How to Use

### 1. Ask About Commands
```
You: "How do I create a repo agent?"
Chat Moderator: "You can create a repo agent using the /create-repo-agent command..."
```

### 2. Get Help with Features
```
You: "How do mentions work?"
Chat Moderator: "You can mention agents by name (@AgentName) or by type (@backend)..."
```

### 3. Learn About Dispatch
```
You: "What are dispatch commands?"
Chat Moderator: "Dispatch commands let you execute CLI tools. Use /dispatch <command>..."
```

### 4. Safety Net for Unanswered Questions
If you ask a question and no specialized agents respond within 20 seconds, the moderator will step in:

```
You: "What's the best programming language?"
[20 seconds pass with no responses]
Chat Moderator: "👋 I noticed no agents responded to your question. This chat is 
                 designed for development discussions. If you need help with the 
                 chat system itself, feel free to ask me about commands, agents, 
                 or features!"
```

## When the Moderator Responds

The moderator is **passive** and only responds when:

1. **You mention it directly**: `@Chat Moderator` or `@moderator`
2. **You ask about chat features**: Commands, mentions, threads, channels, agents
3. **No one else responds**: After 20 seconds of silence on your question

## What the Moderator Knows

### Commands
All slash commands:
- `/create-repo-agent`, `/create-helper`, `/list-agents`
- `/pause-agent`, `/unpause-agent`, `/delete-agent`
- `/reindex-agent`, `/enable-watch`, `/disable-watch`
- `/dispatch`, `/help`

### Agent Types
- `@frontend` - React, Vue, UI/UX
- `@backend` - APIs, Go, Node.js
- `@devops` - Docker, Kubernetes, dispatch CLI
- `@database` - PostgreSQL, queries, schemas
- `@security` - Auth, encryption, best practices
- `@repo` - Code analysis experts
- `@helper` - Custom expert agents

### System Features
- How to use mentions
- Thread functionality
- Channel management
- Dispatch command workflow
- Creating and managing agents

## Example Conversations

### Getting Started
```
You: "I'm new here, what can I do?"
Chat Moderator: "Welcome! This is an AI collaboration chat where specialized 
                 agents help with development tasks. You can:
                 • Mention agents like @backend or @frontend for help
                 • Create repo experts with /create-repo-agent
                 • Run commands with /dispatch
                 • Type /help to see all commands"
```

### Learning Commands
```
You: "What commands are available?"
Chat Moderator: "Here are the main commands:
                 • /list-agents - See active agents
                 • /create-repo-agent <path> - Create code expert
                 • /create-helper <template> - Create helper agent
                 • /dispatch <command> - Execute CLI commands
                 • /help - Full command list"
```

### Understanding Agents
```
You: "What's a repo agent?"
Chat Moderator: "Repo agents are experts on specific codebases. Create one with:
                 /create-repo-agent /path/to/your/repo [name]
                 
                 They analyze your code and can answer questions about structure,
                 patterns, dependencies, and more. First indexing takes 30-60s,
                 but they cache results for fast responses after that!"
```

## Tips

1. **Be specific**: The moderator knows a lot about the chat system, so ask detailed questions
2. **Try commands**: Just type `/help` to see all available commands
3. **Mention agents**: Use `@backend`, `@frontend`, etc. for technical questions
4. **Use the safety net**: If no one responds, the moderator will guide you

## No Configuration Needed

The moderator:
- ✅ Starts automatically with the server
- ✅ Uses the same AI provider as other agents
- ✅ Requires no special setup or commands
- ✅ Just works!

## Troubleshooting

**Moderator not showing up?**
- Check server logs for "✅ Moderator agent started successfully"
- Make sure the server started without errors

**Moderator not responding?**
- Try mentioning it directly: `@Chat Moderator help`
- Make sure you're asking about chat features or commands
- Remember: it only responds to relevant questions (passive mode)

**Want to test it?**
- Ask: "How do I use commands?"
- Or: "@Chat Moderator what can you help me with?"

## Next Steps

- Read [Moderator Agent Documentation](MODERATOR_AGENT.md) for technical details
- Check out [Getting Started](GETTING_STARTED.md) for system setup
- Learn about [Helper Agents](HELPER_AGENTS.md) and [Repo Agents](REPO_AGENTS.md)

---

**Just start chatting!** The moderator is always ready to help. 🤖

