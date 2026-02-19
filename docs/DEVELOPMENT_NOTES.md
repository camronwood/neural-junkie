# Development Notes

Internal notes for developers working on this project.

## GUI Versions Explained

The `cmd/` directory contains multiple GUI implementations. Here's why:

### Working Version ✅
- **cmd/gui-final/** - The stable, production-ready GUI
  - Event-driven architecture
  - Thread-safe with `fyne.Do()`
  - Single window with content swapping
  - **Use this one!**

### Development Iterations (Kept for reference)
- **cmd/gui/** - Original implementation (threading issues)
- **cmd/gui-fixed/** - Intermediate fix attempt
- **cmd/gui-debug/** - Debugging version with verbose logging

**Why keep them?** They document the development journey and may be useful for understanding the evolution of the solution or for teaching purposes.

**Which to use?** Always use `cmd/gui-final/` or run `make gui`.

## Code Organization

```
neural-junkie/
├── cmd/                    # Entry points
│   ├── server/             # HTTP/WebSocket server
│   ├── agent/              # Agent runner
│   ├── chat/               # Terminal chat client
│   ├── cli/                # CLI tool
│   ├── gui-final/          # Stable GUI ✅
│   └── gui*/               # Development versions (reference)
├── internal/               # Core packages
│   ├── hub/                # Message routing & state
│   ├── agent/              # Agent framework
│   ├── protocol/           # Message protocol
│   ├── ai/                 # AI provider interfaces
│   └── repo/               # Repository analysis
├── test/                   # Tests
├── scripts/                # Helper scripts
└── examples/               # Usage scenarios
```

## Key Design Decisions

### Event-Driven GUI
**Problem:** Fyne requires UI updates on main thread  
**Solution:** Event channel + `fyne.Do()` for all UI operations  
**Benefit:** Thread-safe, no race conditions

### Message Deduplication
**Problem:** Agents responding multiple times  
**Solution:** Three-layer deduplication:
1. Polling deduplication (cmd/agent/main.go)
2. Handler-level tracking (internal/agent/agent.go)
3. Agent-type filtering to prevent loops

**Benefit:** Each message processed exactly once

### Single Window Pattern
**Problem:** Creating new windows causes threading errors  
**Solution:** One window, swap content with `SetContent()`  
**Benefit:** Smooth transitions, no threading issues

## Testing Strategy

### Unit Tests
Located in `test/` directory:
- `gui_test.go` - GUI components and message flow
- `deduplication_test.go` - Message deduplication logic

### Integration Tests
Scripts in `scripts/`:
- `test-gui-final.sh` - Architecture verification
- `test-message-sending.sh` - End-to-end message flow
- `quick-test.sh` - Full system test

### Manual Testing
1. Start server: `go run cmd/server/main.go`
2. Start agents: `go run cmd/agent/main.go --type [type]`
3. Test interface: `make gui` or `make chat`

## Common Development Tasks

### Adding a New Agent Type

1. Add type to `internal/protocol/types.go`:
```go
const AgentTypeNewType AgentType = "newtype"
```

2. Add factory in `internal/agent/specialized_agents.go`:
```go
func NewNewTypeAgent(name string) *Agent {
    return NewAgent(...)
}
```

3. Update agent runner in `cmd/agent/main.go`

### Adding a New Message Type

1. Add to `internal/protocol/types.go`:
```go
const MessageTypeNewType MessageType = "new_type"
```

2. Handle in agent logic (`internal/agent/agent.go`)

3. Update UI renderers if needed

### Adding a New Interface

1. Create new cmd directory: `cmd/newinterface/`
2. Connect to hub via HTTP/WebSocket
3. Use protocol types from `internal/protocol/`
4. Add make target in `Makefile`

## Debugging Tips

### Message Flow Issues
1. Check server logs: Hub receives message?
2. Check agent logs: Agent sees message?
3. Verify deduplication: Message ID tracked?

### GUI Issues
1. Check for `fyne.Do()` wrapping UI updates
2. Verify single window pattern
3. Look for threading errors in logs

### Agent Response Issues
1. Check `shouldRespond()` logic
2. Verify expertise keywords
3. Test with mock AI first

## Performance Considerations

### Memory
- Message cache: 100 messages max per channel
- Seen messages: 100 IDs max, cleanup at 50
- Agent history: Last 10 messages for context

### Concurrency
- Hub uses RWMutex for thread-safe state
- Channels for pub/sub messaging
- Goroutines for each agent
- Event channel for GUI (buffered 100)

### Network
- WebSocket for real-time updates
- HTTP polling for agents (1s interval)
- REST API for all operations

## Deployment Notes

### Single Server (Current)
- All components run on one machine
- In-memory state (no persistence)
- Suitable for demos and development

### Production Considerations (Future)
- Add database for message persistence
- Implement Redis for distributed pub/sub
- Add load balancer for horizontal scaling
- Implement rate limiting
- Add monitoring/observability

## Environment Setup

### Development
```bash
# Install dependencies
go mod download

# Run tests
make test-all

# Start development environment
./scripts/quick-test.sh
```

### AI Configuration
```bash
# Copy environment template
cp env.example env.local

# Edit with your credentials
# Then load:
source load-env.sh
```

## Known Limitations

1. **No Persistence** - Messages lost on restart
2. **Single Server** - No distributed deployment yet
3. **No Auth** - Open access to all endpoints
4. **Limited History** - Only keeps recent messages
5. **Polling** - Agents use HTTP polling (not WebSocket)

See [FUTURE_ENHANCEMENTS.md](FUTURE_ENHANCEMENTS.md) for planned improvements.

## Troubleshooting

### "Connection refused"
Server not running. Start with `go run cmd/server/main.go`

### "No agents responding"
Check agents are running and connected to same server/channel

### GUI threading errors
Make sure using `cmd/gui-final/` not older versions

### Duplicate responses
Verify deduplication logic is present (see CHANGELOG.md fixes)

## Useful Commands

```bash
# Build everything
make build

# Run tests
make test-all

# Start GUI
make gui

# View documentation
make docs

# Clean build artifacts
make clean

# Install to PATH
make install
```

## Contributing Guidelines

1. **Test First** - Write tests before implementing features
2. **Thread Safety** - Be careful with concurrency
3. **Documentation** - Update docs with code changes
4. **Examples** - Add usage examples for new features
5. **Changelog** - Document changes in CHANGELOG.md

## Architecture Patterns

### Pub/Sub Messaging
Hub broadcasts messages to subscribed channels

### Event-Driven UI
Background threads → Event channel → Main thread

### Factory Pattern
Agent factories for different agent types

### Provider Interface
Pluggable AI providers (Mock, Claude, etc.)

### Repository Pattern
Hub manages state, provides clean API

## Code Style

- **Go fmt** - Always run `gofmt`
- **Comments** - Document public APIs
- **Error Handling** - Always check errors
- **Concurrency** - Use mutexes and channels properly
- **Testing** - Write tests for new features

## Resources

- **Fyne Docs:** https://developer.fyne.io/
- **Go Concurrency:** https://go.dev/tour/concurrency
- **WebSockets:** github.com/gorilla/websocket

---

*For user-facing documentation, see [GETTING_STARTED.md](GETTING_STARTED.md)*

