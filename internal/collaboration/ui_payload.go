package collaboration

// DiscussionUIPayload is discussion state without message bodies (for WS metadata).
type DiscussionUIPayload struct {
	ID                string                      `json:"id,omitempty"`
	CollaborationID   string                      `json:"collaboration_id,omitempty"`
	Topic             string                      `json:"topic,omitempty"`
	MaxRounds         int                         `json:"max_rounds,omitempty"`
	CurrentRound      int                         `json:"current_round,omitempty"`
	TurnBudget        int                         `json:"turn_budget,omitempty"`
	TotalMessageCount int                         `json:"total_message_count,omitempty"`
	MaxTotalMessages  int                         `json:"max_total_messages,omitempty"`
	Status            DiscussionStatus            `json:"status,omitempty"`
	CurrentTurnIndex  int                         `json:"current_turn_index,omitempty"`
	Consensus         map[string]ConsensusState   `json:"consensus,omitempty"`
}

// CollaborationUIPayload is a slim collaboration snapshot for message metadata.
// Full state remains in CollaborationManager and GET /api/collaborations.
type CollaborationUIPayload struct {
	ID                    string               `json:"id"`
	Title                 string               `json:"title"`
	Description           string               `json:"description,omitempty"`
	Phase                 CollaborationPhase   `json:"phase"`
	Source                CollaborationSource  `json:"source,omitempty"`
	Agents                []CollaborationAgent `json:"agents,omitempty"`
	Plan                  *SharedArtifact      `json:"plan,omitempty"`
	Tasks                 []CollaborationTask  `json:"tasks,omitempty"`
	Discussion            *DiscussionUIPayload `json:"discussion,omitempty"`
	Channel               string               `json:"channel"`
	ThreadID              string               `json:"thread_id,omitempty"`
	CreatedBy             string               `json:"created_by,omitempty"`
	Config                DiscussionConfig     `json:"config,omitempty"`
	ExecutionMode         ExecutionMode        `json:"execution_mode,omitempty"`
	SourceRepoPath        string               `json:"source_repo_path,omitempty"`
	WorktreeBranch        string               `json:"worktree_branch,omitempty"`
	WorkingDirectory      string               `json:"working_directory,omitempty"`
	WorkspaceAcknowledged bool                 `json:"workspace_acknowledged,omitempty"`
	TasksDispatched       bool                 `json:"tasks_dispatched,omitempty"`
}

// ToUIPayload returns metadata safe for WebSocket attachment (no discussion.messages).
func (c *Collaboration) ToUIPayload() *CollaborationUIPayload {
	if c == nil {
		return nil
	}
	out := &CollaborationUIPayload{
		ID:                    c.ID,
		Title:                 c.Title,
		Description:           c.Description,
		Phase:                 c.Phase,
		Source:                c.Source,
		Agents:                append([]CollaborationAgent(nil), c.Agents...),
		Channel:               c.Channel,
		ThreadID:              c.ThreadID,
		CreatedBy:             c.CreatedBy,
		Config:                c.Config,
		ExecutionMode:         c.ExecutionMode,
		SourceRepoPath:        c.SourceRepoPath,
		WorktreeBranch:        c.WorktreeBranch,
		WorkingDirectory:      c.WorkingDirectory,
		WorkspaceAcknowledged: c.WorkspaceAcknowledged,
		TasksDispatched:       c.TasksDispatched,
	}
	if c.Plan != nil {
		planCopy := *c.Plan
		out.Plan = &planCopy
	}
	if len(c.Tasks) > 0 {
		out.Tasks = append([]CollaborationTask(nil), c.Tasks...)
	}
	if c.Discussion != nil {
		d := c.Discussion
		out.Discussion = &DiscussionUIPayload{
			ID:                d.ID,
			CollaborationID:   d.CollaborationID,
			Topic:             d.Topic,
			MaxRounds:         d.MaxRounds,
			CurrentRound:      d.CurrentRound,
			TurnBudget:        d.TurnBudget,
			TotalMessageCount: d.TotalMessageCount,
			MaxTotalMessages:  d.MaxTotalMessages,
			Status:            d.Status,
			CurrentTurnIndex:  d.CurrentTurnIndex,
			Consensus:         d.Consensus,
		}
	}
	return out
}
