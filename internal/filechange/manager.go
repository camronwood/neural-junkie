package filechange

import (
	"fmt"
	"sort"
	"strings"
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
	executors     map[string]*FileChangeExecutor // immutable execution context per change
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewFileChangeManager creates a new file change manager
func NewFileChangeManager(executor *FileChangeExecutor) *FileChangeManager {
	fcm := &FileChangeManager{
		changes:       make(map[string]*FileChange),
		requests:      make(map[string]*FileChangeRequest),
		executor:      executor,
		executors:     make(map[string]*FileChangeExecutor),
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

// RestorePending rehydrates a durable file proposal after restart.
func (fcm *FileChangeManager) RestorePending(change *FileChange) {
	if fcm == nil || change == nil || change.ID == "" || change.Status != FileChangeStatusPending {
		return
	}
	copyChange := *change
	if change.Metadata != nil {
		copyChange.Metadata = make(map[string]interface{}, len(change.Metadata))
		for key, value := range change.Metadata {
			copyChange.Metadata[key] = value
		}
	}
	fcm.mu.Lock()
	if _, exists := fcm.changes[change.ID]; !exists {
		fcm.changes[change.ID] = &copyChange
	}
	fcm.mu.Unlock()
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
		NewContent:  SanitizeFileChangeContent(newContent),
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

	if len(changes) == 0 {
		return nil, fmt.Errorf("batch request requires at least one change")
	}

	requestID := uuid.New().String()[:8]
	now := time.Now()
	expires := now.Add(FileChangeTTL)
	normalized := make([]*FileChange, 0, len(changes))

	for i, change := range changes {
		if change == nil {
			return nil, fmt.Errorf("invalid change at index %d: nil", i)
		}
		if err := fcm.validateFileChange(change.Operation, change.FilePath, change.OldPath, change.NewPath); err != nil {
			return nil, fmt.Errorf("invalid change %s: %w", change.ID, err)
		}
		copyChange := *change
		if strings.TrimSpace(copyChange.ID) == "" {
			copyChange.ID = uuid.New().String()[:8]
		}
		copyChange.Agent = agent
		copyChange.Channel = channel
		copyChange.Status = FileChangeStatusPending
		copyChange.RequestedAt = now
		copyChange.ExpiresAt = expires
		copyChange.NewContent = SanitizeFileChangeContent(copyChange.NewContent)
		if copyChange.Metadata == nil {
			copyChange.Metadata = make(map[string]interface{})
		}
		copyChange.Metadata["request_id"] = requestID
		normalized = append(normalized, &copyChange)
	}

	request := &FileChangeRequest{
		ID:          requestID,
		Changes:     normalized,
		Agent:       agent,
		Channel:     channel,
		RequestedAt: now,
		ExpiresAt:   expires,
		Status:      FileChangeStatusPending,
	}

	fcm.requests[requestID] = request
	for _, change := range normalized {
		fcm.changes[change.ID] = change
	}

	return request, nil
}

// GetFileChangeRequest retrieves a batch request by ID.
func (fcm *FileChangeManager) GetFileChangeRequest(requestID string) (*FileChangeRequest, error) {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()
	req, ok := fcm.requests[requestID]
	if !ok {
		return nil, fmt.Errorf("file change request not found: %s", requestID)
	}
	if req.IsExpired() && req.Status == FileChangeStatusPending {
		return nil, fmt.Errorf("file change request expired")
	}
	return req, nil
}

// ApproveFileChangeRequest approves and executes every pending member of a batch.
// On mid-batch failure, remaining pending members are marked failed.
func (fcm *FileChangeManager) ApproveFileChangeRequest(requestID, requestingUserID string) (*FileChangeRequest, error) {
	fcm.mu.RLock()
	req, ok := fcm.requests[requestID]
	if !ok {
		fcm.mu.RUnlock()
		return nil, fmt.Errorf("file change request not found: %s", requestID)
	}
	if req.IsExpired() && req.Status == FileChangeStatusPending {
		fcm.mu.RUnlock()
		return nil, fmt.Errorf("file change request expired")
	}
	if req.Status != FileChangeStatusPending {
		fcm.mu.RUnlock()
		return nil, fmt.Errorf("file change request already processed")
	}
	ids := make([]string, 0, len(req.Changes))
	for _, c := range req.Changes {
		if c != nil {
			ids = append(ids, c.ID)
		}
	}
	fcm.mu.RUnlock()

	var firstErr error
	applied := 0
	for _, id := range ids {
		if firstErr != nil {
			_, _ = fcm.MarkFileChangeStatus(id, FileChangeStatusFailed, "batch aborted: "+firstErr.Error())
			continue
		}
		change, err := fcm.GetFileChangeRecord(id)
		if err != nil {
			firstErr = err
			_, _ = fcm.MarkFileChangeStatus(id, FileChangeStatusFailed, err.Error())
			continue
		}
		if change.Status != FileChangeStatusPending {
			continue
		}
		if _, err := fcm.ApproveFileChange(id, requestingUserID); err != nil {
			firstErr = err
			continue
		}
		applied++
	}

	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	req = fcm.requests[requestID]
	if req == nil {
		return nil, fmt.Errorf("file change request not found: %s", requestID)
	}
	if firstErr != nil {
		req.Status = FileChangeStatusFailed
		return req, fmt.Errorf("batch apply failed after %d success(es): %w", applied, firstErr)
	}
	now := time.Now()
	req.Status = FileChangeStatusApproved
	for _, c := range req.Changes {
		if c != nil && c.ApprovedAt == nil {
			c.ApprovedAt = &now
		}
	}
	return req, nil
}

// RejectFileChangeRequest rejects every pending member of a batch.
func (fcm *FileChangeManager) RejectFileChangeRequest(requestID, requestingUserID, reason string) (*FileChangeRequest, error) {
	fcm.mu.RLock()
	req, ok := fcm.requests[requestID]
	if !ok {
		fcm.mu.RUnlock()
		return nil, fmt.Errorf("file change request not found: %s", requestID)
	}
	if req.Status != FileChangeStatusPending {
		fcm.mu.RUnlock()
		return nil, fmt.Errorf("file change request already processed")
	}
	ids := make([]string, 0, len(req.Changes))
	for _, c := range req.Changes {
		if c != nil {
			ids = append(ids, c.ID)
		}
	}
	fcm.mu.RUnlock()

	for _, id := range ids {
		change, err := fcm.GetFileChangeRecord(id)
		if err != nil || change.Status != FileChangeStatusPending {
			continue
		}
		_, _ = fcm.RejectFileChange(id, requestingUserID, reason)
	}

	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	req = fcm.requests[requestID]
	if req == nil {
		return nil, fmt.Errorf("file change request not found: %s", requestID)
	}
	req.Status = FileChangeStatusRejected
	return req, nil
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

// GetFileChangeRecord retrieves lifecycle metadata even when a proposal expired.
func (fcm *FileChangeManager) GetFileChangeRecord(changeID string) (*FileChange, error) {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()
	change, ok := fcm.changes[changeID]
	if !ok {
		return nil, fmt.Errorf("file change not found: %s", changeID)
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
		change.Status = FileChangeStatusExpired
		change.Reason = "file change expired"
		return nil, fmt.Errorf("file change expired")
	}

	// Check if already processed
	if change.Status != FileChangeStatusPending {
		return nil, fmt.Errorf("file change already processed")
	}

	executor := fcm.executor
	if bound := fcm.executors[changeID]; bound != nil {
		executor = bound
	}
	if change.Operation == FileOperationEdit {
		current, err := executor.GetFileContent(change.FilePath)
		if err != nil {
			return nil, fmt.Errorf("read current file before approval: %w", err)
		}
		if SanitizeFileChangeContent(current) != SanitizeFileChangeContent(change.OldContent) {
			change.Status = FileChangeStatusStale
			change.Reason = fmt.Sprintf("stale edit rejected: %s changed after proposal", change.FilePath)
			return nil, fmt.Errorf("stale edit rejected: %s changed after proposal", change.FilePath)
		}
		if SanitizeFileChangeContent(current) == SanitizeFileChangeContent(change.NewContent) {
			change.Status = FileChangeStatusStale
			change.Reason = fmt.Sprintf("no-op edit rejected: %s already has proposed content", change.FilePath)
			return nil, fmt.Errorf("no-op edit rejected: %s already has proposed content", change.FilePath)
		}
	}

	// Execute the file change with the workspace/backend captured at registration.
	if err := executor.ExecuteFileChange(change); err != nil {
		change.Status = FileChangeStatusFailed
		change.Reason = err.Error()
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
		change.Status = FileChangeStatusExpired
		change.Reason = "file change expired"
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

// UpdatePendingNewContent updates NewContent on a pending edit proposal (e.g. partial hunk accept).
func (fcm *FileChangeManager) UpdatePendingNewContent(changeID, newContent string) (*FileChange, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	change, ok := fcm.changes[changeID]
	if !ok {
		return nil, fmt.Errorf("file change not found: %s", changeID)
	}
	if change.IsExpired() {
		change.Status = FileChangeStatusExpired
		change.Reason = "file change expired"
		return nil, fmt.Errorf("file change expired")
	}
	if change.Status != FileChangeStatusPending {
		return nil, fmt.Errorf("file change already processed")
	}
	if change.Operation != FileOperationEdit && change.Operation != FileOperationCreate {
		return nil, fmt.Errorf("only create/edit proposals support content updates")
	}
	change.NewContent = SanitizeFileChangeContent(newContent)
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
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].RequestedAt.Before(pending[j].RequestedAt)
	})

	return pending
}

// MarkFileChangeStatus records a terminal status after an approval attempt fails.
func (fcm *FileChangeManager) MarkFileChangeStatus(changeID string, status FileChangeStatus, reason string) (*FileChange, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	change, ok := fcm.changes[changeID]
	if !ok {
		return nil, fmt.Errorf("file change not found: %s", changeID)
	}
	change.Status = status
	change.Reason = reason
	now := time.Now()
	if status == FileChangeStatusRejected {
		change.RejectedAt = &now
	}
	return change, nil
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

// BindExecutionContext stores an immutable workspace/backend executor for a change.
// This prevents proposals from different workspaces retargeting a shared executor.
func (fcm *FileChangeManager) BindExecutionContext(changeID, workspaceRoot string, workspaceIO WorkspaceIO) error {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	if _, ok := fcm.changes[changeID]; !ok {
		return fmt.Errorf("file change not found: %s", changeID)
	}
	executor := NewFileChangeExecutor(workspaceRoot)
	executor.SetWorkspaceIO(workspaceIO)
	fcm.executors[changeID] = executor
	return nil
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
		delete(fcm.executors, id)
	}

	// Remove expired requests
	for _, id := range expiredRequests {
		delete(fcm.requests, id)
	}
}
