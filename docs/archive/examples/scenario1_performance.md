# Scenario 1: Debugging a Performance Issue

This example demonstrates how multiple agents collaborate to solve a performance problem.

## Setup

1. Start the server:
```bash
make run-server
```

2. Start the agents:
```bash
# Terminal 2
go run cmd/agent/main.go --type backend --name "Backend Expert"

# Terminal 3
go run cmd/agent/main.go --type database --name "DB Optimizer"

# Terminal 4
go run cmd/agent/main.go --type devops --name "Performance Engineer"
```

## Scenario

A user reports that their API endpoint is slow when fetching user data.

### Step 1: Ask the Question

```bash
go run cmd/cli/main.go --channel general \
  --message "Our /api/users endpoint is taking 5+ seconds to respond. We have about 10,000 users. What could be causing this?"
```

### Expected Collaboration

**Backend Agent** might respond:
- "Let me check common backend issues. Could be N+1 queries, missing pagination, or unoptimized data fetching. @DB Optimizer, can you check the query patterns?"

**Database Agent** might respond:
- "I see potential N+1 query issues. Are you fetching related data for each user in separate queries? Also, check if you have indexes on frequently queried columns. Can you share your query?"

**DevOps Agent** might respond:
- "From an infrastructure perspective, check your server metrics. Are you seeing high CPU or memory usage? Also consider adding caching with Redis to reduce database load."

### Step 2: Provide More Context

```bash
go run cmd/cli/main.go --channel general \
  --message "Yes, we're fetching user posts and comments for each user. Here's the query: SELECT * FROM users; then for each user: SELECT * FROM posts WHERE user_id = ?"
```

### Expected Follow-up

**Database Agent**:
- "That's a classic N+1 query problem! Solution: Use a JOIN query instead: SELECT users.*, posts.* FROM users LEFT JOIN posts ON users.id = posts.user_id. Also add an index on posts.user_id."

**Backend Agent**:
- "Additionally, implement pagination. Don't fetch all 10,000 users at once. Return 20-50 per page with offset/limit."

**DevOps Agent**:
- "Once you fix the queries, add caching for frequently accessed user data. Set TTL to 5-10 minutes. This will dramatically reduce database load."

## Key Learnings

1. **Specialized Expertise**: Each agent brings domain-specific knowledge
2. **Cross-References**: Agents mention each other when their expertise is needed
3. **Collaborative Problem Solving**: The solution emerges from combined insights
4. **Iterative Refinement**: Agents build on each other's suggestions

## Monitoring the Conversation

Watch the conversation in real-time:
```bash
go run cmd/cli/main.go --watch --channel general
```

Or view it in the web UI:
```
http://localhost:18765
```

