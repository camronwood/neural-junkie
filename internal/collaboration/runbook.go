package collaboration

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunbookUpdatePayload is the body for updating a draft/reviewing runbook.
type RunbookUpdatePayload struct {
	Title            string
	Description      string
	AgentIDs         []string
	Tasks            []CollaborationTask
	ExecutionPolicy  *ExecutionPolicy
	GraphLayout      GraphLayout
}

// CreateRunbook starts a user-authored collaboration in draft phase (no agent discussion).
func (cm *CollaborationManager) CreateRunbook(
	description string,
	agentIDs []string,
	channel string,
	createdBy string,
	cfg DiscussionConfig,
	opts CreateOptions,
) (*Collaboration, error) {
	if len(agentIDs) < 1 {
		return nil, fmt.Errorf("at least 1 agent is required for a runbook")
	}
	opts.Source = SourceRunbook
	opts.SkipDiscussion = true
	if len(opts.InitialTasks) == 0 {
		if existing := cm.FindReusableEmptyDraftRunbook(createdBy); existing != nil {
			updated, err := cm.UpdateRunbook(existing.ID, RunbookUpdatePayload{
				Description: description,
				AgentIDs:    agentIDs,
			})
			if err != nil {
				return nil, err
			}
			log.Printf("[CollaborationManager] Reusing empty draft runbook %s", existing.ID[:8])
			return updated, nil
		}
	}
	collab, err := cm.createCollaborationCore(description, agentIDs, channel, createdBy, cfg, opts, PhaseDraft)
	if err != nil {
		return nil, err
	}
	if len(opts.InitialTasks) > 0 {
		if err := cm.SetTasks(collab.ID, opts.InitialTasks); err != nil {
			return nil, err
		}
		collab, _ = cm.GetCollaborationSnapshot(collab.ID)
	}
	return collab, nil
}

func (cm *CollaborationManager) createCollaborationCore(
	description string,
	agentIDs []string,
	channel string,
	createdBy string,
	cfg DiscussionConfig,
	opts CreateOptions,
	initialPhase CollaborationPhase,
) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	active := 0
	for _, c := range cm.collaborations {
		if collaborationCountsAsActive(c) {
			active++
		}
	}
	if active >= MaxConcurrentCollaborations {
		return nil, fmt.Errorf("maximum concurrent collaborations (%d) reached — %s",
			MaxConcurrentCollaborations, summarizeActiveCollaborations(cm.collaborations))
	}

	agentIDSet := make(map[string]bool)
	for _, id := range agentIDs {
		agentIDSet[id] = true
	}
	for _, t := range opts.InitialTasks {
		if t.AssignedTo != "" {
			agentIDSet[t.AssignedTo] = true
		}
	}
	allIDs := make([]string, 0, len(agentIDSet))
	for id := range agentIDSet {
		allIDs = append(allIDs, id)
	}
	if len(allIDs) < 1 {
		return nil, fmt.Errorf("at least 1 agent is required")
	}
	if opts.Source != SourceRunbook && len(allIDs) < 2 {
		return nil, fmt.Errorf("at least 2 agents are required for a collaboration")
	}

	cfg = cfg.Normalized()
	agents := make([]CollaborationAgent, 0, len(allIDs))
	for _, id := range allIDs {
		info, err := cm.hub.GetAgent(id)
		if err != nil {
			return nil, fmt.Errorf("agent %s not found: %w", id, err)
		}
		agents = append(agents, CollaborationAgent{
			AgentID:   info.ID,
			AgentName: info.Name,
			AgentType: info.Type,
			Expertise: info.Expertise,
			Role:      SuggestRole(info.Type, info.Expertise),
		})
	}

	now := time.Now()
	collabID := uuid.New().String()
	source := opts.Source
	if source == "" {
		source = SourceDiscussion
	}

	artifact := &SharedArtifact{
		ID:        uuid.New().String(),
		Title:     "Collaboration Plan",
		Content:   "",
		Version:   0,
		Status:    ArtifactDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var discussion *DiscussionSession
	if !opts.SkipDiscussion && initialPhase == PhasePlanning {
		participantIDs := make([]string, len(agents))
		for i, a := range agents {
			participantIDs[i] = a.AgentID
		}
		discussion = &DiscussionSession{
			ID:                uuid.New().String(),
			CollaborationID:   collabID,
			Topic:             description,
			Participants:      participantIDs,
			MaxRounds:         cfg.MaxRounds,
			CurrentRound:      1,
			TurnBudget:        cfg.TurnBudget,
			TotalMessageCount: 0,
			MaxTotalMessages:  cfg.MaxTotalMessages,
			Status:            DiscussionActive,
			Timeout:           cfg.Timeout,
			StartedAt:         now,
			CurrentTurnIndex:  0,
			TurnsThisRound:    make(map[string]int),
			Consensus:         make(map[string]ConsensusState),
		}
		for _, id := range participantIDs {
			discussion.Consensus[id] = ConsensusUndecided
		}
	}

	execMode := opts.ExecutionMode
	if execMode == "" {
		execMode = ExecutionModeSandbox
	}

	collab := &Collaboration{
		ID:             collabID,
		Title:          DeriveCollaborationTitle(description),
		Description:    description,
		Phase:          initialPhase,
		Source:         source,
		Agents:         agents,
		Plan:           artifact,
		Discussion:     discussion,
		Channel:        channel,
		CreatedBy:      createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
		Config:         cfg,
		ExecutionMode:  execMode,
		SourceRepoPath: strings.TrimSpace(opts.SourceRepoPath),
	}
	cm.collaborations[collabID] = collab
	log.Printf("[CollaborationManager] Created %s collaboration %s with %d agents", source, collabID[:8], len(agents))
	return collab, nil
}

