# Agent Review Feature

## Overview

The Agent Review feature allows users to request a second opinion from one agent on another agent's response, enabling collaborative multi-perspective analysis in the Neural Junkie system.

For larger, structured multi-agent work (planning + approval + execution), use the Collaboration system documented in `docs/COLLABORATION.md`.

## How It Works

### Basic Flow

1. **User asks Agent A a question:**
   ```
   User: @BackendExpert how should I structure my REST API?
   BackendExpert: You should use RESTful principles with resource-based URLs...
   ```

2. **User asks Agent B to review** (by replying to Agent A's message):
   ```
   User: @SecurityExpert thoughts on this from a security perspective?
   SecurityExpert: The approach is solid, but I'd add: ensure you implement...
   ```

### Review vs Collaboration

- **Agent Review** is lightweight and fast: one agent reviews another response in-line.
- **Collaboration** is structured: bounded agent discussion, shared plan artifact, user approval, then delegated execution.
- Use **Agent Review** for quick "second opinion" checks.
- Use **Collaboration** when multiple agents should jointly design and build.

### Key Features

- **Explicit Mentions Required**: Users must @mention the reviewing agent
- **Reply-To Context**: User must reply to the agent's message being reviewed
- **Depth Limit**: Maximum review depth of 1 (prevents cascading reviews)
- **Enhanced Prompts**: Reviewing agents receive special instructions to provide complementary insights
- **Metadata Tracking**: Review relationships are tracked via message metadata

## Usage Examples

### Example 1: Security Review

```
User: @BackendExpert how should I implement user authentication?
BackendExpert: Use JWT tokens with a 15-minute expiration...

User: @SecurityExpert agree? [replying to BackendExpert's message]
SecurityExpert: JWT is good, but I'd also recommend: adding refresh tokens, 
  implementing rate limiting on auth endpoints, and using httpOnly cookies...
```

### Example 2: Architecture Review

```
User: @DevOpsExpert how should I deploy this microservice?
DevOpsExpert: Use Docker containers with Kubernetes orchestration...

User: @BackendExpert thoughts on this approach? [replying to DevOpsExpert]
BackendExpert: Solid approach! From a backend perspective, ensure you also 
  consider service discovery, circuit breakers for resilience...
```

## Technical Implementation

### Protocol Changes

**New Message Metadata Fields:**
- `review_depth` (int): Depth in the review chain (0 = original, 1 = first review)
- `reviewed_message_id` (string): ID of the message being reviewed
- `original_question_id` (string): ID of the original user question

**New Helper Methods** (`internal/protocol/types.go`):
- `IsReviewRequest()`: Detects review trigger keywords
- `GetReviewDepth()`: Returns review depth
- `SetReviewDepth(int)`: Sets review depth
- `GetReviewedMessageID()`: Gets reviewed message ID
- `SetReviewedMessageID(string)`: Sets reviewed message ID
- `GetOriginalQuestionID()`: Gets original question ID
- `SetOriginalQuestionID(string)`: Sets original question ID

### Agent Logic Changes

**Modified `shouldRespond()` (`internal/agent/agent.go`):**
- Checks if message is replying to another agent's response
- Validates review depth (must be 0 to allow response at depth 1)
- Blocks requests that would create depth 2 or higher

**Enhanced `buildPrompt()` (`internal/agent/agent.go`):**
- Detects review context by checking ReplyTo field
- Includes the original agent's response in the prompt
- Provides special instructions for review mode:
  - Provide second opinion from your expertise
  - Highlight what the other agent got right
  - Add complementary insights
  - Note concerns or alternative approaches
  - Be constructive and collaborative

**Updated `handleMessage()` (`internal/agent/agent.go`):**
- Tracks review metadata in response messages
- Increments review depth appropriately
- Links to reviewed message and original question

### Safety Guardrails

1. **Max Review Depth**: Only allows 1 level of review
   - User → Agent A → User → Agent B ✅
   - User → Agent A → User → Agent B → User → Agent C ❌

2. **Explicit Mentions Only**: Review agent MUST be mentioned by user

3. **Reply-To Required**: User message must reply to the agent's message being reviewed

4. **No Agent-Only Chains**: Original question must be from human user

## Review Keywords

The system detects these phrases as review requests (case-insensitive):
- "thoughts?"
- "what do you think?"
- "agree?" / "disagree?"
- "review this"
- "opinion?"
- "your take?"
- "perspective?"
- "thoughts on this?"
- "makes sense?"
- "sound right?"

**Note**: Keywords are optional if the message has proper @mention + ReplyTo structure.

## Testing

Comprehensive test suite in `/test/agent_review_test.go`:

- **TestAgentReview**: Basic review functionality
- **TestReviewDepthLimit**: Ensures reviews don't cascade beyond depth 1
- **TestReviewWithoutReplyTo**: Validates normal vs review behavior
- **TestReviewMetadataTracking**: Tests metadata helpers
- **TestIsReviewRequest**: Tests keyword detection

All tests passing as of October 2025.

## Benefits

### For Users
- Get multiple expert perspectives on complex problems
- Identify gaps or concerns in initial responses
- Benefit from collaborative AI expertise
- Natural conversation flow

### For the System
- Encourages agent specialization
- Prevents infinite agent-to-agent loops
- Maintains conversation context
- Tracks review relationships

## Limitations

1. **Single Review Level**: Can't chain more than 2 agents on same question (by design)
2. **Requires Reply-To**: UI must support replying to specific messages
3. **No Auto-Review**: Users must explicitly request reviews
4. **Review Context**: Only includes the reviewed message, not full thread

## Future Enhancements

Potential improvements (see `docs/FUTURE_ENHANCEMENTS.md`):
- Allow deeper review chains (configurable depth limit)
- Auto-suggest reviewers based on content
- Review summaries combining multiple agent perspectives
- Visual review trees in UI
- Consensus detection across multiple reviews

For currently implemented consensus and bounded multi-agent discussion, see `docs/COLLABORATION.md`.

## Related Documentation

- `docs/ARCHITECTURE.md` - Overall system architecture
- `docs/COLLABORATION.md` - Multi-agent planning/execution workflow
- `docs/GETTING_STARTED.md` - Setup and usage guide
- `internal/protocol/types.go` - Protocol implementation
- `internal/agent/agent.go` - Agent response logic

---

**Implementation Date**: October 2025  
**Status**: ✅ Fully implemented and tested  
**Version**: 1.0

