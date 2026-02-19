package hub

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

const (
	ToolApprovalTTL        = 3 * time.Minute
	ToolApprovalCleanupInt = 30 * time.Second
)

type ToolApprovalStatus string

const (
	ToolApprovalPending  ToolApprovalStatus = "pending"
	ToolApprovalApproved ToolApprovalStatus = "approved"
	ToolApprovalRejected ToolApprovalStatus = "rejected"
	ToolApprovalExpired  ToolApprovalStatus = "expired"
)

// ToolApproval represents a pending tool call that needs user approval.
type ToolApproval struct {
	ID         string                 `json:"id"`
	AgentID    string                 `json:"agent_id"`
	AgentName  string                 `json:"agent_name"`
	SessionID  string                 `json:"session_id"`
	ToolName   string                 `json:"tool_name"`
	ToolInput  map[string]interface{} `json:"tool_input"`
	Status     ToolApprovalStatus     `json:"status"`
	Reason     string                 `json:"reason,omitempty"`
	Channel    string                 `json:"channel"`
	CreatedAt  time.Time              `json:"created_at"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// ToolApprovalManager manages pending tool approval requests.
// The hook binary creates approvals via CreateApproval and blocks on
// WaitForDecision. The frontend resolves them via Approve/Reject.
type ToolApprovalManager struct {
	mu        sync.Mutex
	approvals map[string]*ToolApproval
	waiters   map[string]chan ToolApprovalStatus // approval ID -> signal channel

	hub         *Hub
	stopCleanup chan struct{}
}

func NewToolApprovalManager(hub *Hub) *ToolApprovalManager {
	tam := &ToolApprovalManager{
		approvals:   make(map[string]*ToolApproval),
		waiters:     make(map[string]chan ToolApprovalStatus),
		hub:         hub,
		stopCleanup: make(chan struct{}),
	}
	go tam.cleanupLoop()
	return tam
}

func (tam *ToolApprovalManager) Stop() {
	close(tam.stopCleanup)
}

// CreateApproval registers a new pending tool approval and broadcasts it to the chat.
func (tam *ToolApprovalManager) CreateApproval(agentID, agentName, sessionID, toolName, channel string, toolInput map[string]interface{}) *ToolApproval {
	tam.mu.Lock()
	defer tam.mu.Unlock()

	approval := &ToolApproval{
		ID:        uuid.New().String()[:8],
		AgentID:   agentID,
		AgentName: agentName,
		SessionID: sessionID,
		ToolName:  toolName,
		ToolInput: toolInput,
		Status:    ToolApprovalPending,
		Channel:   channel,
		CreatedAt: time.Now(),
	}

	tam.approvals[approval.ID] = approval
	tam.waiters[approval.ID] = make(chan ToolApprovalStatus, 1)

	// Broadcast to chat so the frontend can render the approval card
	tam.broadcastApproval(approval)

	return approval
}

// WaitForDecision blocks until the approval is resolved or the timeout expires.
// Returns the final status.
func (tam *ToolApprovalManager) WaitForDecision(approvalID string, timeout time.Duration) (ToolApprovalStatus, string) {
	tam.mu.Lock()
	ch, ok := tam.waiters[approvalID]
	tam.mu.Unlock()

	if !ok {
		return ToolApprovalRejected, "approval not found"
	}

	select {
	case status := <-ch:
		tam.mu.Lock()
		reason := ""
		if a, exists := tam.approvals[approvalID]; exists {
			reason = a.Reason
		}
		tam.mu.Unlock()
		return status, reason
	case <-time.After(timeout):
		tam.mu.Lock()
		if a, exists := tam.approvals[approvalID]; exists && a.Status == ToolApprovalPending {
			now := time.Now()
			a.Status = ToolApprovalExpired
			a.ResolvedAt = &now
			a.Reason = "timed out waiting for user decision"
		}
		delete(tam.waiters, approvalID)
		tam.mu.Unlock()
		return ToolApprovalExpired, "timed out waiting for user decision"
	}
}

// Approve resolves a pending approval as approved.
func (tam *ToolApprovalManager) Approve(approvalID string) error {
	tam.mu.Lock()
	defer tam.mu.Unlock()

	approval, ok := tam.approvals[approvalID]
	if !ok {
		return fmt.Errorf("approval not found: %s", approvalID)
	}
	if approval.Status != ToolApprovalPending {
		return fmt.Errorf("approval already resolved: %s", approval.Status)
	}

	now := time.Now()
	approval.Status = ToolApprovalApproved
	approval.ResolvedAt = &now

	if ch, ok := tam.waiters[approvalID]; ok {
		ch <- ToolApprovalApproved
		delete(tam.waiters, approvalID)
	}

	tam.broadcastApprovalUpdate(approval)
	return nil
}

// Reject resolves a pending approval as rejected.
func (tam *ToolApprovalManager) Reject(approvalID, reason string) error {
	tam.mu.Lock()
	defer tam.mu.Unlock()

	approval, ok := tam.approvals[approvalID]
	if !ok {
		return fmt.Errorf("approval not found: %s", approvalID)
	}
	if approval.Status != ToolApprovalPending {
		return fmt.Errorf("approval already resolved: %s", approval.Status)
	}

	now := time.Now()
	approval.Status = ToolApprovalRejected
	approval.ResolvedAt = &now
	approval.Reason = reason

	if ch, ok := tam.waiters[approvalID]; ok {
		ch <- ToolApprovalRejected
		delete(tam.waiters, approvalID)
	}

	tam.broadcastApprovalUpdate(approval)
	return nil
}

// ListPending returns all currently pending approvals.
func (tam *ToolApprovalManager) ListPending() []*ToolApproval {
	tam.mu.Lock()
	defer tam.mu.Unlock()

	var pending []*ToolApproval
	for _, a := range tam.approvals {
		if a.Status == ToolApprovalPending {
			pending = append(pending, a)
		}
	}
	return pending
}

func (tam *ToolApprovalManager) broadcastApproval(a *ToolApproval) {
	inputSummary := formatToolInput(a.ToolName, a.ToolInput)

	msg := &protocol.Message{
		ID:      uuid.New().String(),
		Type:    protocol.MessageTypeToolApproval,
		Channel: a.Channel,
		From: protocol.AgentInfo{
			ID:   a.AgentID,
			Name: a.AgentName,
			Type: protocol.AgentTypeCLI,
		},
		Content:   fmt.Sprintf("**%s** wants to use tool **%s**: %s", a.AgentName, a.ToolName, inputSummary),
		Timestamp: a.CreatedAt,
		Metadata: map[string]interface{}{
			"approval_id": a.ID,
			"tool_name":   a.ToolName,
			"tool_input":  a.ToolInput,
			"status":      string(a.Status),
		},
	}

	if err := tam.hub.SendMessage(msg); err != nil {
		log.Printf("[ToolApproval] Failed to broadcast approval %s: %v", a.ID, err)
	}
}

func (tam *ToolApprovalManager) broadcastApprovalUpdate(a *ToolApproval) {
	msg := &protocol.Message{
		ID:      uuid.New().String(),
		Type:    protocol.MessageTypeToolApproval,
		Channel: a.Channel,
		From: protocol.AgentInfo{
			ID:   a.AgentID,
			Name: a.AgentName,
			Type: protocol.AgentTypeCLI,
		},
		Content:   fmt.Sprintf("Tool **%s** was **%s**", a.ToolName, a.Status),
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"approval_id": a.ID,
			"tool_name":   a.ToolName,
			"status":      string(a.Status),
			"reason":      a.Reason,
		},
	}

	if err := tam.hub.SendMessage(msg); err != nil {
		log.Printf("[ToolApproval] Failed to broadcast update %s: %v", a.ID, err)
	}
}

func (tam *ToolApprovalManager) cleanupLoop() {
	ticker := time.NewTicker(ToolApprovalCleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tam.mu.Lock()
			cutoff := time.Now().Add(-5 * time.Minute)
			for id, a := range tam.approvals {
				if a.Status != ToolApprovalPending && a.CreatedAt.Before(cutoff) {
					delete(tam.approvals, id)
					delete(tam.waiters, id)
				}
			}
			tam.mu.Unlock()
		case <-tam.stopCleanup:
			return
		}
	}
}

func formatToolInput(toolName string, input map[string]interface{}) string {
	switch toolName {
	case "read_file":
		if p, ok := input["path"].(string); ok {
			return fmt.Sprintf("`%s`", p)
		}
	case "write_file", "edit_file":
		if p, ok := input["path"].(string); ok {
			return fmt.Sprintf("`%s`", p)
		}
	case "run_shell_command", "shell":
		if cmd, ok := input["command"].(string); ok {
			return fmt.Sprintf("`%s`", cmd)
		}
	case "list_directory":
		if p, ok := input["path"].(string); ok {
			return fmt.Sprintf("`%s`", p)
		}
	}

	// Fallback: show first key/value
	for k, v := range input {
		return fmt.Sprintf("%s: `%v`", k, v)
	}
	return "(no arguments)"
}
