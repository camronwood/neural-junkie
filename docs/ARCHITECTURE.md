# Architecture Documentation

## System Overview

The Neural Junkie is a multi-agent collaboration system that enables AI agents with different specializations to communicate, collaborate, and solve complex problems together.

## Core Components

### 1. Chat Hub (`internal/hub/`)

The central message broker and state manager.

**Responsibilities:**
- Manage channels (chat rooms)
- Route messages between agents
- Maintain message history
- Handle agent registration and presence
- Provide real-time subscriptions via channels

**Key Types:**
```go
type Hub struct {
    channels    map[string]*protocol.Channel
    agents      map[string]*protocol.AgentInfo
    messages    map[string][]*protocol.Message
    subscribers map[string][]chan *protocol.Message
}
```

### 2. Protocol (`internal/protocol/`)

Defines the message protocol and data structures.

**Message Types:**
- `chat` - Regular conversation
- `question` - Explicit questions
- `answer` - Responses to questions
- `system_info` - System notifications
- `agent_join` / `agent_leave` - Presence updates
- `context_share` - Sharing code/context
- `request_help` - Explicit help requests

**Agent Types:**
- `frontend` - UI/UX, React, Vue, Angular
- `backend` - APIs, services, architecture
- `devops` - Deployment, infrastructure, CI/CD
- `database` - SQL, schema, optimization
- `security` - Auth, vulnerabilities, compliance

### 3. Agent Framework (`internal/agent/`)

Base framework for creating AI agents.

**Agent Lifecycle:**
1. **Creation** - Initialize with type, name, expertise
2. **Registration** - Register with hub
3. **Join Channel** - Enter a conversation
4. **Message Processing** - Listen and respond
5. **Shutdown** - Leave gracefully

**Decision Logic:**

Agents decide to respond based on:
- Direct mentions (`@AgentName`)
- Questions in their domain
- Keywords matching expertise
- Relevance to their specialty

**Example:**
```go
func (a *Agent) shouldRespond(msg *protocol.Message) bool {
    // Always respond if mentioned
    if msg.IsMentioned(a.Info.ID) { return true }
    
    // Respond to questions in our domain
    if isQuestion(msg.Content) && 
       matchesExpertise(msg.Content, a.Info.Expertise) {
        return true
    }
    
    return false
}
```

### 4. AI Providers (`internal/ai/`)

Interface for AI response generation.

**Current Implementations:**
- `MockProvider` - Rule-based responses for demo
- `ClaudeProvider` - Placeholder for Claude API integration

**Interface:**
```go
type AIProvider interface {
    GenerateResponse(ctx context.Context, prompt string, 
                    history []protocol.Message) (string, error)
    GetModel() string
}
```

### 5. Server (`cmd/server/`)

HTTP/WebSocket server providing:
- REST API for interactions
- WebSocket for real-time updates
- Embedded web UI
- Message routing

**API Endpoints:**
- `GET /` - Web UI
- `GET /ws?channel=X` - WebSocket connection
- `GET /api/channels` - List channels
- `POST /api/channels/create` - Create channel
- `POST /api/channels/join` - Join channel
- `GET /api/agents` - List agents
- `POST /api/agents` - Register agent
- `GET /api/messages?channel=X` - Get history
- `POST /api/send` - Send message

### 6. Agent Runner (`cmd/agent/`)

Standalone executable to run agents.

**Features:**
- Command-line configuration
- Auto-generates agent names
- Connects to hub server
- Handles graceful shutdown

### 7. CLI Tool (`cmd/cli/`)

Command-line interface for interaction.

**Capabilities:**
- Send messages
- List channels/agents
- View history
- Watch for updates
- Create channels

## Data Flow

### Message Flow

```
User/Agent → Hub → Broadcast to Subscribers
                    ↓
            [Agent 1, Agent 2, ...]
                    ↓
            Decision: Should Respond?
                    ↓
            AI Generation
                    ↓
            Response → Hub → Broadcast
```

### Agent Response Flow

```
1. Receive Message
   ↓
2. Add to History
   ↓
3. Check if should respond
   - Mentioned?
   - Relevant question?
   - Matching expertise?
   ↓
4. If yes: Build Prompt
   - Agent role & expertise
   - Other agents in channel
   - Recent conversation
   - Current message
   ↓
5. Generate Response via AI
   ↓
6. Send to Hub
```

## Design Patterns

### 1. Pub/Sub Pattern

The hub uses a publish-subscribe pattern for real-time updates:
- Agents subscribe to channels
- Messages are broadcast to all subscribers
- Subscribers receive via Go channels

### 2. Observer Pattern

Agents observe message streams and react based on their logic.

### 3. Strategy Pattern

AI providers implement a common interface, allowing different strategies:
- Mock responses for testing
- Claude API for production
- Custom providers for specific needs

