package hub

import (
	"encoding/json"
)

// RequestToolApproval blocks until the user approves or rejects a mutating tool call.
func (h *Hub) RequestToolApproval(agentID, agentName, channel, toolName string, toolInput map[string]interface{}) (bool, error) {
	if h == nil || h.toolApprovalManager == nil {
		return false, nil
	}
	if channel == "" {
		channel = "general"
	}
	approval := h.toolApprovalManager.CreateApproval(agentID, agentName, "", toolName, channel, toolInput)
	status, _ := h.toolApprovalManager.WaitForDecision(approval.ID, ToolApprovalTTL)
	return status == ToolApprovalApproved, nil
}

// ToolApprovalSummary returns a human-readable summary for incident/ticket tools.
func ToolApprovalSummary(toolName string, toolInput map[string]interface{}) string {
	if toolInput == nil {
		return toolName
	}
	raw, _ := json.Marshal(toolInput)
	return toolName + ": " + string(raw)
}