// UpdateRunbook updates a draft or reviewing runbook.
func (cm *CollaborationManager) UpdateRunbook(collabID string, payload RunbookUpdatePayload) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Source != SourceRunbook {
		return nil, fmt.Errorf("not a runbook collaboration")
	}
	if c.Phase != PhaseDraft && c.Phase != PhaseReviewing {
		return nil, fmt.Errorf("runbook can only be edited in draft or reviewing phase (current: %s)", c.Phase)
	}

	if t := strings.TrimSpace(payload.Title); t != "" {
		c.Title = t
	}
	if d := strings.TrimSpace(payload.Description); d != "" {
		c.Description = d
	}
	if len(payload.AgentIDs) > 0 {
		agents := make([]CollaborationAgent, 0, len(payload.AgentIDs))
		for _, id := range payload.AgentIDs {
			info, err := cm.hub.GetAgent(id)
			if err != nil {
				return nil, fmt.Errorf("agent %s not found: %w", id, err)
			}
			agents = append(agents, CollaborationAgent{
				AgentID:   info.ID,
				AgentName: info.Name,
				AgentType: info.Type,
				Expertise: info.Expertise,
				Role:      SuggestRole(info.Type, info.Expertise),
			})
		}
		c.Agents = agents
	}
	if payload.Tasks != nil {
		tasks := payload.Tasks
		if len(tasks) > maxTasksLimit() {
			tasks = tasks[:maxTasksLimit()]
		}
		for i := range tasks {
			if tasks[i].ID == "" {
				tasks[i].ID = uuid.New().String()
			}
			if tasks[i].CreatedAt.IsZero() {
				tasks[i].CreatedAt = time.Now()
			}
			tasks[i].UpdatedAt = time.Now()
			if tasks[i].Status == "" {
				tasks[i].Status = TaskPending
			}
			tasks[i].PromptDispatched = false
			normalizeTaskOnSave(&tasks[i])
		}
		NormalizeDependencies(tasks)
		if err := ValidateDAG(tasks); err != nil {
			return nil, fmt.Errorf("invalid task graph: %w", err)
		}
		c.Tasks = tasks
	}
	if payload.ExecutionPolicy != nil {
		c.ExecutionPolicy = *payload.ExecutionPolicy
	}
	if payload.GraphLayout != nil {
		c.GraphLayout = payload.GraphLayout
	}
	c.UpdatedAt = time.Now()
	return c, nil
}

// SubmitRunbook moves draft → reviewing after validating the task graph.
func (cm *CollaborationManager) SubmitRunbook(collabID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Source != SourceRunbook {
		return nil, fmt.Errorf("not a runbook collaboration")
	}
	if c.Phase != PhaseDraft {
		return nil, fmt.Errorf("runbook must be in draft phase to submit (current: %s)", c.Phase)
	}
	if len(c.Tasks) == 0 {
		return nil, fmt.Errorf("add at least one task before submitting")
	}
	if err := ValidateDAG(c.Tasks); err != nil {
		return nil, err
	}
	c.Phase = PhaseReviewing
	if c.Plan != nil {
		c.Plan.Status = ArtifactProposed
	}
	c.UpdatedAt = time.Now()
	return c, nil
}

// SuggestRunbookAssignee recommends an agent from the runbook pool for a task.
func (cm *CollaborationManager) SuggestRunbookAssignee(collabID, title, description string) (*AssignSuggestion, error) {
	c, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("collaboration not found")
	}
	inFlight := make(map[string]int)
	for _, t := range c.Tasks {
		if t.Status == TaskPending || t.Status == TaskInProgress {
			if t.AssignedTo != "" {
				inFlight[t.AssignedTo]++
			}
		}
	}
	return SuggestAssignee(c.Agents, title, description, inFlight), nil
}

// runbookExecutionParticipants returns agent IDs assigned to at least one task.
// Runbook Execution Q&A uses only assignees so the pool does not round-robin non-workers.
// If no tasks have assignees, falls back to the full agent pool.
func runbookExecutionParticipants(c *Collaboration) []string {
	if c == nil {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(c.Tasks))
	for _, t := range c.Tasks {
		id := strings.TrimSpace(t.AssignedTo)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) > 0 {
		return out
	}
	for _, a := range c.Agents {
		out = append(out, a.AgentID)
	}
	return out
}

// ParsePlanTasks extracts tasks from markdown for runbook import.
func ParsePlanTasks(markdown string, agents []CollaborationAgent) ([]CollaborationTask, error) {
	tasks := ExtractTasksFromPlan(markdown, agents)
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found in plan text")
	}
	if err := ValidateDAG(tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}
