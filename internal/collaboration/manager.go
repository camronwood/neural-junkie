package collaboration

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// HubInterface is the subset of Hub that the CollaborationManager needs.
// Defined here to avoid a circular dependency with the hub package.
type HubInterface interface {
	SendMessage(msg *protocol.Message) error
	GetAgent(agentID string) (*protocol.AgentInfo, error)
	GetChannelAgents(channelName string) ([]protocol.AgentInfo, error)
	CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel
}

// CollaborationManager orchestrates multi-agent collaborations
// from creation through planning, approval, execution, and completion.
type CollaborationManager struct {
	hub            HubInterface
	collaborations map[string]*Collaboration // id -> collaboration
	mu             sync.RWMutex
}

// NewCollaborationManager creates a new manager attached to the hub.
func NewCollaborationManager(hub HubInterface) *CollaborationManager {
	return &CollaborationManager{
		hub:            hub,
		collaborations: make(map[string]*Collaboration),
	}
}

// CreateCollaboration starts a new multi-agent collaboration.
// agentIDs are resolved to AgentInfo from the hub, a dedicated channel is
// created, and a planning discussion is immediately started.
func (cm *CollaborationManager) CreateCollaboration(
	description string,
	agentIDs []string,
	channel string,
	createdBy string,
	config DiscussionConfig,
) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	active := 0
	for _, c := range cm.collaborations {
		if c.Phase != PhaseCompleted && c.Phase != PhaseCancelled {
			active++
		}
	}
	if active >= MaxConcurrentCollaborations {
		return nil, fmt.Errorf("maximum concurrent collaborations (%d) reached", MaxConcurrentCollaborations)
	}

	if len(agentIDs) < 2 {
		return nil, fmt.Errorf("at least 2 agents are required for a collaboration")
	}

	cfg := config.Normalized()

	agents := make([]CollaborationAgent, 0, len(agentIDs))
	participantIDs := make([]string, 0, len(agentIDs))
	for _, id := range agentIDs {
		info, err := cm.hub.GetAgent(id)
		if err != nil {
			return nil, fmt.Errorf("agent %s not found: %w", id, err)
		}
		role := SuggestRole(info.Type, info.Expertise)
		agents = append(agents, CollaborationAgent{
			AgentID:   info.ID,
			AgentName: info.Name,
			AgentType: info.Type,
			Expertise: info.Expertise,
			Role:      role,
		})
		participantIDs = append(participantIDs, info.ID)
	}

	now := time.Now()
	collabID := uuid.New().String()

	artifact := &SharedArtifact{
		ID:        uuid.New().String(),
		Title:     "Collaboration Plan",
		Content:   "",
		Version:   0,
		Status:    ArtifactDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}

	discussion := &DiscussionSession{
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

	collab := &Collaboration{
		ID:          collabID,
		Title:       truncate(description, 80),
		Description: description,
		Phase:       PhasePlanning,
		Agents:      agents,
		Plan:        artifact,
		Discussion:  discussion,
		Channel:     channel,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
		Config:      cfg,
	}

	cm.collaborations[collabID] = collab

	log.Printf("[CollaborationManager] Created collaboration %s with %d agents", collabID[:8], len(agents))
	return collab, nil
}

// GetCollaboration returns a collaboration by ID.
func (cm *CollaborationManager) GetCollaboration(id string) (*Collaboration, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[id]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", id)
	}
	return c, nil
}

// GetCollaborationSnapshot returns a deep copy of a collaboration so callers
// can safely attach/publish state without mutating manager-owned data.
func (cm *CollaborationManager) GetCollaborationSnapshot(id string) (*Collaboration, error) {
	cm.mu.RLock()
	c, ok := cm.collaborations[id]
	cm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", id)
	}
	return cloneCollaboration(c)
}

// ListActive returns all non-terminal collaborations.
func (cm *CollaborationManager) ListActive() []*Collaboration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var out []*Collaboration
	for _, c := range cm.collaborations {
		if c.Phase != PhaseCompleted && c.Phase != PhaseCancelled {
			out = append(out, c)
		}
	}
	return out
}

