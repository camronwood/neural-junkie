package hub

import (
	"fmt"
	"strings"
)

// RequestToolApproval blocks until the user approves or rejects a mutating tool call.
func (h *Hub) RequestToolApproval(agentID, agentName, channel, toolName string, toolInput map[string]interface{}) (bool, error) {
	approved, _, err := h.RequestToolApprovalScoped(agentID, agentName, channel, toolName, toolInput)
	return approved, err
}

// RequestToolApprovalScoped is like RequestToolApproval but also returns whether the
// user chose "always" (persist allow) vs a one-shot approve.
func (h *Hub) RequestToolApprovalScoped(agentID, agentName, channel, toolName string, toolInput map[string]interface{}) (approved bool, always bool, err error) {
	if h == nil || h.toolApprovalManager == nil {
		return false, false, nil
	}
	if channel == "" {
		channel = "general"
	}
	approval := h.toolApprovalManager.CreateApproval(agentID, agentName, "", toolName, channel, toolInput)
	status, reason := h.toolApprovalManager.WaitForDecision(approval.ID, ToolApprovalTTL)
	if status != ToolApprovalApproved {
		return false, false, nil
	}
	always = strings.EqualFold(strings.TrimSpace(reason), "always")
	return true, always, nil
}

// RequestRunCommandApproval prompts the user when a shell command is not allowlisted.
func (h *Hub) RequestRunCommandApproval(agentID, agentName, channel, command string) (approved bool, always bool, err error) {
	command = strings.Join(strings.Fields(strings.TrimSpace(command)), " ")
	if command == "" {
		return false, false, fmt.Errorf("command required")
	}
	return h.RequestToolApprovalScoped(agentID, agentName, channel, "run_command", map[string]interface{}{
		"command": command,
		"reason":  "not_allowlisted",
	})
}

// ToolApprovalSummary returns a human-readable summary for incident/ticket tools.
func ToolApprovalSummary(toolName string, toolInput map[string]interface{}) string {
	if toolInput == nil {
		return toolName
	}
	if toolName == "run_command" || toolName == "run_shell_command" || toolName == "shell" {
		if cmd, ok := toolInput["command"].(string); ok && strings.TrimSpace(cmd) != "" {
			return strings.TrimSpace(cmd)
		}
	}
	raw := formatToolInput(toolName, toolInput)
	return toolName + ": " + raw
}
