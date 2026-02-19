# Changelog

All notable changes and fixes to the Neural Junkie project.

## [Unreleased]

See [FUTURE_ENHANCEMENTS.md](FUTURE_ENHANCEMENTS.md) for planned features.

## [1.0.0] - 2025-10-14

### Added
- ✨ **GUI Application** - Fyne-based desktop interface with event-driven architecture
- ✨ **Terminal Chat** - Interactive CLI chat client
- ✨ **Repository Expert Agents** - AI agents that analyze and become experts on codebases
- ✨ **Multi-Interface Support** - GUI, Terminal, Web, and CLI all working together
- 🔧 **AI Hub Integration** - Support for AI Hub endpoint in addition to direct Anthropic API
- 📊 **Real-time Updates** - WebSocket-based live message broadcasting
- 🤖 **Specialized Agents** - Frontend, Backend, DevOps, Database, Security agent types
- 🔄 **Channel System** - Multi-channel conversation support
- 📝 **Message History** - Conversation history with context

### Fixed
- 🐛 **Message Deduplication** - Agents no longer respond multiple times to same message
  - Added subscribe polling deduplication with `seenMessages` map
  - Added handler-level deduplication with `respondedMessages` tracking
  - Added agent-type filtering to prevent agent-to-agent response loops
  
- 🐛 **GUI Threading Issues** - Eliminated Fyne threading errors
  - Implemented event-driven architecture with single UI thread
  - All UI updates now use `fyne.Do()` for thread safety
  - Changed from multi-window to single-window with content swapping
  - Background goroutines communicate via event channel
  
- 🐛 **Username Display** - Messages now show actual username instead of "Human User"
  - Server now accepts `from` field in send message requests
  - Username properly propagated through message flow

### Technical Improvements
- 🏗️ **Thread-Safe Architecture** - Event channel with proper synchronization
- 🔒 **Race Condition Prevention** - Proper use of mutexes and channels
- 🧹 **Memory Management** - Built-in cleanup for seen messages and caches
- 📦 **Code Organization** - Clear separation of concerns across packages
- 🧪 **Test Coverage** - Unit tests, integration tests, architecture tests

### Documentation
- 📚 Comprehensive README with feature overview
- 📚 GETTING_STARTED guide for quick setup
- 📚 ARCHITECTURE document with technical deep-dive
- 📚 REPO_AGENTS documentation for repository analysis feature
- 📚 Usage examples in examples/ directory

## Development Journey

### Phase 1: Core System
Built the foundation with Hub, Agent framework, and Protocol definitions.

### Phase 2: Interfaces
Implemented CLI, Web UI, Terminal chat, and GUI interfaces.

### Phase 3: Debugging & Stabilization
- Tackled message deduplication issues across three layers
- Resolved GUI threading problems with event-driven pattern
- Fixed username propagation through message flow

### Phase 4: Polish & Documentation
- Cleaned up temporary debugging files
- Consolidated documentation
- Created comprehensive getting started guide

## Breaking Changes

None - this is the initial stable release.

## Migration Guide

N/A - First stable version.

## Contributors

Built with Go, Fyne, WebSockets, and a lot of debugging! 🚀

## Version History

- **1.0.0** (2025-10-14) - Initial stable release
  - All core features working
  - All interfaces functional
  - Thread-safe and stable
  - Production-ready prototype

---

For detailed status, see [STATUS.md](STATUS.md)  
For upcoming features, see [FUTURE_ENHANCEMENTS.md](FUTURE_ENHANCEMENTS.md)