### 4. Factory Pattern

`AgentFactory` creates specialized agents based on type:
```go
agent, _ := AgentFactory(protocol.AgentTypeBackend, "Go Expert", ai, hub)
```

## Concurrency Model

### Thread Safety

- **Hub**: Protected by `sync.RWMutex`
- **Agents**: Run in separate goroutines
- **Message Channels**: Buffered Go channels (size 100)

### Goroutines

Each agent spawns:
1. **Main goroutine** - Message processing loop
2. **Subscription goroutine** - Listens for messages
3. **Context goroutine** - Handles cancellation

## Scalability Considerations

### Current Design (Prototype)

- Single hub instance
- In-memory state
- Local message history
- No persistence

### Production Improvements

1. **Distributed Hub**
   - Use Redis Pub/Sub for message routing
   - Distributed state with etcd or Consul
   
2. **Message Persistence**
   - Store history in PostgreSQL/MongoDB
   - Implement pagination
   
3. **Agent Scaling**
   - Run agents as separate services
   - Load balance across instances
   
4. **WebSocket Scaling**
   - Use Socket.io or similar
   - Support horizontal scaling
   
5. **Rate Limiting**
   - Prevent spam
   - Manage AI API costs

## Extension Points

### Adding New Agent Types

1. Define expertise list
2. Implement factory function
3. Add to `AgentFactory`

Example:
```go
func NewMLAgent(name string, ai AIProvider, hub HubClient) *Agent {
    expertise := []string{
        "Machine Learning", "TensorFlow", "PyTorch",
        "Model Training", "Feature Engineering",
    }
    return NewAgent(protocol.AgentTypeML, name, expertise, ai, hub)
}
```

### Adding New Message Types

1. Define in `protocol/types.go`
2. Update agent response logic
3. Update UI rendering

### Custom AI Providers

Implement the `AIProvider` interface:
```go
type MyAIProvider struct { /* config */ }

func (p *MyAIProvider) GenerateResponse(ctx context.Context, 
    prompt string, history []protocol.Message) (string, error) {
    // Your implementation
}

func (p *MyAIProvider) GetModel() string {
    return "my-model"
}
```

### Adding Persistence

Replace in-memory maps with database calls:
```go
// Before
h.messages[channel] = append(h.messages[channel], msg)

// After
db.InsertMessage(ctx, msg)
```

## Security Considerations

### Current State (Prototype)

- No authentication
- No authorization
- All origins allowed for CORS
- No rate limiting

### Production Requirements

1. **Authentication**
   - JWT tokens for agents
   - API keys for CLI
   - User sessions for web UI

2. **Authorization**
   - Channel access control
   - Admin vs user permissions
   - Agent registration approval

3. **Input Validation**
   - Sanitize all inputs
   - Limit message size
   - Prevent injection attacks

4. **Rate Limiting**
   - Per agent limits
   - Per channel limits
   - API endpoint throttling

## Monitoring & Observability

### Metrics to Track

- Messages per second
- Agent response times
- AI API latency
- Error rates
- Active agents/channels

### Logging

Currently logs to stdout. Production should use:
- Structured logging (JSON)
- Log aggregation (ELK, Datadog)
- Different log levels
- Request tracing

### Health Checks

Implement:
- Liveness probe (`/health`)
- Readiness probe (`/ready`)
- Metrics endpoint (`/metrics`)

## Future Enhancements

1. **Agent Memory**
   - Long-term context retention
   - Learning from interactions
   - Personalization

2. **Agent Collaboration**
   - Explicit agent-to-agent calls
   - Subtask delegation
   - Consensus mechanisms

3. **Rich Content**
   - Code syntax highlighting
   - Diagrams (Mermaid)
   - File attachments

4. **Analytics**
   - Conversation insights
   - Agent performance metrics
   - Popular topics

5. **Integration**
   - GitHub integration
   - Slack bridge
   - IDE plugins

6. **Testing**
   - Unit tests for all components
   - Integration tests
   - Load tests
   - E2E tests

## Development Workflow

```bash
# 1. Start server
make run-server

# 2. Run tests (when implemented)
make test

# 3. Start agents
make run-agents

# 4. Build binaries
make build

# 5. Clean artifacts
make clean
```

## Deployment

### Local Development
```bash
go run cmd/server/main.go
```

### Docker (Future)
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o /server cmd/server/main.go
EXPOSE 8080
CMD ["/server"]
```

### Kubernetes (Future)
- Deploy hub as service
- Deploy agents as deployments
- Use ConfigMaps for configuration
- Use Secrets for API keys

## Contributing

When adding features:
1. Maintain the separation of concerns
2. Keep interfaces clean
3. Add tests
4. Update documentation
5. Follow Go best practices

## License

MIT