// ListSnapshots returns deep-copied collaboration snapshots optionally filtered
// by channel and terminal-state inclusion. Results are sorted by most recently
// updated collaboration first.
func (cm *CollaborationManager) ListSnapshots(channel string, includeTerminal bool) []*Collaboration {
	cm.mu.RLock()
	candidates := make([]*Collaboration, 0, len(cm.collaborations))
	for _, c := range cm.collaborations {
		if c == nil {
			continue
		}
		if channel != "" && c.Channel != channel {
			continue
		}
		if !includeTerminal && (c.Phase == PhaseCompleted || c.Phase == PhaseCancelled) {
			continue
		}
		candidates = append(candidates, c)
	}
	cm.mu.RUnlock()

	snapshots := make([]*Collaboration, 0, len(candidates))
	for _, c := range candidates {
		cloned, err := cloneCollaboration(c)
		if err != nil {
			log.Printf("[CollaborationManager] Failed to clone collaboration %s for list: %v", shortCollabID(c.ID), err)
			continue
		}
		snapshots = append(snapshots, cloned)
	}

	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].UpdatedAt.After(snapshots[j].UpdatedAt)
	})
	return snapshots
}

// AddParticipants adds one or more agents to an active planning/reviewing
// collaboration and extends the discussion participant list for turn-taking.
// Returns snapshots of newly-added participants.
func (cm *CollaborationManager) AddParticipants(collabID string, agentIDs []string) ([]CollaborationAgent, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhasePlanning && c.Phase != PhaseReviewing {
		return nil, fmt.Errorf("cannot add participants while collaboration is in %s phase", c.Phase)
	}

	existing := make(map[string]struct{}, len(c.Agents))
	for _, participant := range c.Agents {
		existing[participant.AgentID] = struct{}{}
	}

	added := make([]CollaborationAgent, 0, len(agentIDs))
	for _, id := range agentIDs {
		if id == "" {
			continue
		}
		if _, already := existing[id]; already {
			continue
		}

		info, err := cm.hub.GetAgent(id)
		if err != nil || info == nil {
			return nil, fmt.Errorf("agent %s not found", id)
		}

		participant := CollaborationAgent{
			AgentID:   info.ID,
			AgentName: info.Name,
			AgentType: info.Type,
			Expertise: info.Expertise,
			Role:      SuggestRole(info.Type, info.Expertise),
		}
		c.Agents = append(c.Agents, participant)
		existing[id] = struct{}{}
		added = append(added, participant)

		if c.Discussion != nil {
			c.Discussion.Participants = append(c.Discussion.Participants, id)
			if c.Discussion.TurnsThisRound == nil {
				c.Discussion.TurnsThisRound = make(map[string]int)
			}
			if c.Discussion.Consensus == nil {
				c.Discussion.Consensus = make(map[string]ConsensusState)
			}
			c.Discussion.TurnsThisRound[id] = 0
			c.Discussion.Consensus[id] = ConsensusUndecided
		}
	}

	if len(added) > 0 {
		c.UpdatedAt = time.Now()
	}
	return added, nil
}

// ApprovePlan transitions a collaboration from reviewing -> executing
// and distributes tasks to agents.
func (cm *CollaborationManager) ApprovePlan(collabID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhaseReviewing {
		return nil, fmt.Errorf("collaboration is in %s phase, expected reviewing", c.Phase)
	}

	c.Phase = PhaseApproved
	if c.Plan != nil {
		c.Plan.Status = ArtifactApproved
	}
	c.UpdatedAt = time.Now()

	log.Printf("[CollaborationManager] Plan approved for collaboration %s", collabID[:8])
	return c, nil
}

// TransitionToExecuting moves an approved collaboration into execution
// and creates a fresh bounded discussion for cross-agent Q&A during execution.
func (cm *CollaborationManager) TransitionToExecuting(collabID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhaseApproved {
		return nil, fmt.Errorf("collaboration is in %s phase, expected approved", c.Phase)
	}

	c.Phase = PhaseExecuting
	now := time.Now()
	c.UpdatedAt = now

	participantIDs := make([]string, 0, len(c.Agents))
	for _, a := range c.Agents {
		participantIDs = append(participantIDs, a.AgentID)
	}
	c.Discussion = &DiscussionSession{
		ID:                uuid.New().String(),
		CollaborationID:   collabID,
		Topic:             "Execution Q&A: " + c.Description,
		Participants:      participantIDs,
		MaxRounds:         c.Config.MaxRounds,
		CurrentRound:      1,
		TurnBudget:        c.Config.TurnBudget,
		TotalMessageCount: 0,
		MaxTotalMessages:  c.Config.MaxTotalMessages,
		Status:            DiscussionActive,
		Timeout:           c.Config.Timeout,
		StartedAt:         now,
		CurrentTurnIndex:  0,
		TurnsThisRound:    make(map[string]int),
		Consensus:         make(map[string]ConsensusState),
	}

	log.Printf("[CollaborationManager] Collaboration %s transitioned to executing with %d tasks", collabID[:8], len(c.Tasks))
	return c, nil
}

