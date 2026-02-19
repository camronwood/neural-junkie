package filechange

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// FileOperation defines the type of file operation
type FileOperation string

const (
	FileOperationCreate FileOperation = "create"
	FileOperationEdit   FileOperation = "edit"
	FileOperationDelete FileOperation = "delete"
	FileOperationMove   FileOperation = "move"
)

// FileChangeStatus defines the status of a file change
type FileChangeStatus string

const (
	FileChangeStatusPending  FileChangeStatus = "pending"
	FileChangeStatusApproved FileChangeStatus = "approved"
	FileChangeStatusRejected FileChangeStatus = "rejected"
	FileChangeStatusExpired  FileChangeStatus = "expired"
)

// FileChange represents a proposed file change
type FileChange struct {
	ID          string                 `json:"id"`
	Operation   FileOperation          `json:"operation"`
	FilePath    string                 `json:"file_path"`
	OldPath     string                 `json:"old_path,omitempty"`    // For move operations
	NewPath     string                 `json:"new_path,omitempty"`    // For move operations
	OldContent  string                 `json:"old_content,omitempty"` // For edit operations
	NewContent  string                 `json:"new_content,omitempty"` // For create/edit operations
	Agent       protocol.AgentInfo     `json:"agent"`
	Channel     string                 `json:"channel"`
	Status      FileChangeStatus       `json:"status"`
	RequestedAt time.Time              `json:"requested_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	ApprovedAt  *time.Time             `json:"approved_at,omitempty"`
	RejectedAt  *time.Time             `json:"rejected_at,omitempty"`
	Reason      string                 `json:"reason,omitempty"` // Reason for rejection
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FileChangeRequest represents a collection of file changes proposed together
type FileChangeRequest struct {
	ID          string             `json:"id"`
	Changes     []*FileChange      `json:"changes"`
	Agent       protocol.AgentInfo `json:"agent"`
	Channel     string             `json:"channel"`
	RequestedAt time.Time          `json:"requested_at"`
	ExpiresAt   time.Time          `json:"expires_at"`
	Status      FileChangeStatus   `json:"status"`
}

// IsExpired checks if the file change has expired
func (fc *FileChange) IsExpired() bool {
	return time.Now().After(fc.ExpiresAt)
}

// IsDeleteOperation checks if this is a delete operation
func (fc *FileChange) IsDeleteOperation() bool {
	return fc.Operation == FileOperationDelete
}

// IsMoveOperation checks if this is a move operation
func (fc *FileChange) IsMoveOperation() bool {
	return fc.Operation == FileOperationMove
}

// GetDisplayPath returns the appropriate path for display
func (fc *FileChange) GetDisplayPath() string {
	if fc.IsMoveOperation() {
		return fc.OldPath + " → " + fc.NewPath
	}
	return fc.FilePath
}

// GetTimeRemaining returns the time remaining before expiration
func (fc *FileChange) GetTimeRemaining() time.Duration {
	if fc.IsExpired() {
		return 0
	}
	return time.Until(fc.ExpiresAt)
}

// IsExpired checks if the file change request has expired
func (fcr *FileChangeRequest) IsExpired() bool {
	return time.Now().After(fcr.ExpiresAt)
}

// GetTimeRemaining returns the time remaining before expiration
func (fcr *FileChangeRequest) GetTimeRemaining() time.Duration {
	if fcr.IsExpired() {
		return 0
	}
	return time.Until(fcr.ExpiresAt)
}
