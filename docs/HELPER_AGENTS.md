# Helper Agents - Custom Expert System

## Overview

Helper Agents are **customizable expert agents** that specialize in specific knowledge domains beyond code repositories. They're perfect for creating specialized assistants like onboarding guides, testing experts, documentation specialists, or any domain-specific helper you need.

Unlike Repository Agents that analyze codebases, Helper Agents are powered by **knowledge bases** - collections of markdown/text files that define their expertise.

## Key Features

- 🎯 **Domain-Specific Expertise** - Create agents for onboarding, testing, documentation, etc.
- 📚 **Knowledge Base System** - Powered by markdown/text documents you provide
- 🔍 **Smart Context Matching** - Automatically finds relevant knowledge for each question
- 🚀 **Instant Setup** - Pre-configured templates ready to use
- ✏️ **Fully Customizable** - Add your own knowledge documents and keywords
- 💾 **Persistent Storage** - Configurations saved to `~/.neural-junkie/helpers/`

## Quick Start

### Create a Helper Agent

```bash
# In your chat interface:
/list-helper-templates

# Create from template:
/create-helper day-one
/create-helper testing-expert
/create-helper docs-expert
```

### Start Helper Agent (Alternative Methods)

**Via Chat Command** (Recommended):
```bash
# Create and start in chat:
/create-helper day-one
```

**Via Make Command**:
```bash
# Start a helper agent that's already created:
make helper-agent NAME=day-one CHANNEL=general
```

**Via Direct Command**:
```bash
# Load environment and run directly:
source load-env.sh
go run cmd/helper-agent/main.go --name day-one --channel general
```

### Manage Helper Agents

```bash
/list-agents                    # See all active agents
/pause-agent "Day One Expert"   # Pause responses
/unpause-agent "Day One Expert" # Resume
/delete-agent "Day One Expert"  # Remove agent
```

## Pre-Built Templates

### 1. Day One Expert 👋

**Purpose:** Helps new engineers get set up and oriented

**Use Cases:**
- Development environment setup
- Tool installation and configuration
- Understanding team processes
- First week tasks and onboarding
- "Where do I find X?" questions

**Keywords:** setup, install, onboard, getting started, first day, new engineer, environment, configure

**Example Questions:**
- "How do I set up my development environment?"
- "What tools do I need to install?"
- "Where can I find the team documentation?"
- "What should I do on my first day?"

### 2. Testing Expert 🧪

**Purpose:** Guides developers on testing practices and frameworks

**Use Cases:**
- Unit testing best practices
- Integration and E2E testing
- Test-Driven Development (TDD)
- Mocking and test fixtures
- Code coverage strategies

**Keywords:** test, testing, tdd, mock, coverage, assertion

**Example Questions:**
- "How should I structure my unit tests?"
- "What's the best way to mock external services?"
- "Should I test private methods?"
- "How do I increase code coverage effectively?"

### 3. Documentation Expert 📝

**Purpose:** Helps with documentation standards and API docs

**Use Cases:**
- Writing clear documentation
- API documentation standards
- README best practices
- Code comments guidelines
- Knowledge sharing tips

**Keywords:** documentation, docs, readme, document, api docs

**Example Questions:**
- "How should I document this API endpoint?"
- "What makes a good README?"
- "How much should I comment my code?"
- "What should go in our project documentation?"

### 4. Subenv Expert 🚀

**Purpose:** Specialist in subenvironment management, kubectl operations, pod logs, database queries, and remote development

**Use Cases:**
- Managing subenvironments (create, deploy, wake, destroy)
- kubectl operations through dispatch CLI
- Checking pod logs and troubleshooting issues
- Database queries and debugging
- Remote development setup with DevSpace
- Port forwarding and file synchronization

**Keywords:** subenv, subenvironment, pod, logs, kubectl, container, deployment, service, namespace, database, query, exec, remote, devspace, sync, port-forward