// ExtendDiscussionLimits raises planning/review discussion caps and re-opens
// an exhausted or timed-out discussion (or bumps ceilings while still active).
func (cm *CollaborationManager) ExtendDiscussionLimits(collabID string, extraRounds, extraMessages int) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhasePlanning && c.Phase != PhaseReviewing {
		return nil, fmt.Errorf("can only extend limits during planning or review (current phase: %s)", c.Phase)
	}
	if c.Discussion == nil {
		return nil, fmt.Errorf("no discussion on collaboration %s", collabID)
	}
	if extraRounds <= 0 && extraMessages <= 0 {
		return nil, fmt.Errorf("provide --rounds and/or --messages with positive values")
	}

	d := c.Discussion
	switch d.Status {
	case DiscussionActive, DiscussionBudgetExhausted, DiscussionTimedOut:
		// ok
	default:
		return nil, fmt.Errorf("discussion status %q cannot be extended", d.Status)
	}

	if extraRounds > 0 {
		d.MaxRounds += extraRounds
		if d.MaxRounds > HardMaxRounds {
			d.MaxRounds = HardMaxRounds
		}
	}
	if extraMessages > 0 {
		d.MaxTotalMessages += extraMessages
		if d.MaxTotalMessages > HardMaxTotalMessages {
			d.MaxTotalMessages = HardMaxTotalMessages
		}
	}
	d.Status = DiscussionActive
	c.UpdatedAt = time.Now()

	log.Printf("[Discussion %s] Limits extended: max_rounds=%d max_messages=%d", collabID[:8], d.MaxRounds, d.MaxTotalMessages)
	return c, nil
}

// RevisePlan sends user feedback back into the discussion.
// If the discussion had ended, a new round is started.
func (cm *CollaborationManager) RevisePlan(collabID string, feedback string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhaseReviewing {
		return nil, fmt.Errorf("collaboration is in %s phase, expected reviewing", c.Phase)
	}

	c.Phase = PhasePlanning
	c.UpdatedAt = time.Now()

	if c.Discussion != nil && c.Discussion.Status != DiscussionActive {
		c.Discussion.Status = DiscussionActive
		c.Discussion.CurrentRound++
		c.Discussion.TurnsThisRound = make(map[string]int)
		c.Discussion.CurrentTurnIndex = 0
	}

	log.Printf("[CollaborationManager] Plan revision requested for collaboration %s", collabID[:8])
	return c, nil
}

// CancelCollaboration terminates a collaboration in any phase.
func (cm *CollaborationManager) CancelCollaboration(collabID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase == PhaseCompleted || c.Phase == PhaseCancelled {
		return nil, fmt.Errorf("collaboration is already in terminal phase %s", c.Phase)
	}

	c.Phase = PhaseCancelled
	c.UpdatedAt = time.Now()
	if c.Discussion != nil {
		c.Discussion.Status = DiscussionCancelled
	}

	log.Printf("[CollaborationManager] Collaboration %s cancelled", collabID[:8])
	return c, nil
}

// CompleteCollaboration marks a collaboration as finished.
func (cm *CollaborationManager) CompleteCollaboration(collabID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}

	c.Phase = PhaseCompleted
	c.UpdatedAt = time.Now()

	log.Printf("[CollaborationManager] Collaboration %s completed", collabID[:8])
	return c, nil
}

// TransitionToReviewing moves a planning collaboration into user review.
func (cm *CollaborationManager) TransitionToReviewing(collabID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhasePlanning {
		return nil, fmt.Errorf("collaboration is in %s phase, expected planning", c.Phase)
	}

	c.Phase = PhaseReviewing
	c.UpdatedAt = time.Now()
	if c.Plan != nil {
		c.Plan.Status = ArtifactProposed
	}

	log.Printf("[CollaborationManager] Collaboration %s moved to reviewing", collabID[:8])
	return c, nil
}

