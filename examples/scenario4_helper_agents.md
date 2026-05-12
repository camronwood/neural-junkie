# Scenario 4: Onboarding with Helper Agents

## Overview

This scenario demonstrates how Helper Agents can streamline the onboarding process for new team members. We'll see the "Day One Expert" guide a new engineer through setup, answer questions, and provide helpful context.

## The Scenario

**Context:** Alex just joined the engineering team today. They have the codebase cloned but haven't set up their development environment yet. They're chatting with the Neural Junkie agents to get oriented.

**Agents Involved:**
- **Day One Expert** (Helper Agent) - Onboarding specialist
- **Backend Expert** - For code-specific questions
- **GoExpert** (Repo Agent) - For codebase navigation

---

## The Conversation

### 1. Initial Setup Question

```
Alex: I just joined the team today! How do I get my dev environment set up?

Day One Expert: Welcome to the team, Alex! 🎉 Let's get you up and running. 

For development environment setup, you'll need:

1. **Git** - Version control (you probably have this!)
2. **Go 1.21+** - Check with `go version`
3. **Docker Desktop** - For running services locally
4. **Make** - Build automation tool

Once you have these installed, the setup is simple:
1. Copy `env.example` to `env.local`
2. Run `make install` to install dependencies
3. Run `make start-all` to start all services

What step are you on? Do you have all the required tools installed?
```

### 2. Specific Tool Question

```
Alex: I have Git and Go, but I'm not sure about Docker. How do I check if I have it?

Day One Expert: Great! You're halfway there. To check if you have Docker:

```bash
docker --version
```

If you get a version number, you're good! If not, download Docker Desktop from:
https://www.docker.com/products/docker-desktop

Once installed, make sure it's running (you should see the Docker icon in your system tray/menu bar).

After Docker is ready, you can proceed with copying `env.example` to `env.local`.
```

### 3. Configuration Question

```
Alex: OK Docker is running! I copied env.example to env.local. What do I need to change in it?

Day One Expert: Good progress! In `env.local`, you'll need to configure:

1. **ANTHROPIC_API_KEY** - Your AI API key (ask your team lead or check the team password manager)
2. **USE_AI_HUB** - Set to `true` if using our AI Hub, `false` for direct Anthropic API
3. **AI_HUB_ENDPOINT** - Your AI Hub endpoint URL (if using AI Hub)

The other settings can stay as-is for local development. Once you've added your API key, load the environment:

```bash
source load-env.sh
```

Then you're ready to start services with `make start-all`!
```

### 4. Code-Specific Question (Triggering Different Agent)

```
Alex: Thanks! Services are running. Now I'm looking at the code - where's the main API server code?

Backend Expert: The main API server is in `cmd/server/main.go`. This is the entry point that:
- Starts the HTTP server on port 18765
- Sets up WebSocket handlers for real-time communication
- Initializes the hub for message routing
- Serves the web UI

The core server logic is in `internal/hub/hub.go` which handles:
- Agent registration and management
- Channel creation and messaging
- Message broadcasting to subscribers

Want me to explain any specific part of the architecture?
```

### 5. Repository Navigation

```
Alex: @GoExpert where are the tests located?

GoExpert: Tests are organized in the `test/` directory. Here's what we have:

- **test/deduplication_test.go** - Message deduplication tests
- **test/repo_agent_test.go** - Repository agent tests
- **test/cache_test.go** - Cache performance tests
- **test/gui_test.go** - GUI threading tests

We also have unit tests in various `internal/` packages. Run all tests with:
```bash
make test
```

The test structure follows Go conventions - test files are named `*_test.go`.
```

### 6. Process Question (Back to Day One Expert)

```
Alex: How do I run the tests? And what's the workflow for making changes?

Day One Expert: Great questions! Here's our development workflow:

**Running Tests:**
```bash
make test          # Run all tests
make test-all      # Run comprehensive tests
```

**Making Changes Workflow:**
1. Create a feature branch: `git checkout -b feature/your-feature`
2. Make your changes
3. Run tests: `make test`
4. Run linter: `make lint` (if available)
5. Commit with clear message
6. Push and create PR: `git push origin feature/your-feature`
7. Request review from 2 team members
8. Address feedback
9. Merge after approval

For your first contribution, I'd recommend:
1. Pick a "good first issue" from GitHub
2. Pair program with a team member
3. Get familiar with our coding patterns

Check `docs/DEVELOPMENT_NOTES.md` for more detailed guidelines!
```

### 7. Asking for Next Steps

```
Alex: This is really helpful! What should I focus on for the rest of today?

Day One Expert: Excellent question! Here's a suggested Day 1 plan:

**Morning (Remaining):**
- ✅ Dev environment setup (done!)
- ✅ Run the application (done!)
- 📖 Read the README.md
- 📖 Skim through docs/ARCHITECTURE.md

**Afternoon:**
- 🔍 Explore the codebase (use the repo agents!)
- 🧪 Run the tests to see them pass
- 🎯 Find a "good first issue" to work on
- 💬 Schedule 1:1s with team members

**End of Day:**
- ✍️ Write down questions for tomorrow
- 👋 Say hi in #engineering Slack channel
- 🎉 Celebrate getting set up!

Don't hesitate to ask questions - that's what we're here for! Tomorrow we can dive deeper into the architecture and get you started on your first task.
```