**Example Questions:**
- "What pods are running in my dolphin subenv?"
- "How do I check the logs for the API service?"
- "Can you help me query the database to see how many orders we have?"
- "How do I set up remote development with DevSpace?"
- "My pod keeps crashing, can you help me debug it?"
- "How do I port forward to access my service locally?"

**Special Features:**
- **Dispatch CLI Integration**: Uses dispatch commands for consistency
- **Database Expertise**: PostgreSQL, MySQL, Redis query assistance
- **Remote Development**: DevSpace configuration and troubleshooting
- **Kubernetes Operations**: Pod management, service discovery, resource monitoring
- **Troubleshooting**: Common subenv issues and solutions

## Customizing Helper Agents

### Knowledge Base Structure

Helper agent knowledge is stored in:
```
~/.neural-junkie/helpers/<template-name>/
├── config.json          # Agent configuration
└── knowledge/           # Knowledge base documents
    ├── overview.md      # Main guide (auto-created)
    └── ...              # Add your own .md or .txt files
```

### Adding Custom Knowledge

1. **Find your agent's knowledge directory:**
   ```bash
   cd ~/.neural-junkie/helpers/day-one/knowledge/
   ```

2. **Add markdown files with your content:**
   ```bash
   # Create a new guide
   cat > dev-setup.md << 'EOF'
   # Development Setup Guide
   
   ## Prerequisites
   - Install Docker Desktop
   - Install Node.js 18+
   - Install Go 1.21+
   
   ## Steps
   1. Clone the repository
   2. Copy env.example to env.local
   3. Run `make install`
   4. Run `make start-all`
   EOF
   ```

3. **The agent automatically loads all .md and .txt files in the knowledge directory!**

### Knowledge File Best Practices

✅ **DO:**
- Use clear markdown headings (`#`, `##`, `###`)
- Include specific examples and code snippets
- Keep documents focused on specific topics
- Use bullet points and numbered lists
- Add relevant keywords naturally

❌ **DON'T:**
- Create overly long documents (split into smaller files)
- Use vague or generic content
- Forget to update when processes change
- Use proprietary or sensitive information

### Example Knowledge Document

```markdown
# Git Workflow

## Branching Strategy
We use trunk-based development with short-lived feature branches.

### Creating a Feature Branch
```bash
git checkout main
git pull origin main
git checkout -b feature/your-feature-name
```

### Making Changes
1. Make your changes
2. Write tests
3. Run `make test`
4. Commit with clear messages

### Pull Request Process
1. Push your branch: `git push origin feature/your-feature-name`
2. Create PR in GitHub
3. Request review from 2 team members
4. Address feedback
5. Merge after approval

## Common Issues

**Merge Conflicts:**
```bash
git fetch origin main
git rebase origin/main
# Resolve conflicts
git rebase --continue
```
```

## How Helper Agents Work

### 1. Knowledge Loading
When created, the agent:
- Loads all `.md` and `.txt` files from its knowledge directory
- Extracts headings to build a topic index
- Keeps knowledge in memory for fast access

### 2. Question Matching
When a user asks a question:
- The agent checks if it's mentioned or if keywords match
- It finds relevant knowledge documents based on query keywords
- Multiple keyword matches = higher relevance

### 3. Response Generation
The agent:
- Includes relevant knowledge in its prompt to the AI
- References specific documentation when answering
- Provides step-by-step guidance when appropriate
- Maintains conversation context

### 4. Smart Triggering
Helper agents respond when:
- **Explicitly mentioned:** `@"Day One Expert" how do I set up?`
- **Keywords match:** Message contains configured keywords
- **Expertise relevant:** Question matches their expertise area
- **Direct questions:** Questions in their domain

## Configuration Format

Each helper agent has a `config.json`:

```json
{
  "name": "Day One Expert",
  "description": "Helps new engineers get set up",
  "expertise": [
    "Onboarding",
    "Development Environment Setup",
    "Tool Installation"
  ],
  "keywords": [
    "setup",
    "install",
    "onboard",
    "getting started"
  ],
  "system_prompt": "You are the Day One Expert...",
  "knowledge_path": "/Users/you/.neural-junkie/helpers/day-one/knowledge"
}
```

