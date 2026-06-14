package protocol

import "strings"

var cliAutoEditFileTools = map[string]bool{
	"read_file": true, "write_file": true, "edit_file": true,
	"list_directory": true, "search_files": true, "read_many_files": true,
}

var workspaceReadOnlyMCPTools = map[string]bool{
	"read_file": true, "grep": true, "glob_file_search": true, "semantic_search": true,
	"list_directory": true, "web_search": true,
}

// IsCLIShellToolName reports whether a CLI agent tool call is a shell invocation.
func IsCLIShellToolName(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case "run_shell_command", "shell":
		return true
	default:
		return false
	}
}

// ShellCommandFromToolInput extracts the shell command string from a CLI tool payload.
func ShellCommandFromToolInput(toolInput map[string]interface{}) string {
	if toolInput == nil {
		return ""
	}
	if cmd, ok := toolInput["command"].(string); ok {
		return strings.TrimSpace(cmd)
	}
	return ""
}

// ShouldAutoApproveCLIToolCall decides whether a CLI hook tool call can run without user approval.
func ShouldAutoApproveCLIToolCall(mode, toolName string, toolInput map[string]interface{}) bool {
	mode = strings.TrimSpace(strings.ToLower(mode))
	toolName = strings.TrimSpace(toolName)
	if mode == "yolo" {
		return true
	}
	if mode == "auto_apply_edits" {
		return workspaceReadOnlyMCPTools[toolName]
	}
	if mode != "auto_edit" {
		return false
	}
	if cliAutoEditFileTools[toolName] {
		return true
	}
	if workspaceReadOnlyMCPTools[toolName] {
		return true
	}
	if IsCLIShellToolName(toolName) {
		cmd := ShellCommandFromToolInput(toolInput)
		if cmd != "" && IsSafeShellCommand(cmd) {
			return true
		}
	}
	return false
}
