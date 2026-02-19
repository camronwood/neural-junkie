package dispatch

import (
	"fmt"
	"strings"
)

// CommandRegistry manages dispatch command definitions and permissions
type CommandRegistry struct {
	commands map[string]map[string]*CommandDefinition // plugin -> subcommand -> definition
}

// NewCommandRegistry creates a new command registry with default commands
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]map[string]*CommandDefinition),
	}

	registry.registerDefaultCommands()
	return registry
}

// registerDefaultCommands registers all known dispatch commands
func (r *CommandRegistry) registerDefaultCommands() {
	// Subenv commands
	r.Register("subenv", "list", "List active subenvironments", true, []string{
		"/dispatch subenv list",
	})
	r.Register("subenv", "awake", "Check if a subenvironment is awake", true, []string{
		"/dispatch subenv awake",
	})
	r.Register("subenv", "context", "Show subenvironment context settings", true, []string{
		"/dispatch subenv context",
	})
	r.Register("subenv", "create", "Create a new subenvironment", false, []string{
		"/dispatch subenv create my-subenv",
	})
	r.Register("subenv", "destroy", "Destroy a subenvironment", false, []string{
		"/dispatch subenv destroy my-subenv",
	})
	r.Register("subenv", "deploy", "Deploy branch to subenvironment", false, []string{
		"/dispatch subenv deploy my-branch",
	})
	r.Register("subenv", "exec", "Execute command in subenvironment pod", false, []string{
		"/dispatch subenv exec",
	})
	r.Register("subenv", "open", "Open subenvironment service in browser", true, []string{
		"/dispatch subenv open",
	})
	r.Register("subenv", "prune", "Remove outdated resources", false, []string{
		"/dispatch subenv prune",
	})
	r.Register("subenv", "wake", "Wake up sleeping subenvironment", false, []string{
		"/dispatch subenv wake",
	})
	r.Register("subenv", "update-kubeconfig", "Update kubeconfig for subenv cluster", false, []string{
		"/dispatch subenv update-kubeconfig",
	})

	// kubectl commands
	r.Register("kubectl", "get pods", "List pods in namespace", true, []string{
		"/dispatch kubectl get pods -n <namespace>",
	})
	r.Register("kubectl", "get services", "List services in namespace", true, []string{
		"/dispatch kubectl get services -n <namespace>",
	})
	r.Register("kubectl", "get deployments", "List deployments in namespace", true, []string{
		"/dispatch kubectl get deployments -n <namespace>",
	})
	r.Register("kubectl", "logs", "Get pod logs", true, []string{
		"/dispatch kubectl logs <pod-name> -n <namespace>",
	})
	r.Register("kubectl", "describe pod", "Describe pod details", true, []string{
		"/dispatch kubectl describe pod <pod-name> -n <namespace>",
	})
	r.Register("kubectl", "describe service", "Describe service details", true, []string{
		"/dispatch kubectl describe service <service-name> -n <namespace>",
	})
	r.Register("kubectl", "exec", "Execute command in pod", false, []string{
		"/dispatch kubectl exec -it <pod-name> -n <namespace> -- <command>",
	})
	r.Register("kubectl", "port-forward", "Forward port to pod/service", false, []string{
		"/dispatch kubectl port-forward <pod-name> -n <namespace> <local-port>:<remote-port>",
	})
	r.Register("kubectl", "top pods", "Show pod resource usage", true, []string{
		"/dispatch kubectl top pods -n <namespace>",
	})
	r.Register("kubectl", "get events", "Show events in namespace", true, []string{
		"/dispatch kubectl get events -n <namespace>",
	})

	// AWS commands
	r.Register("aws", "login", "Login to AWS CLI SSO session", false, []string{
		"/dispatch aws login",
	})
	r.Register("aws", "logout", "Logout of AWS CLI SSO session", false, []string{
		"/dispatch aws logout",
	})
	r.Register("aws", "setup", "Setup AWS CLI environment", false, []string{
		"/dispatch aws setup",
	})

	// Docker commands
	r.Register("docker", "login", "Login to container registries", false, []string{
		"/dispatch docker login",
	})
	r.Register("docker", "logout", "Logout of container registries", false, []string{
		"/dispatch docker logout",
	})

	// Kubernetes context commands (kctx)
	r.Register("kctx", "", "Switch Kubernetes cluster context (view only)", true, []string{
		"/dispatch kctx",
	})

	// SOPS commands
	r.Register("sops", "view", "View encrypted files", true, []string{
		"/dispatch sops view secrets.yaml",
	})
	r.Register("sops", "encrypt", "Encrypt files", false, []string{
		"/dispatch sops encrypt file.yaml",
	})
	r.Register("sops", "decrypt", "Decrypt files", false, []string{
		"/dispatch sops decrypt file.yaml",
	})

	// Exec commands
	r.Register("exec", "", "Exec into dispatch namespace pods", false, []string{
		"/dispatch exec",
	})

	// Workstation commands
	r.Register("workstation", "config", "View workstation configuration", true, []string{
		"/dispatch workstation config",
	})
	r.Register("workstation", "shellenv", "Generate environment file", true, []string{
		"/dispatch workstation shellenv",
	})
	r.Register("workstation", "sync", "Run workstation sync commands", false, []string{
		"/dispatch workstation sync",
	})

	// GitHub CLI commands
	// Repository operations
	r.Register("gh", "repo-view", "View repository details", true, []string{
		"/dispatch gh repo-view owner/repo",
	})
	r.Register("gh", "repo-list", "List repositories", true, []string{
		"/dispatch gh repo-list",
		"/dispatch gh repo-list owner",
	})
	r.Register("gh", "repo-clone", "Clone a repository", false, []string{
		"/dispatch gh repo-clone owner/repo",
	})
	r.Register("gh", "repo-fork", "Fork a repository", false, []string{
		"/dispatch gh repo-fork owner/repo",
	})
	r.Register("gh", "repo-create", "Create a new repository", false, []string{
		"/dispatch gh repo-create my-repo",
	})

	// Issue operations
	r.Register("gh", "issue-list", "List issues", true, []string{
		"/dispatch gh issue-list",
		"/dispatch gh issue-list --state open",
	})
	r.Register("gh", "issue-view", "View issue details", true, []string{
		"/dispatch gh issue-view 123",
	})
	r.Register("gh", "issue-create", "Create a new issue", false, []string{
		"/dispatch gh issue-create --title 'Bug report' --body 'Description'",
	})
	r.Register("gh", "issue-close", "Close an issue", false, []string{
		"/dispatch gh issue-close 123",
	})
	r.Register("gh", "issue-reopen", "Reopen an issue", false, []string{
		"/dispatch gh issue-reopen 123",
	})
	r.Register("gh", "issue-comment", "Comment on an issue", false, []string{
		"/dispatch gh issue-comment 123 --body 'My comment'",
	})

	// Pull Request operations
	r.Register("gh", "pr-list", "List pull requests", true, []string{
		"/dispatch gh pr-list",
		"/dispatch gh pr-list --state open",
	})
	r.Register("gh", "pr-view", "View pull request details", true, []string{
		"/dispatch gh pr-view 123",
	})
	r.Register("gh", "pr-create", "Create a new pull request", false, []string{
		"/dispatch gh pr-create --title 'Feature' --body 'Description'",
	})
	r.Register("gh", "pr-checkout", "Checkout a pull request", false, []string{
		"/dispatch gh pr-checkout 123",
	})
	r.Register("gh", "pr-review", "Review a pull request", false, []string{
		"/dispatch gh pr-review 123 --approve",
	})
	r.Register("gh", "pr-merge", "Merge a pull request", false, []string{
		"/dispatch gh pr-merge 123",
	})
	r.Register("gh", "pr-close", "Close a pull request", false, []string{
		"/dispatch gh pr-close 123",
	})
	r.Register("gh", "pr-diff", "View pull request diff", true, []string{
		"/dispatch gh pr-diff 123",
	})

	// Search operations
	r.Register("gh", "search-code", "Search code in repositories", true, []string{
		"/dispatch gh search-code 'query string'",
	})
	r.Register("gh", "search-repos", "Search repositories", true, []string{
		"/dispatch gh search-repos 'query string'",
	})
	r.Register("gh", "search-issues", "Search issues", true, []string{
		"/dispatch gh search-issues 'query string'",
	})
	r.Register("gh", "search-prs", "Search pull requests", true, []string{
		"/dispatch gh search-prs 'query string'",
	})

	// Workflow and run operations
	r.Register("gh", "workflow-list", "List workflows", true, []string{
		"/dispatch gh workflow-list",
	})
	r.Register("gh", "workflow-view", "View workflow details", true, []string{
		"/dispatch gh workflow-view workflow-name",
	})
	r.Register("gh", "workflow-run", "Run a workflow", false, []string{
		"/dispatch gh workflow-run workflow-name",
	})
	r.Register("gh", "run-list", "List workflow runs", true, []string{
		"/dispatch gh run-list",
	})
	r.Register("gh", "run-view", "View workflow run details", true, []string{
		"/dispatch gh run-view 123456",
	})
	r.Register("gh", "run-watch", "Watch a workflow run", true, []string{
		"/dispatch gh run-watch 123456",
	})

	// Status and auth
	r.Register("gh", "auth-status", "Check GitHub authentication status", true, []string{
		"/dispatch gh auth-status",
	})
	r.Register("gh", "status", "View GitHub notifications and status", true, []string{
		"/dispatch gh status",
	})

	// Plugin management
	r.Register("plugin", "list", "List installed plugins", true, []string{
		"/dispatch plugin list",
	})
	r.Register("plugin", "install", "Install a plugin", false, []string{
		"/dispatch plugin install my-plugin",
	})
}

