# Scenario 2: Architecture Decision - Authentication System

This example shows how agents collaborate on architectural decisions.

## Setup

Start a full team of agents for comprehensive architectural discussion:

```bash
# Terminal 1: Server
make run-server

# Terminal 2-6: Agents
go run cmd/agent/main.go --type frontend --name "UI Architect"
go run cmd/agent/main.go --type backend --name "API Architect"
go run cmd/agent/main.go --type security --name "Security Lead"
go run cmd/agent/main.go --type database --name "Data Architect"
go run cmd/agent/main.go --type devops --name "Cloud Engineer"
```

## Scenario

You're building a new application and need to decide on an authentication strategy.

### Step 1: Initial Question

```bash
go run cmd/cli/main.go --channel general \
  --message "We're building a new web app with mobile clients. How should we implement authentication? JWT, sessions, or something else?"
```

### Expected Multi-Agent Discussion

**Security Agent**:
- "For security, I recommend JWT with refresh token rotation. Use RS256 (not HS256), short-lived access tokens (15 min), and httpOnly cookies for refresh tokens. This prevents XSS attacks."

**Backend Agent**:
- "For the API, JWT works well for stateless authentication. Implement token validation middleware. Store refresh tokens in the database with user association for revocation capability."

**Frontend Agent**:
- "From the UI perspective, store access tokens in memory (not localStorage due to XSS risk). Implement automatic token refresh before expiration. Handle 401 responses by redirecting to login."

**Database Agent**:
- "Create a refresh_tokens table with: id, user_id, token_hash, expires_at, created_at. Index on token_hash and user_id. Implement cleanup job for expired tokens."

**DevOps Agent**:
- "Consider using an identity provider like Auth0 or AWS Cognito to offload complexity. If self-hosting, ensure rate limiting on auth endpoints and monitor for brute force attempts."

### Step 2: Follow-up on Specific Concern

```bash
go run cmd/cli/main.go --channel general \
  --message "What about social login (Google, GitHub)? How does that fit in?"
```

### Expected Responses

**Security Agent**:
- "Use OAuth 2.0 flow for social login. Validate tokens server-side, never trust client-provided tokens. Store minimal user data from social providers."

**Backend Agent**:
- "Implement a unified user model. Link social accounts to your users table via a user_social_accounts junction table. Allow multiple social logins per user."

**Frontend Agent**:
- "Show social login buttons prominently. Handle OAuth redirect flow cleanly. Consider using a library like next-auth or passport.js."

## Creating a Project-Specific Channel

For focused architectural discussions:

```bash
# Create dedicated channel
go run cmd/cli/main.go --create "auth-architecture"

# Have agents join
go run cmd/agent/main.go --type security --channel "auth-architecture"
go run cmd/agent/main.go --type backend --channel "auth-architecture"

# Continue discussion there
go run cmd/cli/main.go --channel "auth-architecture" \
  --message "Let's finalize the auth architecture here"
```

## Key Learnings

1. **Comprehensive Coverage**: Each agent covers their domain
2. **Security First**: Security agent leads on security-critical features
3. **Practical Trade-offs**: Agents discuss both DIY and managed solutions
4. **Implementation Details**: Agents provide specific technical guidance

## Exporting the Discussion

The conversation becomes living documentation:

```bash
# Get all messages for documentation
go run cmd/cli/main.go --list messages --channel auth-architecture > docs/auth-decisions.txt
```

