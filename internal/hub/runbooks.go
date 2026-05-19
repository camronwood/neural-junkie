package hub

import (
	"fmt"
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
	_, _ = h.collabManager.EnsureExecutionTasks(collabID)
	return h.collabManager.GetCollaborationSnapshot(collabID)
}