## Creating Custom Templates

Want to create your own helper agent type? Here's how:

### 1. Create Knowledge Directory
```bash
mkdir -p ~/.neural-junkie/helpers/my-custom-expert/knowledge
```

### 2. Create Configuration
```bash
cat > ~/.neural-junkie/helpers/my-custom-expert/config.json << 'EOF'
{
  "name": "My Custom Expert",
  "description": "Specialized expert for X",
  "expertise": [
    "Topic 1",
    "Topic 2",
    "Topic 3"
  ],
  "keywords": [
    "keyword1",
    "keyword2",
    "keyword3"
  ],
  "system_prompt": "You are My Custom Expert, specializing in...",
  "knowledge_path": "/Users/YOUR_USERNAME/.neural-junkie/helpers/my-custom-expert/knowledge"
}
EOF
```

### 3. Add Knowledge Documents
```bash
cd ~/.neural-junkie/helpers/my-custom-expert/knowledge
# Create your markdown files here
```

### 4. Load in Code (Optional)
To make it a built-in template, add it to `internal/agent/helper_agent.go`:
```go
"my-custom": {
    Name:        "My Custom Expert",
    Description: "...",
    Expertise:   []string{"..."},
    // ... rest of config
},
```

## Use Cases

### Onboarding Automation
```markdown
**Problem:** New engineers spend 2-3 days getting set up
**Solution:** Day One Expert with company-specific setup guides
**Result:** Setup time reduced to <1 day, fewer blockers
```

### Code Quality Standards
```markdown
**Problem:** Inconsistent testing practices across teams
**Solution:** Testing Expert with team testing guidelines
**Result:** Better test coverage, consistent patterns
```

### Documentation Culture
```markdown
**Problem:** Developers unsure how to document features
**Solution:** Docs Expert with templates and examples
**Result:** More complete, consistent documentation
```

### Domain-Specific Experts
```markdown
**Problem:** Complex domain knowledge scattered across docs
**Solution:** Custom expert with domain-specific knowledge base
**Result:** Easier onboarding, fewer repeated questions
```

## Evolution Ideas 🚀

Future enhancements for helper agents:

### Week One Progression
Track user progress and evolve responses:
- Day 1: Basic setup and tools
- Day 2-3: Architecture and codebase
- Day 4-5: First tasks and workflows
- Week 2+: Advanced topics

### Learning Tracking
Remember what user has learned:
- Don't repeat already-covered topics
- Suggest next learning steps
- Identify knowledge gaps

### Interactive Guides
Step-by-step walkthroughs:
- Track completion status
- Verify each step
- Provide troubleshooting

### Team Knowledge Capture
Automatically update knowledge base:
- Learn from repeated questions
- Extract knowledge from conversations
- Suggest documentation improvements

## Troubleshooting

### Agent Not Responding

**Check if agent is active:**
```bash
/list-agents
```

**If paused, unpause:**
```bash
/unpause-agent "Day One Expert"
```

**Try mentioning explicitly:**
```
@"Day One Expert" how do I set up my environment?
```

### Knowledge Not Being Used

**Verify knowledge files exist:**
```bash
ls -la ~/.neural-junkie/helpers/day-one/knowledge/
```

**Check file format:**
- Must be `.md` or `.txt` files
- Must have actual content
- Use clear markdown headings

**Keywords not matching:**
- Add more keywords to config
- Make questions more specific
- Include domain-specific terms

### Agent Creation Fails

**Check permissions:**
```bash
ls -la ~/.neural-junkie/
```

**Verify API key is set:**
```bash
source load-env.sh
echo $ANTHROPIC_API_KEY
```

**Try with different template:**
```bash
/create-helper testing-expert
```

## Best Practices

### Knowledge Organization
- ✅ One topic per file
- ✅ Clear, descriptive filenames
- ✅ Consistent heading structure
- ✅ Include examples and code snippets
- ✅ Link between related documents

### Keyword Selection
- ✅ Use natural language terms
- ✅ Include common variations
- ✅ Add domain-specific jargon
- ✅ Keep keywords focused
- ✅ Test with real questions

