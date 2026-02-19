# Confluence Agents - Documentation Expert Agents

Confluence agents are specialized AI agents that become experts on Confluence Cloud documentation spaces. They index, cache, and search through your Confluence pages and comments to provide informed answers about your documentation.

## Overview

Confluence agents follow the same pattern as repository agents, providing:
- **Deep Indexing**: Analyze all pages and comments in a Confluence space
- **Persistent Caching**: Store indexed data locally for fast reloading
- **Intelligent Search**: Full-text search across pages and comments with relevance scoring
- **Staleness Detection**: Automatic detection of changed pages
- **Progress Tracking**: Real-time progress updates during indexing (0-100%)

## Features

### 📚 **Content Indexing**
- Indexes all pages in a Confluence space
- Extracts text content from HTML storage format
- Includes page metadata (labels, authors, timestamps)
- Indexes comments on pages
- Maintains page hierarchy and relationships

### 💾 **Persistent Caching**
- Stores indexes at `~/.neural-junkie/confluence/<space-key>/`
- Compressed storage using gzip
- Fast loading from cached indexes (<2 seconds)
- Automatic staleness detection via page modification times
- Incremental reindexing of changed pages only

### 🔍 **Intelligent Search**
- Full-text search across page content
- Search by page title
- Search by labels
- Search within comments
- Relevance scoring based on:
  - Title matches (highest weight)
  - Content matches
  - Label matches
  - Recency (recent updates score higher)

### 🎯 **Smart Responses**
- Responds when mentioned directly
- Responds to questions about documentation
- Provides relevant page snippets with URLs
- Cites sources from Confluence pages
- Indicates when information isn't found in docs

## Setup

### 1. Get Confluence API Token

Confluence agents use the Confluence Cloud REST API with email + API token authentication. This works even with SSO-enabled accounts.

