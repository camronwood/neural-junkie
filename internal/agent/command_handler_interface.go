package agent

import "github.com/camronwood/neural-junkie/internal/protocol"

// CommandHandlerInterface defines the interface for accessing command handler functionality
type CommandHandlerInterface interface {
	AddPendingReview(repoPath string, originalMsg *protocol.Message, agentName string)
	GetPendingReview(repoPath string) *protocol.PendingReview
	RemovePendingReview(repoPath string)
	HasPendingReview(repoPath string) bool
}