### 8. Cross-Agent Collaboration

```
Alex: @Backend @Day-One I want to understand how the agents communicate. Can you both explain from your perspectives?

Backend Expert: From a technical perspective, agents communicate via the Hub using a pub/sub pattern:

1. Agents subscribe to channels
2. Messages are broadcast to all subscribers
3. Each agent decides whether to respond based on mentions and expertise
4. Responses are sent back through the hub

The core code is in `internal/hub/hub.go` and `internal/agent/agent.go`. We use Go channels and goroutines for concurrency.

Day One Expert: To add context for you as a new team member - this architecture lets multiple specialized agents collaborate naturally! 

Think of it like a Slack channel where experts listen and chime in when they have relevant input. Each agent:
- Has specific expertise (backend, frontend, security, etc.)
- Decides when to respond based on the question
- Can mention other agents for collaboration

As you build features, you can test them by chatting with the agents and seeing their responses. It's great for getting quick feedback!

Does the pub/sub pattern make sense? Feel free to ask more technical questions as you explore the code.
```

### 9. Testing Expert Introduction

```
Alex: How should I write tests for my first contribution?

Testing Expert: Welcome! For your first contribution, follow our testing best practices:

**Test Structure (AAA Pattern):**
```go
func TestFeatureName(t *testing.T) {
    // Arrange - Set up test data and dependencies
    input := "test input"
    expected := "expected output"
    
    // Act - Execute the code under test
    result := YourFunction(input)
    
    // Assert - Verify the results
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

**Key Principles:**
1. Test behavior, not implementation
2. One logical assertion per test
3. Clear test names describing what's tested
4. Independent tests (no shared state)

Start with a simple unit test for your changes. Run it with:
```bash
go test ./... -v
```

Need help with a specific test? Feel free to ask!
```

---

## Key Takeaways

### 1. **Seamless Onboarding**
- New engineer gets immediate, contextual help
- No waiting for teammates to be available
- Consistent onboarding experience

### 2. **Multi-Agent Collaboration**
- Day One Expert handles process questions
- Backend Expert handles technical architecture
- Repo Agent helps with code navigation
- Testing Expert guides on testing practices

### 3. **Progressive Disclosure**
- Information provided when needed
- Doesn't overwhelm with everything at once
- Builds on previous knowledge

### 4. **Knowledge Capture**
- Tribal knowledge encoded in helper agents
- Processes documented and accessible
- Reduces repeated questions

### 5. **24/7 Availability**
- Help available any time
- No timezone constraints
- Instant responses

---

## Customizing for Your Team

To replicate this for your team:

### 1. **Create Day One Expert**
```bash
/create-helper day-one
```

### 2. **Add Your Team's Knowledge**
Edit files in `~/.neural-junkie/helpers/day-one/knowledge/`:

```markdown
# Our Dev Setup

## Required Tools
1. Git 2.40+
2. Python 3.11+
3. PostgreSQL 15
4. Redis 7

## Setup Steps
[Your team's specific steps]

## Common Issues
[Team-specific troubleshooting]
```

### 3. **Add Team-Specific Keywords**
Edit `config.json` to include your terminology:
```json
{
  "keywords": [
    "setup",
    "onboard",
    "your-custom-tool-name",
    "your-process-name"
  ]
}
```

### 4. **Create Additional Helpers**
```bash
/create-helper testing-expert
/create-helper docs-expert
```

Add knowledge specific to your team's practices!

---

## Measuring Success

Track these metrics to see the impact:

### Onboarding Time
- **Before:** 2-3 days to get fully set up
- **After:** <1 day with Day One Expert

### Repeated Questions
- **Before:** Same questions asked repeatedly in Slack
- **After:** Questions answered by helper agents

### Documentation Access
- **Before:** Documentation scattered and hard to find
- **After:** Centralized in helper agent knowledge base

### New Engineer Confidence
- **Before:** Hesitant to ask "basic" questions
- **After:** Comfortable asking helper agents anything

---

## Advanced Patterns

### Progressive Onboarding

Create different helpers for different phases:

```bash
# Day 1: Basic setup
/create-helper day-one

# Week 1: Architecture and patterns
/create-helper architecture-guide

# Month 1: Advanced topics
/create-helper advanced-patterns
```

### Team-Specific Experts

```bash
# Frontend team helper
/create-helper frontend-onboarding

# DevOps team helper  
/create-helper devops-onboarding

# Each with team-specific knowledge!
```

### Domain Experts

```bash
# Business domain expert
/create-helper domain-expert

# Add knowledge about:
# - Business rules
# - Domain terminology
# - Use cases and workflows
```

---

## Conclusion

Helper Agents transform onboarding from a manual, inconsistent process into an automated, consistent experience. The Day One Expert provides immediate guidance, collaborates with other agents, and scales to help multiple new engineers simultaneously.

**Time Investment:**
- Setup: 30 minutes
- Knowledge addition: 2-4 hours
- Maintenance: 1 hour/month

**ROI:**
- Each new engineer saves 1-2 days
- Reduced burden on senior engineers
- More consistent onboarding
- Better documentation culture

Start with the built-in templates and customize them with your team's knowledge!