### Maintenance
- ✅ Review knowledge quarterly
- ✅ Update when processes change
- ✅ Remove outdated information
- ✅ Add based on common questions
- ✅ Keep sensitive data out

### Usage Tips
- ✅ Create agents for frequently asked questions
- ✅ Use for onboarding new team members
- ✅ Standardize processes with guides
- ✅ Capture tribal knowledge
- ✅ Mention agents explicitly for faster responses

## Architecture

### Storage Structure
```
~/.neural-junkie/helpers/
├── day-one/
│   ├── config.json
│   └── knowledge/
│       ├── overview.md
│       ├── setup.md
│       └── tools.md
├── testing-expert/
│   ├── config.json
│   └── knowledge/
│       └── overview.md
└── docs-expert/
    ├── config.json
    └── knowledge/
        └── overview.md
```

### Code Components
- `internal/agent/helper_agent.go` - Helper agent implementation
- `internal/agent/helper_storage.go` - Storage and persistence
- `internal/hub/commands.go` - Chat commands
- `internal/protocol/types.go` - AgentTypeHelper constant

## Comparison: Helper vs Repo Agents

| Feature | Helper Agents | Repo Agents |
|---------|---------------|-------------|
| **Purpose** | Domain knowledge & guidance | Code analysis |
| **Knowledge Source** | Markdown/text documents | Repository files & git history |
| **Use Cases** | Onboarding, testing, docs | Code questions, architecture |
| **Setup Time** | Instant | 30-60s first time |
| **Customization** | Add markdown files | Automatic from code |
| **Best For** | Processes, guidelines, FAQs | Code-specific questions |

## Examples

### Example 1: Onboarding Flow
```
User: I just joined the team. Where do I start?

Day One Expert: Welcome to the team! 🎉 Let's get you set up. Based on our onboarding 
guide, here's your Day 1 checklist:

1. Set up your dev environment (I can help with this!)
2. Get access to GitHub, Slack, and Jira
3. Read the project README and architecture docs
4. Schedule 1:1s with your onboarding buddy

What would you like to start with?
```

### Example 2: Testing Question
```
User: How should I test this API endpoint?

Testing Expert: For API endpoint testing, I recommend a 3-layer approach:

1. **Unit tests**: Test the handler function in isolation with mocked dependencies
2. **Integration tests**: Test the endpoint with a real database (use test fixtures)
3. **E2E tests**: Test the full request/response cycle

Start with a unit test using our standard AAA pattern (Arrange-Act-Assert). 
Would you like me to provide a code example?
```

### Example 3: Documentation Help
```
User: What should I include in this API's documentation?

Docs Expert: Great question! For API documentation, include these sections:

1. **Overview**: What the endpoint does (1-2 sentences)
2. **Request**: Method, path, parameters, body schema
3. **Response**: Success/error codes, response schema
4. **Examples**: Real request/response examples
5. **Notes**: Rate limits, authentication, edge cases

Use our API doc template in `docs/api-template.md` as a starting point.
```

## Summary

Helper Agents bring **specialized expertise** to your Neural Junkie through **customizable knowledge bases**. They're perfect for:

- 🎓 Onboarding new team members
- 📖 Standardizing processes and guidelines
- 🔧 Providing domain-specific expertise
- ❓ Answering frequently asked questions
- 🚀 Capturing and sharing tribal knowledge

Start with the built-in templates (`day-one`, `testing-expert`, `docs-expert`) and customize them with your team's knowledge!

---

**Next Steps:**
1. Try creating a helper agent: `/create-helper day-one`
2. Add custom knowledge documents to `~/.neural-junkie/helpers/day-one/knowledge/`
3. Ask questions and see how the agent helps!
4. Create custom templates for your team's needs

**See Also:**
- [Repository Agents](./REPO_AGENTS.md) - For code-specific analysis
- [Architecture](./ARCHITECTURE.md) - System design details
- [Getting Started](./GETTING_STARTED.md) - Initial setup guide

