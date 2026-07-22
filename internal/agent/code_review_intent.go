package agent

import (
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// userRequestsCodeReview reports read-only review/audit asks (whole project or codebase).
// Prefer userRequestsCodeReviewForMessage when a stamped turn decision may be present.
func userRequestsCodeReview(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	phrases := []string{
		"code review", "review this project", "review the project", "review this codebase",
		"review the codebase", "review this repo", "review the repository", "review this app",
		"review the app", "audit the code", "audit this project", "audit the codebase",
		"review the code in the workspace", "review the code in workspace",
		"review code in the workspace", "review the workspace", "review this workspace",
	}
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	if strings.Contains(lower, "review") {
		for _, scope := range []string{
			"project", "codebase", "repository", " repo", "whole app", "this app", "the app",
			"workspace", "in the workspace",
		} {
			if strings.Contains(lower, scope) {
				return true
			}
		}
	}
	return false
}

// userRequestsCodeReviewForMessage uses a stamped decision when present; phrases are
// emergency rollback only when no canonical decision is stamped.
func userRequestsCodeReviewForMessage(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		return decision.Domain == "code_review" || decision.RecipientType == "code-review"
	}
	return userRequestsCodeReview(msg.Content)
}

func repoPathFromMessage(content string) string {
	result := protocol.DetectLocalPaths(content)
	if !result.Found {
		return ""
	}
	best := protocol.GetBestPathForRepository(result.Paths)
	if best == nil {
		return ""
	}
	return normalizeRepoPath(best.Path)
}

// messageDefersToRepoExpert reports when a repo-path review should be handled by a repo agent, not pack specialists.
func messageDefersToRepoExpert(a *Agent, msg *protocol.Message) bool {
	if a == nil || msg == nil || a.Info.Type == protocol.AgentTypeRepo {
		return false
	}
	if !userRequestsCodeReviewForMessage(msg) {
		return false
	}
	repoPath := repoPathFromMessage(msg.Content)
	if repoPath == "" {
		return false
	}
	if ch := a.Hub.GetCommandHandler(); ch != nil && ch.HasPendingReview(repoPath) {
		return true
	}
	agents, _ := a.Hub.GetChannelAgents(msg.Channel)
	for _, ag := range agents {
		if ag.Type == protocol.AgentTypeRepo && normalizeRepoPath(ag.RepositoryPath) == repoPath {
			return true
		}
	}
	return false
}

func normalizeRepoPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}
