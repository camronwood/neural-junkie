package collaboration

import (
	"fmt"
	"time"
)

// UpdateTaskStatusResult describes side effects after a task status change.
type UpdateTaskStatusResult struct {
	ShouldDispatchWave bool
	ShouldFailRun      bool
	FailRunReason      string
}

// UpdateTaskStatus updates task status and applies blocked-upstream policy side effects.
func (cm *CollaborationManager) UpdateTaskStatusWithEffects(collabID, taskID string, status TaskStatus, output string) (UpdateTaskStatusResult, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return UpdateTaskStatusResult{}, fmt.Errorf("collaboration %s not found", collabID)
	}

	var result UpdateTaskStatusResult
	policy := c.EffectiveExecutionPolicy().BlockedUpstreamPolicy

	for i := range c.Tasks {
		if c.Tasks[i].ID != taskID {
			continue
		}
		c.Tasks[i].Status = status
		if output != "" {
			c.Tasks[i].Output = output
		}
		c.Tasks[i].UpdatedAt = time.Now()
		c.UpdatedAt = time.Now()

		if status == TaskBlocked && policy == BlockedPolicyFailRun {
			result.ShouldFailRun = true
			result.FailRunReason = fmt.Sprintf("task %q is blocked", c.Tasks[i].Title)
		}
		if status == TaskCompleted {
			result.ShouldDispatchWave = c.Phase == PhaseExecuting && !c.DispatchPaused
		}
		if status == TaskBlocked && policy == BlockedPolicySkipBranch {
			cm.markSkippedDownstreamLocked(c, taskID)
		}
		return result, nil
	}
	return UpdateTaskStatusResult{}, fmt.Errorf("task %s not found in collaboration %s", taskID, collabID)
}

func (cm *CollaborationManager) markSkippedDownstreamLocked(c *Collaboration, blockedID string) {
	for i := range c.Tasks {
		for _, dep := range c.Tasks[i].Dependencies {
			if dep == blockedID && c.Tasks[i].Status == TaskPending {
				c.Tasks[i].SkippedDueToBlocked = true
			}
		}
	}
}

// ReassignTask changes assignee and clears dispatch state for re-prompting.
func (cm *CollaborationManager) ReassignTask(collabID, taskID, agentID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	info, err := cm.hub.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent %s not found: %w", agentID, err)
	}
	for i := range c.Tasks {
		if c.Tasks[i].ID == taskID {
			c.Tasks[i].AssignedTo = info.ID
			c.Tasks[i].AssignedName = info.Name
			c.Tasks[i].PromptDispatched = false
			if c.Tasks[i].Status == TaskInProgress {
				c.Tasks[i].Status = TaskPending
			}
			c.Tasks[i].UpdatedAt = time.Now()
			c.UpdatedAt = time.Now()
			return cloneCollaborationMust(c)
		}
	}
	return nil, fmt.Errorf("task %s not found", taskID)
}

// ResetTaskDispatch clears prompt_dispatched so the task can be sent again.
func (cm *CollaborationManager) ResetTaskDispatch(collabID, taskID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	for i := range c.Tasks {
		if c.Tasks[i].ID == taskID {
			c.Tasks[i].PromptDispatched = false
			c.Tasks[i].UpdatedAt = time.Now()
			c.UpdatedAt = time.Now()
			return cloneCollaborationMust(c)
		}
	}
	return nil, fmt.Errorf("task %s not found", taskID)
}

// SkipTask marks a task completed with a skip reason.
func (cm *CollaborationManager) SkipTask(collabID, taskID, reason string) (UpdateTaskStatusResult, error) {
	if reason == "" {
		reason = "skipped by user"
	}
	return cm.UpdateTaskStatusWithEffects(collabID, taskID, TaskCompleted, reason)
}

// SetDispatchPaused toggles whether new task waves may be dispatched.
func (cm *CollaborationManager) SetDispatchPaused(collabID string, paused bool) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	c.DispatchPaused = paused
	c.UpdatedAt = time.Now()
	return cloneCollaborationMust(c)
}

// ApproveTaskDispatch clears awaiting_approval for a task pending user gate.
func (cm *CollaborationManager) ApproveTaskDispatch(collabID, taskID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	for i := range c.Tasks {
		if c.Tasks[i].ID == taskID {
			c.Tasks[i].AwaitingApproval = false
			c.Tasks[i].UpdatedAt = time.Now()
			c.UpdatedAt = time.Now()
			return cloneCollaborationMust(c)
		}
	}
	return nil, fmt.Errorf("task %s not found", taskID)
}

func cloneCollaborationMust(c *Collaboration) (*Collaboration, error) {
	cloned, err := cloneCollaboration(c)
	if err != nil {
		return nil, err
	}
	return cloned, nil
}

// normalizeTaskOnSave applies defaults when persisting tasks.
func normalizeTaskOnSave(t *CollaborationTask) {
	if t.Kind == "" {
		t.Kind = TaskKindAgent
	}
	if t.Options != nil && t.Options.RequiresApproval {
		t.AwaitingApproval = true
	}
}
