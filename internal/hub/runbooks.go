package hub

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// RunbookCreateRequest is the JSON body for POST /api/runbooks.
type RunbookCreateRequest struct {
	Description    string                         `json:"description"`
	AgentIDs       []string                       `json:"agent_ids"`
	Channel        string                         `json:"channel"`
	CreatedBy      string                         `json:"created_by"`
	Tasks          []collaboration.CollaborationTask `json:"tasks,omitempty"`
	ExecutionMode  string                         `json:"execution_mode,omitempty"`
	SourceRepoPath string                         `json:"source_repo_path,omitempty"`
}

// RunbookCreateResult is returned when a runbook is created.
type RunbookCreateResult struct {
	CollaborationID    string `json:"collaboration_id"`
	CollaborationChannel string `json:"collaboration_channel"`
}

// CreateRunbookSession creates a draft runbook and binds a collab channel.
func (h *Hub) CreateRunbookSession(req RunbookCreateRequest) (*RunbookCreateResult, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		return nil, fmt.Errorf("description is required")
	}
	if len(req.AgentIDs) < 1 {
		return nil, fmt.Errorf("at least one agent_id is required")
	}
	ch := strings.TrimSpace(req.Channel)
	if ch == "" {
		ch = "general"
	}
	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = "user"
	}

	opts := collaboration.CreateOptions{InitialTasks: req.Tasks}
	switch strings.TrimSpace(req.ExecutionMode) {
	case string(collaboration.ExecutionModeWorktree):
		opts.ExecutionMode = collaboration.ExecutionModeWorktree
		opts.SourceRepoPath = strings.TrimSpace(req.SourceRepoPath)
	}

	collab, err := h.collabManager.CreateRunbook(desc, req.AgentIDs, ch, createdBy, collaboration.DiscussionConfig{}, opts)
	if err != nil {
		return nil, err
	}

	collabChannelName := "collab-" + collab.ID
	h.CreateChannelWithType(
		collabChannelName,
		collab.Title,
		ch,
		protocol.ChannelTypeCollaboration,
		createdBy,
	)
	if err := h.collabManager.BindCollaborationChannel(collab.ID, collabChannelName); err != nil {
		return nil, err
	}

	for _, id := range req.AgentIDs {
		_ = h.AddAgentToChannel(id, collabChannelName)
	}

	return &RunbookCreateResult{
		CollaborationID:      collab.ID,
		CollaborationChannel: collabChannelName,
	}, nil
}

// GetRunbookSnapshot returns a collaboration by id.
func (h *Hub) GetRunbookSnapshot(collabID string) (*collaboration.Collaboration, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	return h.collabManager.GetCollaborationSnapshot(collabID)
}

// UpdateRunbookSession updates draft/reviewing runbook fields.
func (h *Hub) UpdateRunbookSession(collabID string, payload collaboration.RunbookUpdatePayload) (*collaboration.Collaboration, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	return h.collabManager.UpdateRunbook(collabID, payload)
}

// SubmitRunbookForReview moves draft → reviewing.
func (h *Hub) SubmitRunbookForReview(collabID string) (*collaboration.Collaboration, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	return h.collabManager.SubmitRunbook(collabID)
}

// StartRunbook approves and transitions to executing (same as /approve-plan).
func (h *Hub) StartRunbook(collabID string) (*collaboration.Collaboration, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	c, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil {
		return nil, err
	}
	if c.Source != collaboration.SourceRunbook {
		return nil, fmt.Errorf("not a runbook collaboration")
	}
	if c.Phase != collaboration.PhaseReviewing && c.Phase != collaboration.PhaseApproved {
		return nil, fmt.Errorf("runbook must be in reviewing phase to start (current: %s)", c.Phase)
	}
	if _, err := h.collabManager.ApprovePlan(collabID); err != nil {
		return nil, err
	}
	if len(c.Tasks) == 0 {
		snap, _ := h.collabManager.GetCollaborationSnapshot(collabID)
		if snap != nil && len(snap.Tasks) > 0 {
			// tasks already set
		}
	}
	if _, err := h.collabManager.TransitionToExecuting(collabID); err != nil {
		return nil, err
	}
	if _, err := h.collabManager.EnsureExecutionTasks(collabID); err != nil {
		log.Printf("[Runbook] EnsureExecutionTasks for %s: %v", shortCollabID(collabID), err)
	}
	return h.finalizeCollaborationExecutionStart(collabID, "✅ **Runbook execution started**")
}

