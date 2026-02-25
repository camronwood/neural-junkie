# Dispatch CLI Integration

The AI Chat Room integrates with the [Dispatch CLI](https://github.com/dispatchit/dispatch-cli), allowing users and AI agents to execute DevOps commands directly from the chat interface.

## Overview

The Dispatch integration enables:

- ✅ **Execute DevOps commands** from chat without switching contexts
- 🔒 **Security controls** - Read-only commands run immediately, write operations require approval
- 🤖 **AI assistance** - DevOps agents suggest relevant dispatch commands
- 📊 **Formatted output** - Command results displayed with syntax highlighting and collapsible views
- ⏱️ **Timeout handling** - Commands are automatically terminated if they run too long

## Prerequisites

### Install Dispatch CLI

The Dispatch CLI must be installed and available in your `PATH`. To check if it's installed:

```bash
which dispatch
# Should output: /usr/local/bin/dispatch (or similar)

dispatch version
# Should show version information
```

If not installed, contact your DevOps team or visit the Dispatch CLI installation instructions.

### Configure Dispatch

Ensure your Dispatch CLI is configured with appropriate credentials:

```bash
# AWS configuration (if using AWS operations)
dispatch aws login

# Docker configuration (if using container registries)
dispatch docker login

# Kubernetes context (if using cluster operations)
dispatch kctx
```

## Available Commands

### Subenvironment Management (`subenv`)

**Read-only commands (execute immediately):**
- `/dispatch subenv list` - List active subenvironments
- `/dispatch subenv awake` - Check if a subenvironment is awake
- `/dispatch subenv context` - Show subenvironment context settings
- `/dispatch subenv open` - Open subenvironment service in browser

**Write commands (require approval):**
- `/dispatch subenv create <name>` - Create a new subenvironment
- `/dispatch subenv destroy <name>` - Destroy a subenvironment
- `/dispatch subenv deploy <branch>` - Deploy branch to subenvironment
- `/dispatch subenv wake` - Wake up sleeping subenvironment
- `/dispatch subenv prune` - Remove outdated resources
- `/dispatch subenv update-kubeconfig` - Update kubeconfig for cluster

### AWS Operations (`aws`)

**All AWS commands require approval:**
- `/dispatch aws login` - Login to AWS CLI SSO session
- `/dispatch aws logout` - Logout of AWS CLI SSO session
- `/dispatch aws setup` - Setup AWS CLI environment

### Docker Registry (`docker`)

**All Docker commands require approval:**
- `/dispatch docker login` - Login to container registries
- `/dispatch docker logout` - Logout of container registries

### Kubernetes Context (`kctx`)

**Read-only:**
- `/dispatch kctx` - Switch/view Kubernetes cluster context

### Secrets Management (`sops`)

**Read-only:**
- `/dispatch sops view <file>` - View encrypted files

**Require approval:**
- `/dispatch sops encrypt <file>` - Encrypt files
- `/dispatch sops decrypt <file>` - Decrypt files

### Workstation Management (`workstation`)

**Read-only:**
- `/dispatch workstation config` - View workstation configuration
- `/dispatch workstation shellenv` - Generate environment file

**Require approval:**
- `/dispatch workstation sync` - Run workstation sync commands

### Plugin Management (`plugin`)

**Read-only:**
- `/dispatch plugin list` - List installed plugins

**Require approval:**
- `/dispatch plugin install <name>` - Install a plugin

## Command Execution Flow

### Read-Only Commands (Immediate Execution)

1. User types `/dispatch subenv list`
2. System validates command is read-only
3. Command executes immediately
4. Results displayed in chat with formatted output

```
User: /dispatch subenv list
System: ✅ Command executed successfully (1.2s)
```

### Write Commands (Approval Required)

1. User types `/dispatch subenv create test-env`
2. System creates approval request with unique ID
3. User approves with `/approve <id>`
4. Command executes and results displayed

```
User: /dispatch subenv create test-env

System: 🔒 Approval Required
User Test User wants to execute:
dispatch subenv create test-env

⚠️  This command requires approval because it can modify system state.

To approve: /approve abc123
To reject: /reject abc123

*Request expires in 5 minutes (14:35:00)*

User: /approve abc123

System: ✅ Approved - Executing command:
dispatch subenv create test-env

System: ✅ Command executed successfully (5.3s)
[output shown here]
```

## Usage Examples

### Check Active Subenvironments

```
/dispatch subenv list
```

### Create New Subenvironment (with approval)

```
/dispatch subenv create my-feature-env
# System responds with approval request
/approve abc123
# Command executes after approval
```

### View Encrypted Secrets

```
/dispatch sops view config/secrets.yaml
```

### Deploy to Subenvironment

```
/dispatch subenv deploy feature-branch
# Requires approval
/approve xyz789
```

### List All Available Commands

```
/dispatch-list
```

Shows all available dispatch commands organized by plugin, with indicators for read-only vs. approval-required.

## AI Agent Assistance

The DevOps AI agent can suggest relevant dispatch commands based on your questions:

```
User: @devops How can I check my active subenvironments?

DevOps Agent: You can list your active subenvironments with:
`/dispatch subenv list`

This will show all currently running subenvironments along with their status.
```

The DevOps agent recognizes dispatch-related keywords:
- subenv, subenvironment
- kubectl, cluster, context
- deploy, deployment
- secrets, sops
- aws login, docker login

## Command Output Display

Command outputs are formatted for readability:

### Successful Command

```
✅ Command executed successfully (1.23s)
dispatch subenv list

Output:
  test-env-1 (awake)
  test-env-2 (sleeping)
  prod-env (awake)
```

### Failed Command

```
❌ Command failed (exit code 1, 0.45s)
dispatch subenv create existing-env

Errors:
  Error: subenvironment 'existing-env' already exists

💡 Hint: Resource already exists. Use a different name or delete the existing resource.
```

### Collapsible Output

Long outputs are collapsible by clicking the header:

```
[▼] ✅ Command executed successfully (2.1s)
    Full output visible

[▶] ✅ Command executed successfully (2.1s)
    Output collapsed
```

## Security Features

### Command Classification

Commands are classified into two categories:

1. **Read-Only** (🟢) - Execute immediately without approval
   - List operations
   - View operations
   - Status checks
   - Context viewing

2. **Write Operations** (🔒) - Require explicit approval
   - Create operations
   - Delete operations
   - Modify operations
   - Login/logout operations

### Approval System

- **Unique approval IDs**: Each request gets a short ID (e.g., `abc123`)
- **User verification**: Only the requesting user can approve their own commands
- **Time-limited**: Approvals expire after 5 minutes
- **Audit trail**: All commands are logged with timestamp and user

### Command Validation

- **Whitelist approach**: Only known commands are allowed
- **Argument sanitization**: No shell injection possible
- **Timeout protection**: Commands timeout after 30 seconds
- **Context cancellation**: Long-running commands can be interrupted

## Error Handling

### Dispatch CLI Not Installed

```
❌ Dispatch CLI Not Found

The dispatch CLI is not installed or not in your PATH.

To install dispatch CLI:
Visit the dispatch CLI repository or contact your DevOps team.
```

### Authentication Required

```
❌ Command failed (exit code 1)

Errors:
  authentication required

💡 Hint: Authentication required. Try `/dispatch aws login` first.
```

### Timeout

```
❌ Command failed (exit code -1, 30.0s)

Errors:
  context deadline exceeded

💡 Hint: The command timed out. The operation may be taking longer than expected.
```

## Configuration

### Timeout Settings

Default timeout is 30 seconds. This is configured in `internal/dispatch/executor.go`:

```go
const DefaultTimeout = 30 * time.Second
```

### Approval Expiration

Approval requests expire after 5 minutes. This is configured in `internal/dispatch/approval.go`:

```go
const ApprovalTTL = 5 * time.Minute
```

### Adding New Commands

To register a new command, update `internal/dispatch/registry.go`:

```go
r.Register("plugin", "command", "Description", readOnly, []string{
    "/dispatch plugin command example",
})
```

## Troubleshooting

### Command Not Found

**Problem:** `/dispatch xyz abc` returns "Unknown command"

**Solution:**
- Check if the command is registered in the registry
- Use `/dispatch-list` to see available commands
- Verify dispatch CLI supports the command: `dispatch xyz --help`

### Permission Denied

**Problem:** Command fails with "permission denied"

**Solution:**
- Ensure dispatch CLI is configured: `dispatch aws login`
- Check your user has appropriate permissions
- Verify credentials are not expired

### Approval Request Not Found

**Problem:** `/approve abc123` returns "approval request not found"

**Solution:**
- Approval may have expired (5 minute limit)
- Check approval ID is correct
- Request the command again

## Architecture

### Components

```
internal/dispatch/
├── types.go          # Data structures
├── executor.go       # Command execution via subprocess
├── registry.go       # Command classification and metadata
├── formatter.go      # Output formatting for chat
└── approval.go       # Approval workflow management
```

### Integration Points

1. **Hub Commands** (`internal/hub/commands.go`)
   - `/dispatch` command handler
   - `/dispatch-list` command handler
   - `/approve` and `/reject` handlers

2. **Protocol** (`internal/protocol/types.go`)
   - `MessageTypeCommandOutput` message type
   - `CommandOutput` struct for metadata

3. **Desktop UI** (`desktop/src/components/`)
   - `CommandOutput.tsx` - Formatted output display
   - `Message.tsx` - Command output rendering
   - `protocol.ts` - TypeScript type definitions

4. **DevOps Agent** (`internal/agent/specialized_agents.go`)
   - Enhanced expertise keywords
   - Dispatch command suggestions

## Future Enhancements

Planned improvements for dispatch integration:

- [ ] Command history and replay
- [ ] Bulk approval for multiple commands
- [ ] Role-based command permissions
- [ ] Real-time streaming output for long-running commands
- [ ] Command scheduling and automation
- [ ] Integration with approval workflows (e.g., Slack notifications)
- [ ] Command aliases and shortcuts
- [ ] Auto-complete for dispatch commands in chat input

## Related Documentation

- [Architecture Documentation](ARCHITECTURE.md)
- [Getting Started Guide](GETTING_STARTED.md)
- [DevOps Agent Documentation](AGENT_REVIEW.md)
- [Repository Agents](REPO_AGENTS.md)

## Support

For issues or questions:

1. Check `/dispatch-list` for available commands
2. Ask the DevOps agent: `@devops How do I use dispatch commands?`
3. Review dispatch CLI documentation
4. Check command output for hints and suggestions

---

**Last Updated:** October 2025
**Status:** Production ready
**Version:** 1.0.0

