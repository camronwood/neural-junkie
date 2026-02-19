package protocol

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CommandDetector detects shell commands in agent responses
type CommandDetector struct {
	dispatchRegistry interface {
		IsKnownCommand(plugin, subCmd string) bool
		IsReadOnly(plugin, subCmd string) bool
	}
}

// NewCommandDetector creates a new command detector
func NewCommandDetector(dispatchRegistry interface {
	IsKnownCommand(plugin, subCmd string) bool
	IsReadOnly(plugin, subCmd string) bool
}) *CommandDetector {
	return &CommandDetector{
		dispatchRegistry: dispatchRegistry,
	}
}

// DetectCommands scans message content for shell commands and returns suggestions
func (cd *CommandDetector) DetectCommands(content, agentName, messageID string) []CommandSuggestion {
	var suggestions []CommandSuggestion

	// Look for code blocks with bash, sh, or shell tags
	codeBlockRegex := regexp.MustCompile("```(?:bash|sh|shell|zsh|fish)\n(.*?)\n```")
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			command := strings.TrimSpace(match[1])
			if command != "" {
				suggestion := cd.createCommandSuggestion(command, agentName, messageID, content)
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	// Look for inline commands (single backticks with shell commands)
	inlineRegex := regexp.MustCompile("`([^`]+)`")
	inlineMatches := inlineRegex.FindAllStringSubmatch(content, -1)

	for _, match := range inlineMatches {
		if len(match) > 1 {
			command := strings.TrimSpace(match[1])
			if cd.isShellCommand(command) {
				suggestion := cd.createCommandSuggestion(command, agentName, messageID, content)
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	// Look for dispatch commands specifically
	dispatchRegex := regexp.MustCompile("`(/dispatch [^`]+)`")
	dispatchMatches := dispatchRegex.FindAllStringSubmatch(content, -1)

	for _, match := range dispatchMatches {
		if len(match) > 1 {
			command := strings.TrimSpace(match[1])
			suggestion := cd.createDispatchSuggestion(command, agentName, messageID, content)
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// isShellCommand checks if a command looks like a shell command
func (cd *CommandDetector) isShellCommand(command string) bool {
	// Common shell command patterns
	shellPatterns := []string{
		"ls", "cd", "pwd", "mkdir", "rm", "cp", "mv", "cat", "grep", "find",
		"ps", "kill", "top", "htop", "df", "du", "tar", "zip", "unzip",
		"git", "npm", "yarn", "pip", "docker", "kubectl", "aws", "curl", "wget",
		"ssh", "scp", "rsync", "chmod", "chown", "sudo", "su",
	}

	command = strings.TrimSpace(command)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	firstWord := parts[0]
	for _, pattern := range shellPatterns {
		if strings.HasPrefix(firstWord, pattern) {
			return true
		}
	}

	// Check for commands that start with common prefixes
	if strings.HasPrefix(firstWord, "./") || strings.HasPrefix(firstWord, "../") {
		return true
	}

	return false
}

// createCommandSuggestion creates a command suggestion for a shell command
func (cd *CommandDetector) createCommandSuggestion(command, agentName, messageID, content string) CommandSuggestion {
	description := cd.extractDescription(command, content)

	return CommandSuggestion{
		ID:          uuid.New().String(),
		Command:     command,
		Plugin:      "shell",
		Description: description,
		IsSafe:      cd.isSafeCommand(command),
		AgentName:   agentName,
		MessageID:   messageID,
		CreatedAt:   time.Now(),
	}
}

// createDispatchSuggestion creates a command suggestion for a dispatch command
func (cd *CommandDetector) createDispatchSuggestion(command, agentName, messageID, content string) CommandSuggestion {
	description := cd.extractDescription(command, content)

	// Parse dispatch command to get plugin and subcommand
	parts := strings.Fields(command)
	plugin := "unknown"
	subCmd := "unknown"

	if len(parts) >= 3 && parts[0] == "/dispatch" {
		plugin = parts[1]
		subCmd = parts[2]
	}

	// Check if it's a known dispatch command and if it's read-only
	isKnown := cd.dispatchRegistry.IsKnownCommand(plugin, subCmd)
	isReadOnly := false
	if isKnown {
		isReadOnly = cd.dispatchRegistry.IsReadOnly(plugin, subCmd)
	}

	return CommandSuggestion{
		ID:          uuid.New().String(),
		Command:     command,
		Plugin:      plugin,
		Description: description,
		IsSafe:      isReadOnly,
		AgentName:   agentName,
		MessageID:   messageID,
		CreatedAt:   time.Now(),
	}
}

// isSafeCommand determines if a shell command is safe to execute
func (cd *CommandDetector) isSafeCommand(command string) bool {
	// Read-only commands that are generally safe
	safeCommands := []string{
		"ls", "pwd", "cat", "head", "tail", "grep", "find", "ps", "top", "htop",
		"df", "du", "who", "whoami", "date", "uptime", "uname", "which", "whereis",
		"git status", "git log", "git diff", "git show", "git branch", "git tag",
		"docker ps", "docker images", "docker logs", "kubectl get", "kubectl describe",
		"aws s3 ls", "aws ec2 describe", "curl -I", "wget --spider",
	}

	command = strings.TrimSpace(strings.ToLower(command))

	for _, safe := range safeCommands {
		if strings.HasPrefix(command, safe) {
			return true
		}
	}

	// Commands that are definitely not safe
	unsafePatterns := []string{
		"rm ", "rmdir", "del ", "rm -rf", "rm -r", "rm -f",
		"chmod", "chown", "chgrp", "sudo", "su ", "su-",
		"kill", "killall", "pkill", "xkill",
		"format", "fdisk", "mkfs", "dd if=", "dd of=",
		"shutdown", "reboot", "halt", "poweroff",
		"passwd", "useradd", "userdel", "usermod",
		"mount", "umount", "umount -f",
		"> ", ">> ", "tee ", "> /dev/", "> /proc/",
		"curl -X POST", "curl -X PUT", "curl -X DELETE",
		"wget -O", "wget --output-document",
	}

	for _, pattern := range unsafePatterns {
		if strings.Contains(command, pattern) {
			return false
		}
	}

	// Default to unsafe for unknown commands
	return false
}

// extractDescription extracts a description from the surrounding content
func (cd *CommandDetector) extractDescription(command, content string) string {
	// Look for text before the command that might be a description
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		if strings.Contains(line, command) {
			// Look at the previous line for description
			if i > 0 {
				prevLine := strings.TrimSpace(lines[i-1])
				if prevLine != "" && !strings.HasPrefix(prevLine, "```") && !strings.HasPrefix(prevLine, "`") {
					return prevLine
				}
			}

			// Look at the current line for description
			parts := strings.Split(line, command)
			if len(parts) > 0 {
				before := strings.TrimSpace(parts[0])
				if before != "" && !strings.HasPrefix(before, "```") && !strings.HasPrefix(before, "`") {
					return before
				}
			}
		}
	}

	// Default description
	return "Command suggested by " + strings.Split(command, " ")[0]
}