// finalizeCollaborationExecutionStart auto-acks when allowed, dispatches ready tasks,
// and posts execution status to the collaboration channel (mirrors /approve-plan).
func (h *Hub) finalizeCollaborationExecutionStart(collabID, heading string) (*collaboration.Collaboration, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	collabSnap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || collabSnap == nil {
		return nil, fmt.Errorf("could not load collaboration %s after execution start", shortCollabID(collabID))
	}
	h.persistCollaborationReviewAssets(collabID)

	var autoAckErr error
	if collaboration.ShouldAutoAckWorkspaceOnApprove(collabSnap) && !collabSnap.WorkspaceAcknowledged {
		if err := h.AcknowledgeCollaborationWorkspace(collabID, ""); err != nil {
			log.Printf("[Collaboration] Auto workspace ack for %s: %v", shortCollabID(collabID), err)
			autoAckErr = err
		} else {
			collabSnap, err = h.collabManager.GetCollaborationSnapshot(collabID)
			if err != nil || collabSnap == nil {
				return nil, fmt.Errorf("could not reload collaboration %s after workspace ack", shortCollabID(collabID))
			}
		}
	}

	pathWarnings := collaboration.FormatTaskPathWarnings(
		collaboration.ValidateCollaborationPaths(collabSnap),
		collabSnap.SourceRepoPath,
	)

	var taskSummary strings.Builder
	taskSummary.WriteString(fmt.Sprintf("%s (Collaboration `%s`)\n\n", heading, collabID[:8]))
	taskSummary.WriteString("**Assigned Tasks:**\n\n")
	if len(collabSnap.Tasks) == 0 {
		taskSummary.WriteString("_No tasks to assign (no participants)._\n\n")
	}
	for i, task := range collabSnap.Tasks {
		assigneeLabel := task.AssignedName
		if assigneeLabel == "" {
			assigneeLabel = "unassigned"
		}
		taskSummary.WriteString(fmt.Sprintf("⬜ **Task %d:** %s\n   Assigned to: **@%s**\n\n", i+1, task.Description, assigneeLabel))
	}

	if h.CollaborationCanDispatchTasks(collabSnap) {
		h.DispatchReadyCollabTasksForSnapshot(collabSnap, false)
		if collaboration.ShouldAutoAckWorkspaceOnApprove(collabSnap) && collabSnap.WorkspaceAcknowledged {
			taskSummary.WriteString("\n**Tasks dispatched** — workspace was auto-confirmed (bound project repo).\n")
		} else if collabSnap.WorkspaceAcknowledged {
			taskSummary.WriteString("\n**Tasks dispatched** to assignees.\n")
		}
	} else {
		if autoAckErr != nil {
			taskSummary.WriteString(fmt.Sprintf(
				"\n⚠️ **Auto workspace confirmation failed** — %v. Use **Continue** or `/ack-collab-workspace %s` before tasks can run.\n",
				autoAckErr, collabID[:8],
			))
		}
		if collabSnap.ExecutionMode == collaboration.ExecutionModeWorktree {
			if strings.TrimSpace(collabSnap.WorkingDirectory) != "" {
				taskSummary.WriteString(fmt.Sprintf("\n**Git worktree:** `%s` (branch `%s`)\n", collabSnap.WorkingDirectory, collabSnap.WorktreeBranch))
			} else {
				taskSummary.WriteString("\n**Git worktree:** will be created from your active workspace when you confirm.\n")
			}
		}
		chName := strings.TrimSpace(collabSnap.Channel)
		if chName == "" {
			chName = "the collaboration channel"
		} else {
			chName = "#" + chName
		}
		taskSummary.WriteString(fmt.Sprintf("\n⏸ **Waiting for workspace confirmation** — agents will receive their task prompts after you click **Continue** on %s in the desktop app, or run `/ack-collab-workspace %s` here.\n", chName, collabID[:8]))
		if collabSnap.ExecutionMode == collaboration.ExecutionModeWorktree && collabSnap.WorktreeBranch != "" && strings.TrimSpace(collabSnap.WorkingDirectory) != "" {
			taskSummary.WriteString(fmt.Sprintf("_After completion, merge branch `%s` from your main checkout._\n", collabSnap.WorktreeBranch))
		}
	}
	if pathWarnings != "" {
		taskSummary.WriteString(pathWarnings)
	}

	ch := strings.TrimSpace(collabSnap.Channel)
	if ch != "" {
		statusMsg := protocol.NewMessage(
			protocol.MessageTypeCollabStatus,
			ch,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			taskSummary.String(),
		)
		statusMsg.SetCollaborationID(collabID)
		statusMsg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
		if statusMsg.Metadata == nil {
			statusMsg.Metadata = map[string]interface{}{}
		}
		statusMsg.Metadata["collab_internal_event"] = true
		_ = h.SendMessage(statusMsg)
	}

	return h.collabManager.GetCollaborationSnapshot(collabID)
}