// Register registers a command definition
func (r *CommandRegistry) Register(plugin, subCmd, description string, readOnly bool, examples []string) {
	if r.commands[plugin] == nil {
		r.commands[plugin] = make(map[string]*CommandDefinition)
	}

	r.commands[plugin][subCmd] = &CommandDefinition{
		Plugin:      plugin,
		SubCommand:  subCmd,
		Description: description,
		ReadOnly:    readOnly,
		Examples:    examples,
	}
}

// IsReadOnly checks if a command is read-only (doesn't require approval)
func (r *CommandRegistry) IsReadOnly(plugin, subCmd string) bool {
	if cmds, ok := r.commands[plugin]; ok {
		// Check exact subcommand match
		if def, ok := cmds[subCmd]; ok {
			return def.ReadOnly
		}

		// Check if plugin has default entry (empty subCmd)
		if def, ok := cmds[""]; ok {
			return def.ReadOnly
		}
	}

	// Default to requiring approval (not read-only) for unknown commands
	return false
}

// IsKnownCommand checks if a command is registered
func (r *CommandRegistry) IsKnownCommand(plugin, subCmd string) bool {
	if cmds, ok := r.commands[plugin]; ok {
		_, exactMatch := cmds[subCmd]
		_, defaultMatch := cmds[""]
		return exactMatch || defaultMatch
	}
	return false
}