// SetTasks sets the parsed tasks on a collaboration (used after plan synthesis).
func (cm *CollaborationManager) SetTasks(collabID string, tasks []CollaborationTask) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	if len(tasks) > MaxTasksPerCollaboration {
		tasks = tasks[:MaxTasksPerCollaboration]
	}
	c.Tasks = tasks
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateTaskStatus updates the status of a specific task.
func (cm *CollaborationManager) UpdateTaskStatus(collabID, taskID string, status TaskStatus, output string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return fmt.Errorf("collaboration %s not found", collabID)
	}

	for i := range c.Tasks {
		if c.Tasks[i].ID == taskID {
			c.Tasks[i].Status = status
			if output != "" {
				c.Tasks[i].Output = output
			}
			c.Tasks[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("task %s not found in collaboration %s", taskID, collabID)
}

// AllTasksComplete returns true if every task in the collaboration is done.
func (cm *CollaborationManager) AllTasksComplete(collabID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || len(c.Tasks) == 0 {
		return false
	}
	for _, t := range c.Tasks {
		if t.Status != TaskCompleted {
			return false
		}
	}
	return true
}

// IsParticipant checks if an agent is part of a collaboration.
func (cm *CollaborationManager) IsParticipant(collabID, agentID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return false
	}
	for _, a := range c.Agents {
		if a.AgentID == agentID {
			return true
		}
	}
	return false
}

// IsActive returns true if the collaboration is in a non-terminal phase.
func (cm *CollaborationManager) IsActive(collabID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return false
	}
	return c.Phase != PhaseCompleted && c.Phase != PhaseCancelled
}

// GetCollaborationForAgent returns the active collaboration a given agent
// is participating in (if any). Returns nil if the agent has no active collab.
func (cm *CollaborationManager) GetCollaborationForAgent(agentID string) *Collaboration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, c := range cm.collaborations {
		if c.Phase == PhaseCompleted || c.Phase == PhaseCancelled {
			continue
		}
		for _, a := range c.Agents {
			if a.AgentID == agentID {
				return c
			}
		}
	}
	return nil
}

// Len returns the number of collaborations in memory (including terminal).
func (cm *CollaborationManager) Len() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.collaborations)
}

// Snapshot returns a deep-copied snapshot of all collaborations.
func (cm *CollaborationManager) Snapshot() map[string]*Collaboration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	out := make(map[string]*Collaboration, len(cm.collaborations))
	for id, collab := range cm.collaborations {
		cloned, err := cloneCollaboration(collab)
		if err != nil {
			log.Printf("[CollaborationManager] Failed to clone collaboration %s for snapshot: %v", shortCollabID(id), err)
			continue
		}
		out[id] = cloned
	}
	return out
}

// Restore replaces the in-memory collaboration map from a snapshot.
func (cm *CollaborationManager) Restore(snapshot map[string]*Collaboration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.collaborations = make(map[string]*Collaboration, len(snapshot))
	for id, collab := range snapshot {
		cloned, err := cloneCollaboration(collab)
		if err != nil {
			log.Printf("[CollaborationManager] Failed to restore collaboration %s: %v", shortCollabID(id), err)
			continue
		}
		cm.collaborations[id] = cloned
	}
}

func cloneCollaboration(c *Collaboration) (*Collaboration, error) {
	if c == nil {
		return nil, nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var cloned Collaboration
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}

func shortCollabID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// SuggestRole returns a human-readable role string based on agent type.
func SuggestRole(agentType protocol.AgentType, expertise []string) string {
	switch agentType {
	case protocol.AgentTypeCLI:
		return "Implementation & Code Generation"
	case protocol.AgentTypeSecurity:
		return "Security Review & Auth Design"
	case protocol.AgentTypeRust:
		return "Rust Architecture & Systems Design"
	case protocol.AgentTypeBackend:
		return "Backend Architecture & API Design"
	case protocol.AgentTypeFrontend:
		return "Frontend Architecture & UI Design"
	case protocol.AgentTypeDevOps:
		return "Infrastructure & Deployment"
	case protocol.AgentTypeDatabase:
		return "Data Modeling & Query Optimization"
	case protocol.AgentTypeRepo:
		return "Codebase Analysis & Review"
	default:
		return "General Contributor"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
