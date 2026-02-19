package filechange

import (
	"fmt"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

const (
	// FileChangeTTL is how long a file change request remains valid
	FileChangeTTL = 30 * time.Minute
)

// FileChangeManager manages pending file change requests
type FileChangeManager struct {
	mu            sync.RWMutex
	changes       map[string]*FileChange        // changeID -> change
	requests      map[string]*FileChangeRequest // requestID -> request
	executor      *FileChangeExecutor
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewFileChangeManager creates a new file change manager
func NewFileChangeManager(executor *FileChangeExecutor) *FileChangeManager {
	fcm := &FileChangeManager{
		changes:       make(map[string]*FileChange),
		requests:      make(map[string]*FileChangeRequest),
		executor:      executor,
		cleanupTicker: time.NewTicker(1 * time.Minute),
		stopCleanup:   make(chan bool),
	}

	// Start cleanup goroutine
	go fcm.cleanupExpired()

	return fcm
}

// Stop stops the file change manager and cleanup goroutine
func (fcm *FileChangeManager) Stop() {
	fcm.stopCleanup <- true
	fcm.cleanupTicker.Stop()
}

// ProposeFileChange creates a new file change proposal
func (fcm *FileChangeManager) ProposeFileChange(operation FileOperation, filePath, oldPath, newPath, oldContent, newContent string, agent protocol.AgentInfo, channel string) (*FileChange, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	// Generate change ID
	changeID := uuid.New().String()[:8]

	// Validate operation-specific parameters
	if err := fcm.validateFileChange(operation, filePath, oldPath, newPath); err != nil {
		return nil, err
	}

	change := &FileChange{
		ID:          changeID,
		Operation:   operation,
		FilePath:    filePath,
		OldPath:     oldPath,
		NewPath:     newPath,
		OldContent:  oldContent,
		NewContent:  newContent,
		Agent:       agent,
		Channel:     channel,
		Status:      FileChangeStatusPending,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(FileChangeTTL),
		Metadata:    make(map[string]interface{}),
	}

	fcm.changes[changeID] = change
	return change, nil
}

// ProposeFileChangeRequest creates a new file change request with multiple changes
func (fcm *FileChangeManager) ProposeFileChangeRequest(changes []*FileChange, agent protocol.AgentInfo, channel string) (*FileChangeRequest, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	// Generate request ID
	requestID := uuid.New().String()[:8]

	// Validate all changes
	for _, change := range changes {
		if err := fcm.validateFileChange(change.Operation, change.FilePath, change.OldPath, change.NewPath); err != nil {
			return nil, fmt.Errorf("invalid change %s: %w", change.ID, err)
		}
	}

	request := &FileChangeRequest{
		ID:          requestID,
		Changes:     changes,
		Agent:       agent,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(FileChangeTTL),
		Status:      FileChangeStatusPending,
	}

	fcm.requests[requestID] = request

	// Also store individual changes
	for _, change := range changes {
		fcm.changes[change.ID] = change
	}

	return request, nil
}

// GetFileChange retrieves a file change by ID
func (fcm *FileChangeManager) GetFileChange(changeID string) (*FileChange, error) {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	change, ok := fcm.changes[changeID]
	if !ok {
		return nil, fmt.Errorf("file change not found: %s", changeID)
	}

	// Check if expired
	if change.IsExpired() {
		return nil, fmt.Errorf("file change expired")
	}

	return change, nil
}

// ApproveFileChange approves and executes a file change
func (fcm *FileChangeManager) ApproveFileChange(changeID, requestingUserID string) (*FileChange, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	change, ok := fcm.changes[changeID]
	if !ok {
		return nil, fmt.Errorf("file change not found: %s", changeID)
	}

	// Check if expired
	if change.IsExpired() {
		delete(fcm.changes, changeID)
		return nil, fmt.Errorf("file change expired")
	}

	// Check if already processed
	if change.Status != FileChangeStatusPending {
		return nil, fmt.Errorf("file change already processed")
	}

	// Execute the file change
	if err := fcm.executor.ExecuteFileChange(change); err != nil {
		return nil, fmt.Errorf("failed to execute file change: %w", err)
	}

	// Update status
	now := time.Now()
	change.Status = FileChangeStatusApproved
	change.ApprovedAt = &now

	return change, nil
}

// RejectFileChange rejects a file change
func (fcm *FileChangeManager) RejectFileChange(changeID, requestingUserID, reason string) (*FileChange, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	change, ok := fcm.changes[changeID]
	if !ok {
		return nil, fmt.Errorf("file change not found: %s", changeID)
	}

	// Check if expired
	if change.IsExpired() {
		delete(fcm.changes, changeID)
		return nil, fmt.Errorf("file change expired")
	}

	// Check if already processed
	if change.Status != FileChangeStatusPending {
		return nil, fmt.Errorf("file change already processed")
	}

	// Update status
	now := time.Now()
	change.Status = FileChangeStatusRejected
	change.RejectedAt = &now
	change.Reason = reason

	return change, nil
}

// ListPendingFileChanges returns all pending file changes for a user
func (fcm *FileChangeManager) ListPendingFileChanges(userID string) []*FileChange {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	var pending []*FileChange
	now := time.Now()

	for _, change := range fcm.changes {
		if change.Status == FileChangeStatusPending && now.Before(change.ExpiresAt) {
			pending = append(pending, change)
		}
	}

	return pending
}

// ListAllFileChanges returns all file changes (for admin/debug purposes)
func (fcm *FileChangeManager) ListAllFileChanges() []*FileChange {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	var all []*FileChange
	for _, change := range fcm.changes {
		all = append(all, change)
	}

	return all
}

// GetExecutor returns the file change executor for external access.
func (fcm *FileChangeManager) GetExecutor() *FileChangeExecutor {
	return fcm.executor
}

// GetPendingCount returns the number of pending file changes
func (fcm *FileChangeManager) GetPendingCount() int {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	count := 0
	now := time.Now()

	for _, change := range fcm.changes {
		if change.Status == FileChangeStatusPending && now.Before(change.ExpiresAt) {
			count++
		}
	}

	return count
}

// validateFileChange validates a file change based on operation type
func (fcm *FileChangeManager) validateFileChange(operation FileOperation, filePath, oldPath, newPath string) error {
	// Basic path validation
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Operation-specific validation
	switch operation {
	case FileOperationCreate:
		if filePath == "" {
			return fmt.Errorf("file path required for create operation")
		}
	case FileOperationEdit:
		if filePath == "" {
			return fmt.Errorf("file path required for edit operation")
		}
	case FileOperationDelete:
		if filePath == "" {
			return fmt.Errorf("file path required for delete operation")
		}
	case FileOperationMove:
		if oldPath == "" || newPath == "" {
			return fmt.Errorf("both old and new paths required for move operation")
		}
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}

// cleanupExpired removes expired file changes
func (fcm *FileChangeManager) cleanupExpired() {
	for {
		select {
		case <-fcm.cleanupTicker.C:
			fcm.doCleanup()
		case <-fcm.stopCleanup:
			return
		}
	}
}

// doCleanup performs the actual cleanup
func (fcm *FileChangeManager) doCleanup() {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	now := time.Now()
	expiredChanges := []string{}
	expiredRequests := []string{}

	// Find expired changes
	for id, change := range fcm.changes {
		if now.After(change.ExpiresAt) {
			expiredChanges = append(expiredChanges, id)
		}
	}

	// Find expired requests
	for id, request := range fcm.requests {
		if now.After(request.ExpiresAt) {
			expiredRequests = append(expiredRequests, id)
		}
	}

	// Remove expired changes
	for _, id := range expiredChanges {
		delete(fcm.changes, id)
	}

	// Remove expired requests
	for _, id := range expiredRequests {
		delete(fcm.requests, id)
	}
}
