package protocol

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/google/uuid"
)

// hubMCPToolNames are in-process MCP tools that must never be offered as shell commands.
// Canonical list lives in internal/mcp/tool_names.go.

var bareFileRefExtensions = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true,
	".xml": true, ".html": true, ".htm": true, ".csv": true, ".ts": true, ".tsx": true,
	".js": true, ".jsx": true, ".go": true, ".py": true, ".rs": true, ".java": true,
	".c": true, ".h": true, ".cpp": true, ".hpp": true, ".sql": true, ".sh": true,
	".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true,
}

var moduleImportPathPrefixes = []string{
	"github.com/", "gitlab.com/", "golang.org/", "gopkg.in/", "bitbucket.org/",
}

// golangci-lint and similar tools emit locations like path.go:64.16,66.3 1 0
var linterOutputLinePattern = regexp.MustCompile(`:\d+\.\d+,\d+\.\d+(?:\s+\d+(?:\s+\d+)?)?$`)

// CommandDetector detects shell commands in agent responses
type CommandDetector struct{}

// NewCommandDetector creates a new command detector
func NewCommandDetector(_ interface{}) *CommandDetector {
	return &CommandDetector{}
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
			if command != "" && cd.shouldSuggestCodeBlock(command) {
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
			if isHubMCPToolCommand(command) {
				continue
			}
			if looksLikeNonShellReference(command) {
				continue
			}
			if cd.isShellCommand(command) {
				suggestion := cd.createCommandSuggestion(command, agentName, messageID, content)
				suggestions = append(suggestions, suggestion)
			}
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
		"git", "go", "make", "npm", "yarn", "pnpm", "pip", "python", "node",
		"cargo", "rustc", "docker", "kubectl", "aws", "curl", "wget",
		"ssh", "scp", "rsync", "chmod", "chown", "sudo", "su",
	}

	command = strings.TrimSpace(command)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	firstWord := parts[0]
	for _, pattern := range shellPatterns {
		if firstWord == pattern {
			return true
		}
	}

	// Check for commands that start with common prefixes
	if strings.HasPrefix(firstWord, "./") || strings.HasPrefix(firstWord, "../") {
		return true
	}

	return false
}

// looksLikeNonShellReference matches file paths, module import paths, and linter output
// that agents sometimes put in ```bash blocks or inline backticks by mistake.
func looksLikeNonShellReference(s string) bool {
	if looksLikeBareFileReference(s) {
		return true
	}
	if looksLikeModuleImportPath(s) {
		return true
	}
	return looksLikeLinterOutputLine(s)
}

func looksLikeModuleImportPath(s string) bool {
	s = strings.TrimSpace(s)
	for _, prefix := range moduleImportPathPrefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func looksLikeLinterOutputLine(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if linterOutputLinePattern.MatchString(s) {
		return true
	}
	// file.go:10:5 or file.go:10 style references in tool output
	if strings.Count(s, ":") >= 2 {
		lastColon := strings.LastIndex(s, ":")
		if lastColon > 0 && lastColon < len(s)-1 {
			suffix := s[lastColon+1:]
			if _, ok := parseLeadingInt(suffix); ok {
				return true
			}
		}
	}
	return false
}

func parseLeadingInt(s string) (int, bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, false
	}
	return i, true
}

// looksLikeBareFileReference matches deliverable paths (e.g. findings.md, collabs/id/out.md).
func looksLikeBareFileReference(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.Contains(s, "|") || strings.Contains(s, "&&") || strings.Contains(s, ";") {
		return false
	}
	if strings.Contains(s, " ") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(s))
	if ext != "" && bareFileRefExtensions[ext] {
		return true
	}
	if strings.Contains(s, "/") {
		return true
	}
	return false
}

func substantiveCodeBlockLines(command string) []string {
	var lines []string
	for _, line := range strings.Split(command, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// shouldSuggestCodeBlock filters ```bash``` blocks that are only file paths, not commands.
func isHubMCPToolCommand(command string) bool {
	line := strings.TrimSpace(strings.Split(command, "\n")[0])
	if line == "" {
		return false
	}
	first := strings.Fields(line)[0]
	return mcp.IsHubMCPToolName(first)
}

func (cd *CommandDetector) shouldSuggestCodeBlock(command string) bool {
	lines := substantiveCodeBlockLines(command)
	if len(lines) == 0 {
		return false
	}
	for _, line := range lines {
		if looksLikeNonShellReference(line) {
			return false
		}
		if isHubMCPToolCommand(line) {
			return false
		}
	}
	for _, line := range lines {
		if cd.isShellCommand(line) {
			return true
		}
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

// IsSafeShellCommand reports whether a shell command is read-only / low-risk enough to auto-run.
func IsSafeShellCommand(command string) bool {
	return NewCommandDetector(nil).isSafeCommand(command)
}

// isSafeCommand determines if a shell command is safe to execute
func (cd *CommandDetector) isSafeCommand(command string) bool {
	// Read-only commands that are generally safe
	safeCommands := []string{
		"ls", "pwd", "cd ", "cat", "head", "tail", "grep", "find", "file", "tree",
		"wc", "sort", "uniq", "env", "printenv", "ps", "top", "htop",
		"df", "du", "who", "whoami", "date", "uptime", "uname", "which", "whereis",
		"git status", "git log", "git diff", "git show", "git branch", "git tag",
		"docker ps", "docker images", "docker logs", "kubectl get", "kubectl describe",
		"aws s3 ls", "aws ec2 describe", "curl -I", "wget --spider",
		"go test", "go list", "go version", "cargo check", "cargo test", "make test",
		"python -m pytest", "python -m compileall", "npm test", "npm run",
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
