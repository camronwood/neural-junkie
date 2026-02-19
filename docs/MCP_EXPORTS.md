# MCP Agent Exports

This guide explains how to export Neural Junkie agents to MCP (Model Context Protocol) format, making their knowledge portable and reusable in any MCP-compatible tool.

## Overview

The MCP export system allows you to:

- **Export repo agents** - Convert repository experts to portable MCP resources
- **Export helper agents** - Package knowledge bases as MCP prompts
- **Share expertise** - Distribute agent knowledge as files
- **Recreate agents** - Use exports to recreate experts elsewhere
- **Integrate with tools** - Use exports in Claude Desktop, IDEs, etc.

**Status: ✅ ENABLED** - All export/import functionality is now working.

## Quick Start

### Export an Agent

```bash
# Export a specific agent
/export-agent-mcp MyProject Expert

# List all exports
/list-exports

# Export all agents at once
/export-all-agents
```

### Import an Agent

```bash
# Import from file
/import-agent-mcp /path/to/export.json

# Delete an export
/delete-export MyProject Expert
```

## Export Format

MCP exports are JSON files containing:

- **Agent metadata** - Name, type, expertise, description
- **Resources** - Code files, documentation, architecture
- **Prompts** - Pre-configured prompt templates
- **System prompt** - Complete agent personality and instructions

### Example Export Structure

```json
{
  "version": "1.0",
  "agent": {
    "name": "MyProject Expert",
    "type": "repo",
    "expertise": ["React", "TypeScript", "Node.js"],
    "description": "Repository expert for MyProject",
    "createdAt": "2025-10-16T10:30:00Z",
    "repository": "/path/to/project"
  },
  "resources": [
    {
      "uri": "repo://architecture",
      "name": "Architecture Overview",
      "mimeType": "text/markdown",
      "content": "# Project Architecture\n\n..."
    },
    {
      "uri": "repo://files/src/main.ts",
      "name": "Main Entry Point",
      "mimeType": "text/typescript",
      "content": "import React from 'react';\n..."
    }
  ],
  "prompts": [
    {
      "name": "analyze_architecture",
      "description": "Analyze the architecture of the repository",
      "prompt": "Using the architecture documentation at {{repo://architecture}}, analyze..."
    }
  ],
  "systemPrompt": "You are MyProject Expert, a repository expert agent..."
}
```

## Repository Agent Exports

Repository agents export their complete knowledge of a codebase:

### Resources Exported

- **Architecture documentation** - Generated overview of the system
- **Source files** - Important code files (limited to prevent huge exports)
- **Key files** - README, package.json, config files
- **Code patterns** - Identified frameworks and patterns
- **Dependencies** - Project dependencies and relationships
- **Git information** - Recent commits and history

### Prompt Templates

Repository exports include these prompt templates:

- `analyze_architecture` - Analyze system architecture
- `explain_code` - Explain specific code functionality
- `find_files` - Find files related to a topic
- `get_dependencies` - Get dependency information
- `suggest_improvements` - Suggest code improvements

### Usage Example

```bash
# Create a repo agent
/create-repo-agent /path/to/project "MyProject Expert"

# Wait for indexing to complete, then export
/export-agent-mcp MyProject Expert

# The export is saved to ~/.neural-junkie/exports/repo/myproject_expert.json
```

## Helper Agent Exports

Helper agents export their knowledge base and configuration:

### Resources Exported

- **Agent configuration** - Name, expertise, keywords, system prompt
- **Knowledge documents** - All markdown/text files in knowledge base
- **Expertise areas** - List of expertise domains
- **Keywords** - Trigger keywords for responses
- **Knowledge index** - Topics and headings for quick reference

### Prompt Templates

Helper exports include these prompt templates:

- `ask_question` - Ask a question to the helper
- `get_guidance` - Get guidance on a specific topic
- `find_topic` - Find information about a topic
- `suggest_resources` - Suggest relevant resources

### Usage Example

```bash
# Create a helper agent
/create-helper day-one "Day One Expert" "Helps new engineers get started"

# Export the helper
/export-agent-mcp Day One Expert

# The export is saved to ~/.neural-junkie/exports/helper/day_one_expert.json
```

## MCP Resource Server

The MCP resource server exposes exported agents via the Model Context Protocol:

### Starting the Server

```bash
# Start the resource server (if enabled in env.local)
ENABLE_MCP_RESOURCES=true make server
```

### Available Tools

The resource server provides these MCP tools:

