package dispatch

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// ApprovalTTL is how long an approval request remains valid
	ApprovalTTL = 5 * time.Minute
)

// ApprovalManager manages pending approval requests
type ApprovalManager struct {
	mu            sync.RWMutex
	approvals     map[string]*PendingApproval // approvalID -> approval
	executor      *Executor
	registry      *CommandRegistry
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewApprovalManager creates a new approval manager
func NewApprovalManager(executor *Executor, registry *CommandRegistry) *ApprovalManager {
	am := &ApprovalManager{
		approvals:     make(map[string]*PendingApproval),
		executor:      executor,
		registry:      registry,
		cleanupTicker: time.NewTicker(1 * time.Minute),
		stopCleanup:   make(chan bool),
	}

	// Start cleanup goroutine
	go am.cleanupExpired()

	return am
}

// Stop stops the approval manager and cleanup goroutine
func (am *ApprovalManager) Stop() {
	am.stopCleanup <- true
	am.cleanupTicker.Stop()
}

// RequestApproval creates a new approval request
func (am *ApprovalManager) RequestApproval(userID, username, channel, plugin, subCmd string, args []string) (*PendingApproval, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate approval ID
	approvalID := uuid.New().String()[:8] // Short ID for easier typing

	// Build full command string
	command := fmt.Sprintf("dispatch %s %s", plugin, subCmd)
	if len(args) > 0 {
		for _, arg := range args {
			command = fmt.Sprintf("%s %s", command, arg)
		}
	}

	approval := &PendingApproval{
		ID:          approvalID,
		UserID:      userID,
		Username:    username,
		Channel:     channel,
		Command:     command,
		Plugin:      plugin,
		SubCmd:      subCmd,
		Args:        args,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(ApprovalTTL),
	}

	am.approvals[approvalID] = approval

	return approval, nil
}

// GetApproval retrieves a pending approval by ID
func (am *ApprovalManager) GetApproval(approvalID string) (*PendingApproval, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	approval, ok := am.approvals[approvalID]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", approvalID)
	}

	// Check if expired
	if time.Now().After(approval.ExpiresAt) {
		return nil, fmt.Errorf("approval request expired")
	}

	return approval, nil
}

// ApproveCommand approves and executes a command
func (am *ApprovalManager) ApproveCommand(approvalID, requestingUserID string) (*PendingApproval, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	approval, ok := am.approvals[approvalID]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", approvalID)
	}

	// Check if expired
	if time.Now().After(approval.ExpiresAt) {
		delete(am.approvals, approvalID)
		return nil, fmt.Errorf("approval request expired")
	}

	// Verify requesting user matches (users can only approve their own requests)
	if approval.UserID != requestingUserID {
		return nil, fmt.Errorf("only the requesting user can approve this command")
	}

	// Remove from pending approvals
	delete(am.approvals, approvalID)

	return approval, nil
}

// RejectCommand rejects a pending command
func (am *ApprovalManager) RejectCommand(approvalID, requestingUserID string) (*PendingApproval, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	approval, ok := am.approvals[approvalID]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", approvalID)
	}

	// Verify requesting user matches
	if approval.UserID != requestingUserID {
		return nil, fmt.Errorf("only the requesting user can reject this command")
	}

	// Remove from pending approvals
	delete(am.approvals, approvalID)

	return approval, nil
}

// ListPendingApprovals returns all pending approvals for a user
func (am *ApprovalManager) ListPendingApprovals(userID string) []*PendingApproval {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var pending []*PendingApproval
	now := time.Now()

	for _, approval := range am.approvals {
		if approval.UserID == userID && now.Before(approval.ExpiresAt) {
			pending = append(pending, approval)
		}
	}

	return pending
}

// cleanupExpired removes expired approval requests
func (am *ApprovalManager) cleanupExpired() {
	for {
		select {
		case <-am.cleanupTicker.C:
			am.doCleanup()
		case <-am.stopCleanup:
			return
		}
	}
}

// doCleanup performs the actual cleanup
func (am *ApprovalManager) doCleanup() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	expired := []string{}

	for id, approval := range am.approvals {
		if now.After(approval.ExpiresAt) {
			expired = append(expired, id)
		}
	}

	// Remove expired approvals
	for _, id := range expired {
		delete(am.approvals, id)
	}
}

// GetPendingCount returns the number of pending approvals
func (am *ApprovalManager) GetPendingCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	count := 0
	now := time.Now()

	for _, approval := range am.approvals {
		if now.Before(approval.ExpiresAt) {
			count++
		}
	}

	return count
}
