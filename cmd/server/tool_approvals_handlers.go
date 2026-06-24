package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func handleToolApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID   string                 `json:"agent_id"`
		AgentName string                 `json:"agent_name"`
		SessionID string                 `json:"session_id"`
		ToolName  string                 `json:"tool_name"`
		ToolInput map[string]interface{} `json:"tool_input"`
		Channel   string                 `json:"channel"`
		Mode      string                 `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ToolName == "" {
		http.Error(w, "tool_name is required", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		req.Channel = "general"
	}

	// If mode is yolo, auto-approve without user interaction
	if req.Mode == "yolo" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "approved",
			"decision": "allow",
		})
		return
	}

	if protocol.ShouldAutoApproveCLIToolCall(req.Mode, req.ToolName, req.ToolInput) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "approved",
			"decision": "allow",
		})
		return
	}

	tam := chatHub.GetToolApprovalManager()
	approval := tam.CreateApproval(req.AgentID, req.AgentName, req.SessionID, req.ToolName, req.Channel, req.ToolInput)

	log.Printf("[ToolApproval] Created approval %s for %s.%s", approval.ID, req.AgentName, req.ToolName)

	// Block until user decides (up to 3 minutes)
	status, reason := tam.WaitForDecision(approval.ID, hub.ToolApprovalTTL)

	decision := "deny"
	if status == hub.ToolApprovalApproved {
		decision = "allow"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   string(status),
		"decision": decision,
		"reason":   reason,
	})
}

// handleApproveToolCall approves a pending tool call

func handleApproveToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}

	approvalID := strings.TrimPrefix(r.URL.Path, "/api/tool-approvals/approve/")
	if approvalID == "" {
		http.Error(w, "Approval ID required", http.StatusBadRequest)
		return
	}

	tam := chatHub.GetToolApprovalManager()
	if err := tam.Approve(approvalID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// handleRejectToolCall rejects a pending tool call

func handleRejectToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}

	approvalID := strings.TrimPrefix(r.URL.Path, "/api/tool-approvals/reject/")
	if approvalID == "" {
		http.Error(w, "Approval ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Reason == "" {
		req.Reason = "User rejected"
	}

	tam := chatHub.GetToolApprovalManager()
	if err := tam.Reject(approvalID, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

// handlePendingToolApprovals lists all currently pending tool approvals

func handlePendingToolApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tam := chatHub.GetToolApprovalManager()
	pending := tam.ListPending()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pending)
}

// handleSetApprovalMode updates the approval mode for a CLI agent

func handleSetApprovalMode(w http.ResponseWriter, r *http.Request, agentID string) {
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	validModes := map[string]bool{"interactive": true, "auto_edit": true, "yolo": true}
	if !validModes[req.Mode] {
		http.Error(w, "Invalid mode. Use 'interactive', 'auto_edit', or 'yolo'", http.StatusBadRequest)
		return
	}

	agents := chatHub.ListAgents()
	var targetAgent *protocol.AgentInfo
	for _, agent := range agents {
		if agent.ID == agentID {
			targetAgent = agent
			break
		}
	}

	if targetAgent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	if targetAgent.Type != protocol.AgentTypeCLI {
		http.Error(w, "Approval mode only applies to CLI agents", http.StatusBadRequest)
		return
	}

	targetAgent.ApprovalMode = req.Mode
	log.Printf("[ApprovalMode] Set %s (%s) to %s", targetAgent.Name, agentID, req.Mode)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "mode": req.Mode})
}

// handleSetAgentCustomRules updates persisted markdown instructions for any registered agent.

func handleSetAgentCustomRules(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := chatHub.SetAgentCustomRulesMarkdown(agentID, req.Markdown); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAgentProvider handles switching individual agent providers and approval mode
