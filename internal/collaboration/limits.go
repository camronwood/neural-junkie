package collaboration

import "strings"

// maxTasksLimit returns the effective maximum tasks per collaboration.
func maxTasksLimit() int {
	return HardMaxTasksPerCollaboration
}

// collaborationCountsAsActive reports whether a collaboration consumes a concurrent slot.
// Empty draft runbooks (RB opened but never edited) do not count — avoids blocking real work.
func collaborationCountsAsActive(c *Collaboration) bool {
	if c == nil {
		return false
	}
	if c.Phase == PhaseCompleted || c.Phase == PhaseCancelled {
		return false
	}
	if c.Phase == PhaseDraft && c.Source == SourceRunbook && len(c.Tasks) == 0 {
		return false
	}
	return true
}

// FindReusableEmptyDraftRunbook returns the newest empty draft runbook for the same author, if any.
func (cm *CollaborationManager) FindReusableEmptyDraftRunbook(createdBy string) *Collaboration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	createdBy = strings.TrimSpace(createdBy)
	var best *Collaboration
	for _, c := range cm.collaborations {
		if c == nil || c.Source != SourceRunbook || c.Phase != PhaseDraft {
			continue
		}
		if len(c.Tasks) > 0 {
			continue
		}
		if createdBy != "" && strings.TrimSpace(c.CreatedBy) != createdBy {
			continue
		}
		if best == nil || c.UpdatedAt.After(best.UpdatedAt) {
			best = c
		}
	}
	return best
}