// GetCommand gets a command definition
func (r *CommandRegistry) GetCommand(plugin, subCmd string) (*CommandDefinition, error) {
	if cmds, ok := r.commands[plugin]; ok {
		if def, ok := cmds[subCmd]; ok {
			return def, nil
		}
		if def, ok := cmds[""]; ok {
			return def, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %s %s", plugin, subCmd)
}

// ListPlugins lists all registered plugins
func (r *CommandRegistry) ListPlugins() []string {
	plugins := make([]string, 0, len(r.commands))
	for plugin := range r.commands {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// ListCommands lists all commands for a plugin
func (r *CommandRegistry) ListCommands(plugin string) []*CommandDefinition {
	cmds := make([]*CommandDefinition, 0)
	if pluginCmds, ok := r.commands[plugin]; ok {
		for _, def := range pluginCmds {
			cmds = append(cmds, def)
		}
	}
	return cmds
}

// GetAllCommands returns all registered commands grouped by plugin
func (r *CommandRegistry) GetAllCommands() map[string][]*CommandDefinition {
	result := make(map[string][]*CommandDefinition)
	for plugin := range r.commands {
		result[plugin] = r.ListCommands(plugin)
	}
	return result
}

// FormatCommandHelp formats help text for a command
func (r *CommandRegistry) FormatCommandHelp(plugin, subCmd string) string {
	def, err := r.GetCommand(plugin, subCmd)
	if err != nil {
		return fmt.Sprintf("Unknown command: %s %s", plugin, subCmd)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s %s**\n", plugin, subCmd))
	sb.WriteString(fmt.Sprintf("%s\n\n", def.Description))

	if def.ReadOnly {
		sb.WriteString("🟢 **Read-only** - Executes immediately\n")
	} else {
		sb.WriteString("🔒 **Requires approval** - Must be approved with `/approve`\n")
	}

	if len(def.Examples) > 0 {
		sb.WriteString("\n**Examples:**\n")
		for _, ex := range def.Examples {
			sb.WriteString(fmt.Sprintf("- `%s`\n", ex))
		}
	}

	return sb.String()
}
