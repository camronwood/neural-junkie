package collaboration

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// FinalizeOptions controls how a collaboration is closed.
type FinalizeOptions struct {
	// MarkOpenTasksComplete sets pending/in_progress/blocked tasks to completed.
	MarkOpenTasksComplete bool
}

// HasOpenTasks reports whether any task is not completed.
func HasOpenTasks(c *Collaboration) bool {
	if c == nil {
		return false
	}
	for _, t := range c.Tasks {
		if t.Status != TaskCompleted {
			return true
		}
	}
	return false
}

// OpenTaskTitles returns titles of non-completed tasks.
func OpenTaskTitles(c *Collaboration) []string {
	if c == nil {
		return nil
	}
	var out []string
	for _, t := range c.Tasks {
		if t.Status != TaskCompleted {
			out = append(out, t.Title)
		}
	}
	return out
}

// FinalizeCollaboration moves a collaboration to completed and stops discussion.
// Idempotent when already completed; cancelled collaborations are not reopened.
func (cm *CollaborationManager) FinalizeCollaboration(collabID string, opts FinalizeOptions) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase == PhaseCancelled {
		return nil, fmt.Errorf("collaboration is cancelled; start a new session with /collaborate")
	}
	if c.Phase == PhaseCompleted {
		return c, nil
	}

	now := time.Now()
	if opts.MarkOpenTasksComplete {
		for i := range c.Tasks {
			if c.Tasks[i].Status != TaskCompleted {
				c.Tasks[i].Status = TaskCompleted
				if c.Tasks[i].Output == "" {
					c.Tasks[i].Output = "Closed by user"
				}
				c.Tasks[i].UpdatedAt = now
			}
		}
	}

	c.Phase = PhaseCompleted
	c.UpdatedAt = now
	if c.Discussion != nil {
		c.Discussion.Status = DiscussionConverged
	}

	log.Printf("[CollaborationManager] Collaboration %s finalized (force_tasks=%v)", collabID[:8], opts.MarkOpenTasksComplete)
	return c, nil
}

// CompleteCollaboration marks a collaboration finished when all tasks are already done.
func (cm *CollaborationManager) CompleteCollaboration(collabID string) (*Collaboration, error) {
	return cm.FinalizeCollaboration(collabID, FinalizeOptions{MarkOpenTasksComplete: false})
}

// ResolveTaskIndex resolves a 1-based task index or task ID prefix to a task ID.
func (cm *CollaborationManager) ResolveTaskIndex(collabID, ref string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return "", fmt.Errorf("collaboration %s not found", collabID)
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("task reference required")
	}

	if n, err := strconv.Atoi(ref); err == nil && n > 0 && n <= len(c.Tasks) {
		return c.Tasks[n-1].ID, nil
	}

	for _, t := range c.Tasks {
		if t.ID == ref || strings.HasPrefix(t.ID, ref) {
			return t.ID, nil
		}
	}
	return "", fmt.Errorf("task %q not found in collaboration %s", ref, collabID[:8])
}