**Steps to create an API token:**

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Give it a descriptive name (e.g., "Neural Junkie")
4. Copy the generated token immediately (you won't be able to see it again)

### 2. Configure Environment Variables

Add the following to your `env.local` file:

```bash
# Confluence Cloud Configuration
CONFLUENCE_DOMAIN=yourcompany.atlassian.net
CONFLUENCE_EMAIL=your.email@company.com
CONFLUENCE_API_TOKEN=your_confluence_api_token_here
```

**Important Notes:**
- `CONFLUENCE_DOMAIN`: Your Confluence Cloud domain (without https://)
- `CONFLUENCE_EMAIL`: The email address associated with your Atlassian account
- `CONFLUENCE_API_TOKEN`: The API token you created above

### 3. Load Environment

```bash
source load-env.sh
```

## Usage

### Creating a Confluence Agent

Create an agent for a specific Confluence space:

```
/create-confluence-agent <space-key> [agent-name]
```

**Examples:**
```
/create-confluence-agent TECH
/create-confluence-agent TECH Technical Documentation Expert
/create-confluence-agent PRODUCT Product Docs
```

The agent will:
1. Fetch space information
2. Index all pages in the space
3. Extract content and comments
4. Build search indexes
5. Save to persistent cache
6. Report progress throughout (0-100%)

**First indexing** takes 30-60 seconds for typical spaces (depends on page count).

**Subsequent loads** are <2 seconds from cache.

### Chat Commands

#### `/list-confluence-agents`
Lists all active Confluence agents with their status and statistics.

**Example output:**
```
**Confluence Agents:**

• **Tech Docs Expert**
  Space: TECH
  Status: ✅ Ready
  Stats: 45 pages

• **Product Documentation**
  Space: PRODUCT
  Status: 🔄 Indexing (60%)
  Stats: Not indexed yet

**Commands:**
• /reindex-confluence-agent <name> - Reindex a space
• /pause-agent <name> - Pause an agent
• /unpause-agent <name> - Unpause an agent
• /delete-agent <name> - Delete an agent
```

#### `/reindex-confluence-agent <name>`
Triggers a manual reindex of a Confluence space.

**Example:**
```
/reindex-confluence-agent Tech Docs Expert
```

The agent will:
1. Check for changed pages
2. Incrementally update only changed content
3. Full reindex if major changes detected
4. Update persistent cache

#### `/pause-agent <name>`
Pauses a Confluence agent (stops responding to messages).

**Example:**
```
/pause-agent Tech Docs Expert
```

#### `/unpause-agent <name>`
Resumes a paused Confluence agent.

**Example:**
```
/unpause-agent Tech Docs Expert
```

#### `/delete-agent <name>`
Deletes a Confluence agent (cached data remains for future use).

**Example:**
```
/delete-agent Tech Docs Expert
```

### Asking Questions

Once an agent is ready, you can ask questions about the documentation:

**Examples:**
```
@Tech Docs Expert what's our API authentication process?

How do I configure SSL in the application?

What are the deployment steps for production?
```

The agent will:
1. Search the indexed Confluence space
2. Find relevant pages and snippets
3. Provide an answer with citations
4. Include URLs to source pages

## Architecture

### Component Structure

```
internal/confluence/
├── client.go      # Confluence REST API client
├── index.go       # Index data structures
├── storage.go     # Persistent caching
├── analyzer.go    # Content indexing & analysis
└── search.go      # Search functionality

internal/agent/
└── confluence_agent.go  # Confluence agent implementation
```

### Data Flow

1. **Indexing**: `Analyzer` → `Client` (fetch) → `Index` (build) → `Storage` (save)
2. **Loading**: `Storage` (load) → `Index` → staleness check → incremental update
3. **Searching**: Query → `Searcher` → relevance scoring → results with snippets
4. **Responding**: Message → search → AI context → response with citations

### Index Structure

```go
type ConfluenceIndex struct {
    SpaceKey     string
    SpaceName    string
    LastIndexed  time.Time
    PageCount    int
    Pages        map[string]*Page      // pageID -> Page
    Hierarchy    map[string][]string   // parentID -> childIDs
    Labels       map[string][]string   // label -> pageIDs
    LastModified map[string]time.Time  // pageID -> timestamp
}

type Page struct {
    ID          string
    Title       string
    Content     string        // Extracted text
    Labels      []string
    Comments    []PageComment
    Author      string
    LastUpdated time.Time
    ParentID    string
    URL         string
}
```

## Performance

### Indexing Performance
- **First indexing**: 30-60 seconds for typical spaces (100-200 pages)
- **Cached loading**: <2 seconds from persistent storage
- **Staleness check**: <1 second (checks page modification times)
- **Incremental update**: Proportional to number of changed pages

### Storage & Limits
- **Storage location**: `~/.neural-junkie/confluence/<space-key>/`
- **Index size**: ~1-5MB per space (compressed)
- **Max content size**: 10MB per page
- **Max total size**: 500MB per space

### API Rate Limiting
- **Built-in rate limiting**: 200ms between API calls
- **Confluence API limits**: ~100 requests/minute
- **Automatic retries**: 3 retries with exponential backoff
- **Large spaces**: May take several minutes on first index

## Troubleshooting

### "Missing Confluence credentials"
**Problem**: Environment variables not set.
**Solution**:
```bash
# Add to env.local
CONFLUENCE_DOMAIN=yourcompany.atlassian.net
CONFLUENCE_EMAIL=your.email@company.com
CONFLUENCE_API_TOKEN=your_token

# Reload
source load-env.sh
```

### "gh CLI not installed or not authenticated"
**Problem**: This error appears when trying GitHub commands, not Confluence.
**Solution**: GitHub integration is separate. See GitHub CLI setup.

### "API error 401: Unauthorized"
**Problem**: Invalid credentials or expired API token.
**Solution**:
- Verify `CONFLUENCE_EMAIL` matches your Atlassian account
- Regenerate API token at https://id.atlassian.com/manage-profile/security/api-tokens
- Update `CONFLUENCE_API_TOKEN` in env.local

### "API error 404: Space not found"
**Problem**: Space key doesn't exist or you don't have access.
**Solution**:
- Verify space key is correct (case-sensitive)
- Check you have access to the space in Confluence
- Space must be accessible with your API token credentials

### "Space too large: total content exceeds limit"
**Problem**: Space exceeds 500MB total content size.
**Solution**:
- Index smaller spaces or specific sections
- Consider archiving old/unused pages
- Contact support for increased limits

### Slow indexing
**Problem**: Indexing takes very long.
**Reasons**:
- Large space with many pages
- Confluence API rate limiting
- Network latency

**Solutions**:
- First indexing is always slower (30-60s for typical spaces)
- Subsequent loads are fast (<2s from cache)
- Be patient with large spaces (500+ pages may take 5-10 minutes)

### Stale data
**Problem**: Agent answers don't reflect recent page updates.
**Solution**:
```
/reindex-confluence-agent <name>
```

Agents automatically check for staleness on load, but manual reindex ensures fresh data.

## Best Practices

### Space Organization
- Create separate agents for different documentation spaces
- Use clear, descriptive agent names
- Match agent expertise to space content

### Question Patterns
- Ask specific questions about documented topics
- Mention the agent by name for direct responses
- Include context about what you're trying to accomplish

### Maintenance
- Reindex after major documentation updates
- Monitor indexing progress during first-time setup
- Delete unused agents to free resources

### Security
- Keep API tokens secure (never commit to git)
- Use environment variables for credentials
- Tokens work with SSO-enabled accounts
- Tokens can be revoked at any time

## Limitations

### Current Limitations
1. **Cloud only**: Works with Confluence Cloud, not Server/Data Center
2. **No writes**: Agents are read-only (can't create/edit pages)
3. **No attachments**: Doesn't index file attachments
4. **HTML only**: Extracts text from HTML storage format
5. **Rate limits**: Respects Confluence API rate limits (~100 req/min)

### Future Enhancements
- Support for Confluence Server/Data Center
- Attachment content indexing (text extraction from PDFs, docs)
- Multi-space agents
- Advanced filtering (by date, author, label)
- Page version history analysis
- Real-time updates via webhooks

## Integration with Other Features

### Works With
- **Repository Agents**: Different agents for code vs docs
- **Helper Agents**: Custom knowledge bases alongside Confluence
- **Dispatch System**: Execute commands based on documentation
- **Multi-channel**: Same agent can serve multiple channels

### Example Workflow
```
# Create agents for code and docs
/create-repo-agent ~/projects/myapp MyApp Code Expert
/create-confluence-agent MYAPP MyApp Docs Expert

# Ask questions to both
@MyApp Code Expert how is authentication implemented?
@MyApp Docs Expert what's the authentication flow in the docs?

# Compare code implementation with documentation
```

## API Token Security

### Token Best Practices
1. **Create dedicated tokens** for Neural Junkie (easier to revoke)
2. **Name tokens clearly** (e.g., "Neural Junkie - Dev Machine")
3. **Rotate tokens periodically** (e.g., every 90 days)
4. **Revoke unused tokens** immediately
5. **Never share tokens** or commit to version control

### Token Permissions
- API tokens inherit your user permissions
- Agents can only access spaces you can access
- Consider creating a dedicated service account for production use
- Service accounts can have specific space permissions

## Examples

### Example 1: Technical Documentation
```
/create-confluence-agent TECH Technical Documentation

# After indexing completes...
@Technical Documentation what's our microservices architecture?
@Technical Documentation how do we handle database migrations?
@Technical Documentation where are the API endpoints documented?
```

### Example 2: Product Documentation
```
/create-confluence-agent PRODUCT Product Docs

# Product questions
@Product Docs what are the new features in version 2.0?
@Product Docs how does the user onboarding flow work?
@Product Docs what's the pricing model for enterprise customers?
```

### Example 3: Team Knowledge Base
```
/create-confluence-agent TEAM Team Wiki

# Team processes
@Team Wiki what's the on-call rotation schedule?
@Team Wiki how do I request PTO?
@Team Wiki where is the incident response runbook?
```

## Conclusion

Confluence agents bring your documentation into Neural Junkie, making it easy to:
- **Find information** quickly without manual searching
- **Get contextual answers** with citations to source pages
- **Stay current** with automatic staleness detection
- **Scale knowledge** across your entire team

For questions or issues, see the main [README.md](../README.md) or check [TROUBLESHOOTING](../README.md#troubleshooting).

