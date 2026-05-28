package collaboration

import "strings"

// ShouldAutoAckWorkspaceOnApprove reports whether the hub may acknowledge the
// execution workspace immediately after plan approval (sandbox + bound project repo).
// Worktree runs and research-only sandboxes still require explicit user confirmation.
func ShouldAutoAckWorkspaceOnApprove(c *Collaboration) bool {
	if c == nil {
		return false
	}
	if c.ExecutionMode == ExecutionModeWorktree {
		return false
	}
	return strings.TrimSpace(c.SourceRepoPath) != ""
}
