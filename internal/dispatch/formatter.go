package dispatch

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Formatter handles formatting of command outputs for chat display
type Formatter struct{}

// NewFormatter creates a new output formatter
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatOutput formats a command result for chat display
func (f *Formatter) FormatOutput(result *CommandResult) string {
	var sb strings.Builder

	// Header with command and status
	if result.Success {
		sb.WriteString(fmt.Sprintf("✅ **Command executed successfully** (%.2fs)\n", result.Duration.Seconds()))
	} else {
		sb.WriteString(fmt.Sprintf("❌ **Command failed** (exit code %d, %.2fs)\n", result.ExitCode, result.Duration.Seconds()))
	}

	sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", result.Command))

	// Output section
	if result.Stdout != "" {
		formatted := f.formatOutputText(result.Stdout)
		sb.WriteString("**Output:**\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", formatted))
	}

	// Error section (only if present)
	if result.Stderr != "" {
		formatted := f.formatOutputText(result.Stderr)
		sb.WriteString("\n**Errors:**\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", formatted))
	}

	// Helpful hints for common errors
	if !result.Success {
		hint := f.getErrorHint(result)
		if hint != "" {
			sb.WriteString(fmt.Sprintf("\n💡 **Hint:** %s\n", hint))
		}
	}

	return sb.String()
}

// FormatShortOutput creates a compact one-line summary
func (f *Formatter) FormatShortOutput(result *CommandResult) string {
	status := "✅"
	if !result.Success {
		status = "❌"
	}

	return fmt.Sprintf("%s `%s` (%.2fs)", status, result.Command, result.Duration.Seconds())
}

// FormatApprovalRequest formats an approval request message
func (f *Formatter) FormatApprovalRequest(approval *PendingApproval) string {
	var sb strings.Builder

	sb.WriteString("🔒 **Approval Required**\n\n")
	sb.WriteString(fmt.Sprintf("User **%s** wants to execute:\n", approval.Username))
	sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", approval.Command))
	sb.WriteString("⚠️  **This command requires approval because it can modify system state.**\n\n")
	sb.WriteString(fmt.Sprintf("**To approve:** `/approve %s`\n", approval.ID))
	sb.WriteString(fmt.Sprintf("**To reject:** `/reject %s`\n\n", approval.ID))
	sb.WriteString(fmt.Sprintf("*Request expires in 5 minutes (%s)*\n", approval.ExpiresAt.Format("15:04:05")))

	return sb.String()
}

// FormatApproved formats an approval confirmation message
func (f *Formatter) FormatApproved(approval *PendingApproval) string {
	return fmt.Sprintf("✅ **Approved** - Executing command:\n```\n%s\n```", approval.Command)
}

// FormatRejected formats a rejection confirmation message
func (f *Formatter) FormatRejected(approval *PendingApproval) string {
	return fmt.Sprintf("🚫 **Rejected** - Command not executed:\n```\n%s\n```", approval.Command)
}

// FormatExpired formats an expiration message
func (f *Formatter) FormatExpired(approval *PendingApproval) string {
	return fmt.Sprintf("⏱️  **Expired** - Approval request timed out:\n```\n%s\n```", approval.Command)
}

// formatOutputText formats raw output text for better display
func (f *Formatter) formatOutputText(text string) string {
	// Trim excessive whitespace
	text = strings.TrimSpace(text)

	// Try to detect and format JSON
	if f.looksLikeJSON(text) {
		if formatted := f.formatJSON(text); formatted != "" {
			return formatted
		}
	}

	// Limit output length for very long outputs
	const maxLength = 2000
	if len(text) > maxLength {
		text = text[:maxLength] + "\n\n... (output truncated, " + fmt.Sprintf("%d", len(text)-maxLength) + " bytes omitted)"
	}

	return text
}

// looksLikeJSON checks if text appears to be JSON
func (f *Formatter) looksLikeJSON(text string) bool {
	trimmed := strings.TrimSpace(text)
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))
}

// formatJSON attempts to pretty-print JSON
func (f *Formatter) formatJSON(text string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return ""
	}

	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return ""
	}

	return string(formatted)
}

// getErrorHint provides helpful hints for common errors
func (f *Formatter) getErrorHint(result *CommandResult) string {
	errText := strings.ToLower(result.Stderr + " " + result.Stdout)

	switch {
	case strings.Contains(errText, "not found"):
		return "The command or resource was not found. Check spelling and availability."
	case strings.Contains(errText, "permission denied"):
		return "Permission denied. You may need to authenticate first."
	case strings.Contains(errText, "timeout"):
		return "The command timed out. The operation may be taking longer than expected."
	case strings.Contains(errText, "authentication") || strings.Contains(errText, "auth"):
		return fmt.Sprintf("Authentication required. Try `/dispatch %s login` first.", result.Plugin)
	case strings.Contains(errText, "not logged in"):
		return fmt.Sprintf("You're not logged in. Try `/dispatch %s login` first.", result.Plugin)
	case strings.Contains(errText, "does not exist"):
		return "The specified resource doesn't exist. Try listing available resources first."
	case strings.Contains(errText, "already exists"):
		return "Resource already exists. Use a different name or delete the existing resource."
	case result.ExitCode == 127:
		return "Command not found. Is the dispatch CLI installed and in your PATH?"
	case result.ExitCode == 126:
		return "Command cannot be executed. Check permissions."
	default:
		return ""
	}
}

// FormatCommandList formats a list of available commands
func (f *Formatter) FormatCommandList(commandsByPlugin map[string][]*CommandDefinition) string {
	var sb strings.Builder

	sb.WriteString("**📋 Available Dispatch Commands**\n\n")
	sb.WriteString("Commands are organized by plugin. ")
	sb.WriteString("🟢 Read-only commands execute immediately, ")
	sb.WriteString("🔒 Write commands require approval.\n\n")

	// Sort plugins for consistent output
	plugins := []string{"subenv", "aws", "docker", "kctx", "sops", "exec", "workstation", "plugin"}

	for _, plugin := range plugins {
		cmds, ok := commandsByPlugin[plugin]
		if !ok || len(cmds) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### **%s**\n", plugin))

		for _, cmd := range cmds {
			icon := "🔒"
			if cmd.ReadOnly {
				icon = "🟢"
			}

			subCmdStr := cmd.SubCommand
			if subCmdStr == "" {
				subCmdStr = "(default)"
			}

			sb.WriteString(fmt.Sprintf("%s `%s %s` - %s\n", icon, plugin, subCmdStr, cmd.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("**Usage:** `/dispatch <plugin> <command> [args...]`\n")
	sb.WriteString("**Example:** `/dispatch subenv list`\n\n")
	sb.WriteString("For more info on a specific command, try running it with `--help`\n")

	return sb.String()
}

// FormatNotInstalled formats a message when dispatch CLI is not installed
func (f *Formatter) FormatNotInstalled() string {
	return `❌ **Dispatch CLI Not Found**

The dispatch CLI is not installed or not in your PATH.

**To install dispatch CLI:**
Visit the dispatch CLI repository or contact your DevOps team for installation instructions.

**To check if it's installed:**
` + "```\nwhich dispatch\n```"
}