- `list_exported_agents` - List all available exports
- `get_agent_resource` - Get a specific resource from an export
- `get_agent_prompt` - Get a pre-configured prompt
- `recreate_agent` - Get instructions to recreate an agent
- `get_agent_info` - Get detailed information about an export
- `search_agents` - Search agents by expertise or keywords

### Integration with Claude Desktop

1. **Export your agents** using `/export-agent-mcp`
2. **Start the resource server** (if not already running)
3. **Configure Claude Desktop** to connect to the resource server
4. **Use the tools** to access agent knowledge in Claude Desktop

## CLI Commands

The CLI tool supports export operations:

```bash
# Export a repo agent
neural-junkie export repo-agent --name="MyProject Expert" --output=export.json

# Export a helper agent
neural-junkie export helper-agent --name="Day One Expert" --output=export.json

# List all exports
neural-junkie list-exports

# Start MCP resource server
neural-junkie serve-mcp-resources --port=8086
```

## Environment Configuration

Add these settings to your `env.local`:

```bash
# MCP Resource Server
ENABLE_MCP_RESOURCES=true
MCP_RESOURCES_PORT=8086

# Export Storage
MCP_EXPORTS_DIR=~/.neural-junkie/exports
```

## File Structure

Exports are stored in:

```
~/.neural-junkie/exports/
├── repo/
│   ├── myproject_expert.json
│   └── backend_expert.json
└── helper/
    ├── day_one_expert.json
    └── testing_expert.json
```

## Use Cases

### 1. Team Knowledge Sharing

Export your team's repository experts and share them:

```bash
# Export all agents
/export-all-agents

# Share the export files with your team
# They can import them with /import-agent-mcp
```

### 2. Backup Agent Knowledge

Before deleting agents, export their knowledge:

```bash
# Export before deletion
/export-agent-mcp Important Expert

# Delete the agent
/delete-agent Important Expert

# Restore later if needed
/import-agent-mcp /path/to/backup.json
```

### 3. Claude Desktop Integration

Use exported agents in Claude Desktop:

1. Export your agents
2. Start the MCP resource server
3. Configure Claude Desktop to use the resource server
4. Access agent knowledge through MCP tools

### 4. IDE Integration

Use exports in your IDE with MCP support:

1. Export repository experts
2. Configure your IDE's MCP client
3. Access code knowledge and architecture through MCP

## Best Practices

### Export Management

- **Regular exports** - Export important agents regularly
- **Version control** - Track export files in git for team sharing
- **Cleanup** - Delete old exports to save space
- **Naming** - Use descriptive names for exports

### Performance

- **File size** - Large repositories may create large exports
- **Resource limits** - Only important files are included in exports
- **Compression** - Exports use compressed content for efficiency

### Security

- **Sensitive data** - Be careful exporting agents with sensitive information
- **Access control** - Export files may contain proprietary code
- **Sharing** - Only share exports with trusted parties

## Troubleshooting

### Export Fails

```bash
# Check if agent exists
/list-agents

# Ensure agent is fully indexed (for repo agents)
# Wait for indexing to complete before exporting
```

### Import Fails

```bash
# Check file path and permissions
ls -la /path/to/export.json

# Validate export format
cat /path/to/export.json | jq .
```

### Resource Server Issues

```bash
# Check if server is running
lsof -i :8086

# Check environment variables
echo $ENABLE_MCP_RESOURCES
echo $MCP_RESOURCES_PORT
```

## Advanced Usage

### Custom Export Paths

```bash
# Export to specific location
/export-agent-mcp MyAgent
# Then copy from ~/.neural-junkie/exports/ to desired location
```

### Batch Operations

```bash
# Export all agents
/export-all-agents

# List all exports
/list-exports

# Delete multiple exports
/delete-export Agent1
/delete-export Agent2
```

### Integration Scripts

Create scripts to automate export/import:

```bash
#!/bin/bash
# Export all agents daily
/export-all-agents
# Backup to cloud storage
rsync -av ~/.neural-junkie/exports/ backup-server:/backups/
```

## Future Enhancements

Planned improvements include:

- **Compressed exports** - Reduce file sizes for large repositories
- **Incremental exports** - Only export changes since last export
- **Cloud storage** - Direct integration with cloud storage services
- **Team collaboration** - Real-time sharing of agent knowledge
- **Export templates** - Pre-configured export formats for common use cases

## Support

For issues with MCP exports:

1. Check the logs: `tail -f /tmp/chat-server.log`
2. Verify environment configuration
3. Ensure agents are properly indexed
4. Check file permissions for export directory

The MCP export system makes your AI agents' knowledge truly portable and reusable across different tools and environments.
