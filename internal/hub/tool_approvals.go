package hub

import (
	"fmt"
	"log"
	"reflect"
	"strings"
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
	waiters   map[string][]chan ToolApprovalStatus // approval ID -> signal channels

	hub         *Hub
	stopCleanup chan struct{}
}

func NewToolApprovalManager(hub *Hub) *ToolApprovalManager {
	tam := &ToolApprovalManager{
		approvals:   make(map[string]*ToolApproval),
		waiters:     make(map[string][]chan ToolApprovalStatus),
		hub:         hub,
		stopCleanup: make(chan struct{}),
	}
	go tam.cleanupLoop()
	return tam
}

func (tam *ToolApprovalManager) Stop() {
	close(tam.stopCleanup)
}

// RestorePending rehydrates a durable approval card after restart. The
// approval remains visible and resolvable even though the original hook
// process is gone; a retried tool call must create a new execution attempt.
func (tam *ToolApprovalManager) RestorePending(approval *ToolApproval) {
	if tam == nil || approval == nil || approval.ID == "" || approval.Status != ToolApprovalPending {
		return
	}
	copyApproval := *approval
	if approval.ToolInput != nil {
		copyApproval.ToolInput = make(map[string]interface{}, len(approval.ToolInput))
		for key, value := range approval.ToolInput {
			copyApproval.ToolInput[key] = value
		}
	}
	tam.mu.Lock()
	if _, exists := tam.approvals[approval.ID]; !exists {
		tam.approvals[approval.ID] = &copyApproval
	}
	tam.mu.Unlock()
}

// CreateApproval registers a new pending tool approval and broadcasts it to the chat.
func (tam *ToolApprovalManager) CreateApproval(agentID, agentName, sessionID, toolName, channel string, toolInput map[string]interface{}) *ToolApproval {
	tam.mu.Lock()
	defer tam.mu.Unlock()

	for _, pending := range tam.approvals {
		if pending != nil && pending.Status == ToolApprovalPending &&
			pending.SessionID == sessionID && pending.ToolName == toolName &&
			pending.Channel == channel && reflect.DeepEqual(pending.ToolInput, toolInput) {
			return pending
		}
	}

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

	// Broadcast to chat so the frontend can render the approval card
	tam.broadcastApproval(approval)
	if tam.hub != nil {
		tam.hub.persistToolApproval(approval, approval.CreatedAt.Add(ToolApprovalTTL))
	}

	return approval
}

// WaitForDecision blocks until the approval is resolved or the timeout expires.
// Returns the final status.
func (tam *ToolApprovalManager) WaitForDecision(approvalID string, timeout time.Duration) (ToolApprovalStatus, string) {
	ch := make(chan ToolApprovalStatus, 1)
	tam.mu.Lock()
	approval, ok := tam.approvals[approvalID]
	if ok && approval.Status == ToolApprovalPending {
		tam.waiters[approvalID] = append(tam.waiters[approvalID], ch)
	}
	tam.mu.Unlock()

	if !ok || approval.Status != ToolApprovalPending {
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
		var waiters []chan ToolApprovalStatus
		if a, exists := tam.approvals[approvalID]; exists && a.Status == ToolApprovalPending {
			now := time.Now()
			a.Status = ToolApprovalExpired
			a.ResolvedAt = &now
			a.Reason = "timed out waiting for user decision"
			waiters = tam.waiters[approvalID]
		}
		delete(tam.waiters, approvalID)
		tam.mu.Unlock()
		for _, waiter := range waiters {
			if waiter != ch {
				waiter <- ToolApprovalExpired
			}
		}
		if tam.hub != nil {
			tam.hub.expireDurableInput(approvalID, "timed out waiting for user decision")
		}
		return ToolApprovalExpired, "timed out waiting for user decision"
	}
}

// Approve resolves a pending approval as approved (once).
func (tam *ToolApprovalManager) Approve(approvalID string) error {
	return tam.ApproveScoped(approvalID, "once")
}

// ApproveScoped resolves a pending approval. scope is "once" or "always".
func (tam *ToolApprovalManager) ApproveScoped(approvalID, scope string) error {
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
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "always" {
		approval.Reason = "always"
	} else {
		approval.Reason = "once"
	}

	if waiters, ok := tam.waiters[approvalID]; ok {
		for _, ch := range waiters {
			ch <- ToolApprovalApproved
		}
		delete(tam.waiters, approvalID)
	}

	tam.broadcastApprovalUpdate(approval)
	if tam.hub != nil {
		tam.hub.resolveDurableInput(approvalID, "user", map[string]any{
			"status": ToolApprovalApproved, "scope": approval.Reason,
		})
	}
	return nil
}

// GetApproval returns a copy of an approval by id, or nil.
func (tam *ToolApprovalManager) GetApproval(approvalID string) *ToolApproval {
	if tam == nil {
		return nil
	}
	tam.mu.Lock()
	defer tam.mu.Unlock()
	a, ok := tam.approvals[approvalID]
	if !ok || a == nil {
		return nil
	}
	copyApproval := *a
	if a.ToolInput != nil {
		copyApproval.ToolInput = make(map[string]interface{}, len(a.ToolInput))
		for k, v := range a.ToolInput {
			copyApproval.ToolInput[k] = v
		}
	}
	return &copyApproval
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

	if waiters, ok := tam.waiters[approvalID]; ok {
		for _, ch := range waiters {
			ch <- ToolApprovalRejected
		}
		delete(tam.waiters, approvalID)
	}

	tam.broadcastApprovalUpdate(approval)
	if tam.hub != nil {
		tam.hub.resolveDurableInput(approvalID, "user", map[string]any{
			"status": ToolApprovalRejected, "reason": reason,
		})
	}
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
	if tam == nil || tam.hub == nil {
		return
	}
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
	if a.ToolName == "run_command" {
		msg.Content = fmt.Sprintf("**%s** wants to run a command that is not on the allowlist: %s", a.AgentName, inputSummary)
		msg.Metadata["allowlist_prompt"] = true
	}

	if err := tam.hub.SendMessage(msg); err != nil {
		log.Printf("[ToolApproval] Failed to broadcast approval %s: %v", a.ID, err)
	}
}

func (tam *ToolApprovalManager) broadcastApprovalUpdate(a *ToolApproval) {
	if tam == nil || tam.hub == nil {
		return
	}
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
			var expired []*ToolApproval
			var expiredWaiters []chan ToolApprovalStatus
			tam.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-5 * time.Minute)
			for id, a := range tam.approvals {
				if a.Status == ToolApprovalPending && now.Sub(a.CreatedAt) > ToolApprovalTTL {
					a.Status = ToolApprovalExpired
					a.ResolvedAt = &now
					a.Reason = "timed out waiting for user decision"
					expired = append(expired, a)
					expiredWaiters = append(expiredWaiters, tam.waiters[id]...)
					delete(tam.waiters, id)
					continue
				}
				if a.Status != ToolApprovalPending && a.CreatedAt.Before(cutoff) {
					delete(tam.approvals, id)
					delete(tam.waiters, id)
				}
			}
			tam.mu.Unlock()
			for _, waiter := range expiredWaiters {
				waiter <- ToolApprovalExpired
			}
			for _, approval := range expired {
				tam.broadcastApprovalUpdate(approval)
				if tam.hub != nil {
					tam.hub.expireDurableInput(approval.ID, approval.Reason)
				}
			}
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
	case "run_shell_command", "shell", "run_command":
		if cmd, ok := input["command"].(string); ok {
			return fmt.Sprintf("`%s`", cmd)
		}
	case "list_directory", "list_dir":
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
