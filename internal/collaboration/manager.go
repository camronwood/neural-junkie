package collaboration

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/collabworktree"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// HubInterface is the subset of Hub that the CollaborationManager needs.
// Defined here to avoid a circular dependency with the hub package.
type HubInterface interface {
	SendMessage(msg *protocol.Message) error
	GetAgent(agentID string) (*protocol.AgentInfo, error)
	FindLiveAgentByDisplayName(name string, agentType protocol.AgentType) *protocol.AgentInfo
	GetChannelAgents(channelName string) ([]protocol.AgentInfo, error)
	CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel
}

// CollaborationManager orchestrates multi-agent collaborations
// from creation through planning, approval, execution, and completion.
type CollaborationManager struct {
	hub            HubInterface
	collaborations map[string]*Collaboration // id -> collaboration
	assetsRootFn   func() string             // parent dir for per-collab sandboxes; optional
	mu             sync.RWMutex
}

// NewCollaborationManager creates a new manager attached to the hub.
func NewCollaborationManager(hub HubInterface) *CollaborationManager {
	return &CollaborationManager{
		hub:            hub,
		collaborations: make(map[string]*Collaboration),
	}
}

// SetAssetsRootResolver supplies the parent directory for collaboration execution
// sandboxes (<root>/<collaboration-id>/). Called on each transition to executing.
func (cm *CollaborationManager) SetAssetsRootResolver(fn func() string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.assetsRootFn = fn
}

func (cm *CollaborationManager) collabAssetsBaseDir() (string, error) {
	cm.mu.RLock()
	fn := cm.assetsRootFn
	cm.mu.RUnlock()

	if fn != nil {
		if root := strings.TrimSpace(fn()); root != "" {
			return root, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("collaboration working directory: home dir: %w", err)
	}
	return filepath.Join(home, ".neural-junkie", "collaborations"), nil
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
	opts ...CreateOptions,
) (*Collaboration, error) {
	var createOpts CreateOptions
	if len(opts) > 0 {
		createOpts = opts[0]
	}
	createOpts.Source = SourceDiscussion
	return cm.createCollaborationCore(description, agentIDs, channel, createdBy, config, createOpts, PhasePlanning)
}

// BindCollaborationChannel sets the hub channel where collaboration messages are routed.
func (cm *CollaborationManager) BindCollaborationChannel(collabID, channelName string) error {
	if strings.TrimSpace(channelName) == "" {
		return fmt.Errorf("channel name is required")
	}
	cm.mu.Lock()
	defer cm.mu.Unlock()
	c, ok := cm.collaborations[collabID]
	if !ok || c == nil {
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	c.Channel = channelName
	c.UpdatedAt = time.Now()
	return nil
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

func summarizeActiveCollaborations(collabs map[string]*Collaboration) string {
	type row struct {
		id    string
		phase CollaborationPhase
		ch    string
		title string
		tasks int
	}
	rows := make([]row, 0, len(collabs))
	for _, c := range collabs {
		if c == nil || c.Phase == PhaseCompleted || c.Phase == PhaseCancelled {
			continue
		}
		rows = append(rows, row{
			id:    shortCollabID(c.ID),
			phase: c.Phase,
			ch:    c.Channel,
			title: c.Title,
			tasks: len(c.Tasks),
		})
	}
	if len(rows) == 0 {
		return "no active collaborations listed"
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].id < rows[j].id
	})
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		title := strings.TrimSpace(r.title)
		if title == "" {
			title = "untitled"
		}
		if len(title) > 48 {
			title = title[:45] + "..."
		}
		parts = append(parts, fmt.Sprintf("`%s` %s — %s on #%s (%d task(s); cancel via Task Management or /cancel-plan %s)",
			r.id, r.phase, title, r.ch, r.tasks, r.id))
	}
	return strings.Join(parts, "; ")
}

// GetByChannel returns the collaboration bound to channelName, if any.
// When multiple match (unusual), returns the most recently updated.
func (cm *CollaborationManager) GetByChannel(channelName string) *Collaboration {
	channelName = strings.TrimSpace(channelName)
	if channelName == "" {
		return nil
	}
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	var best *Collaboration
	var bestTime time.Time
	for _, c := range cm.collaborations {
		if c == nil || c.Channel != channelName {
			continue
		}
		t := c.UpdatedAt
		if best == nil || t.After(bestTime) {
			best = c
			bestTime = t
		}
	}
	return best
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
// For each collaboration, EnsureExecutionTasks runs first so list payloads match
// execution routing; assigneesUpdated[i] is true when that call created or
// reassigned tasks so the hub can redispatch collaboration_task messages.
func (cm *CollaborationManager) ListSnapshots(channel string, includeTerminal bool) ([]*Collaboration, []bool) {
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

	type row struct {
		snap             *Collaboration
		assigneesUpdated bool
	}
	rows := make([]row, 0, len(candidates))
	for _, c := range candidates {
		assigneesUpdated, err := cm.EnsureExecutionTasks(c.ID)
		if err != nil {
			log.Printf("[CollaborationManager] ListSnapshots EnsureExecutionTasks for %s: %v", shortCollabID(c.ID), err)
			continue
		}
		cloned, err := cloneCollaboration(c)
		if err != nil {
			log.Printf("[CollaborationManager] Failed to clone collaboration %s for list: %v", shortCollabID(c.ID), err)
			continue
		}
		rows = append(rows, row{snap: cloned, assigneesUpdated: assigneesUpdated})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].snap.UpdatedAt.After(rows[j].snap.UpdatedAt)
	})
	snapshots := make([]*Collaboration, len(rows))
	flags := make([]bool, len(rows))
	for i := range rows {
		snapshots[i] = rows[i].snap
		flags[i] = rows[i].assigneesUpdated
	}
	return snapshots, flags
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

// ApprovePlan transitions a collaboration from reviewing -> approved.
// If the collaboration is already approved (e.g. a prior TransitionToExecuting failed),
// this call is a no-op so the hub can retry execution without getting stuck.
func (cm *CollaborationManager) ApprovePlan(collabID string) (*Collaboration, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}

	switch c.Phase {
	case PhaseCompleted, PhaseCancelled:
		return nil, fmt.Errorf("collaboration is in %s phase, cannot approve plan", c.Phase)
	case PhaseExecuting:
		return nil, fmt.Errorf("collaboration is already executing")
	case PhaseApproved:
		log.Printf("[CollaborationManager] ApprovePlan: collaboration %s already approved (retry execution)", collabID[:8])
		return c, nil
	case PhaseReviewing:
		c.Phase = PhaseApproved
		if c.Plan != nil {
			c.Plan.Status = ArtifactApproved
		}
		c.UpdatedAt = time.Now()
		log.Printf("[CollaborationManager] Plan approved for collaboration %s", collabID[:8])
		return c, nil
	default:
		return nil, fmt.Errorf("collaboration is in %s phase, expected reviewing", c.Phase)
	}
}

// TransitionToExecuting moves an approved collaboration into execution
// and creates a fresh bounded discussion for cross-agent Q&A during execution.
func (cm *CollaborationManager) TransitionToExecuting(collabID string) (*Collaboration, error) {
	baseDir, err := cm.collabAssetsBaseDir()
	if err != nil {
		return nil, err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhaseApproved {
		return nil, fmt.Errorf("collaboration is in %s phase, expected approved", c.Phase)
	}

	ch := c.Channel
	now := time.Now()
	for id, other := range cm.collaborations {
		if other == nil || other.ID == collabID || other.Channel != ch {
			continue
		}
		if other.Phase == PhaseExecuting {
			other.Phase = PhaseCancelled
			other.UpdatedAt = now
			if other.Discussion != nil {
				other.Discussion.Status = DiscussionCancelled
			}
			log.Printf("[CollaborationManager] Auto-cancelled executing collaboration %s (channel %q) so %s can execute", id[:8], ch, collabID[:8])
			cm.cleanupWorktreeLocked(other)
		}
	}

	c.Phase = PhaseExecuting
	c.UpdatedAt = now

	execMode := c.ExecutionMode
	if execMode == "" {
		execMode = ExecutionModeSandbox
		c.ExecutionMode = execMode
	}
	switch execMode {
	case ExecutionModeWorktree:
		if c.WorktreeBranch == "" {
			c.WorktreeBranch = collabworktree.DefaultBranchName(c.ID)
		}
		if err := cm.createWorktreeIfReady(c, baseDir); err != nil {
			return nil, err
		}
	default:
		if err := cm.createSandboxWorkingDir(c, baseDir); err != nil {
			return nil, err
		}
	}

	var participantIDs []string
	if c.Source == SourceRunbook {
		participantIDs = runbookExecutionParticipants(c)
	} else {
		participantIDs = make([]string, 0, len(c.Agents))
		for _, a := range c.Agents {
			participantIDs = append(participantIDs, a.AgentID)
		}
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

// AcknowledgeWorkspace records that the user (desktop or /ack-collab-workspace)
// is ready for task delivery. Idempotent: returns alreadyAck=true if it was
// already acknowledged.
func (cm *CollaborationManager) AcknowledgeWorkspace(collabID string) (alreadyAck bool, _ *Collaboration, err error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return false, nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhaseExecuting {
		return false, nil, fmt.Errorf("collaboration is in %s phase, expected executing", c.Phase)
	}
	if c.WorkspaceAcknowledged {
		return true, c, nil
	}
	c.WorkspaceAcknowledged = true
	c.UpdatedAt = time.Now()
	return false, c, nil
}

// MarkTasksDispatched records that collaboration_task prompts were sent.
func (cm *CollaborationManager) MarkTasksDispatched(collabID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	c, ok := cm.collaborations[collabID]
	if !ok {
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	c.TasksDispatched = true
	c.UpdatedAt = time.Now()
	return nil
}

// IncrementExecutionMessageCount bumps the executing-phase chat counter.
// Returns the new count and whether the cap is exceeded.
func (cm *CollaborationManager) IncrementExecutionMessageCount(collabID string) (count int, overCap bool, err error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	c, ok := cm.collaborations[collabID]
	if !ok {
		return 0, false, fmt.Errorf("collaboration %s not found", collabID)
	}
	c.ExecutionMessageCount++
	c.UpdatedAt = time.Now()
	count = c.ExecutionMessageCount
	return count, count > MaxExecutionMessages, nil
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
	cm.cleanupWorktreeLocked(c)

	log.Printf("[CollaborationManager] Collaboration %s cancelled", collabID[:8])
	return c, nil
}

// SetCollaborationTitle updates the display title (description is unchanged).
func (cm *CollaborationManager) SetCollaborationTitle(collabID, title string) (*Collaboration, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase == PhaseCompleted || c.Phase == PhaseCancelled {
		return nil, fmt.Errorf("cannot rename collaboration in terminal phase %s", c.Phase)
	}
	c.Title = truncate(title, 120)
	c.UpdatedAt = time.Now()
	return c, nil
}

// synthesizePlanFromDiscussionLocked fills an empty plan artifact from discussion messages.
// Caller must hold cm.mu.
func (cm *CollaborationManager) synthesizePlanFromDiscussionLocked(c *Collaboration) {
	if c == nil || c.Plan == nil || strings.TrimSpace(c.Plan.Content) != "" {
		return
	}
	planContent, tasks := SynthesizePlanFromDiscussion(c)
	if strings.TrimSpace(planContent) == "" {
		return
	}
	now := time.Now()
	c.Plan.Content = planContent
	c.Plan.Version++
	c.Plan.UpdatedAt = now
	c.Plan.Status = ArtifactProposed
	if len(tasks) > 0 {
		if len(tasks) > MaxTasksPerCollaboration {
			tasks = tasks[:MaxTasksPerCollaboration]
		}
		c.Tasks = tasks
	}
	c.UpdatedAt = now
	log.Printf("[CollaborationManager] Synthesized plan for %s (%d tasks)", c.ID[:8], len(c.Tasks))
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

	cm.synthesizePlanFromDiscussionLocked(c)
	c.Phase = PhaseReviewing
	c.UpdatedAt = time.Now()
	if c.Plan != nil && c.Plan.Status == ArtifactDraft {
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
	NormalizeDependencies(tasks)
	if err := ValidateDAG(tasks); err != nil {
		return fmt.Errorf("invalid task graph: %w", err)
	}
	c.Tasks = tasks
	c.UpdatedAt = time.Now()
	return nil
}

// MarkTaskPromptDispatched records that a collaboration_task prompt was sent for one task.
func (cm *CollaborationManager) MarkTaskPromptDispatched(collabID, taskID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	c, ok := cm.collaborations[collabID]
	if !ok {
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	for i := range c.Tasks {
		if c.Tasks[i].ID == taskID {
			c.Tasks[i].PromptDispatched = true
			c.Tasks[i].UpdatedAt = time.Now()
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("task %s not found in collaboration %s", taskID, collabID)
}

// assignRoundRobinToUnassignedTasks fills AssignedTo / AssignedName on tasks that
// were parsed from a plan without @participant mentions. Without assignees,
// execution task messages omit task_assigned_to and no agent responds.
func assignRoundRobinToUnassignedTasks(tasks []CollaborationTask, agents []CollaborationAgent) bool {
	if len(agents) == 0 || len(tasks) == 0 {
		return false
	}
	now := time.Now()
	changed := false
	ri := 0
	for i := range tasks {
		if strings.TrimSpace(tasks[i].AssignedTo) != "" {
			continue
		}
		ag := agents[ri%len(agents)]
		ri++
		tasks[i].AssignedTo = ag.AgentID
		tasks[i].AssignedName = ag.AgentName
		tasks[i].UpdatedAt = now
		changed = true
	}
	return changed
}

// EnsureExecutionTasks creates one pending task per collaboration participant when
// no tasks were parsed from the plan, so execution still notifies assignees.
// When tasks exist but lack assignees (common when the plan omits @mentions),
// it round-robins participants onto those tasks so agents receive task_assigned_to.
// The returned bool is true when this call created default tasks or filled missing
// assignees (so the hub can (re)send collaboration_task messages).
func (cm *CollaborationManager) EnsureExecutionTasks(collabID string) (assigneesUpdated bool, err error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return false, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Phase != PhaseExecuting {
		return false, nil
	}
	if len(c.Agents) == 0 {
		return false, nil
	}

	if len(c.Tasks) == 0 {
		hint := strings.TrimSpace(c.Description)
		if c.Plan != nil && strings.TrimSpace(c.Plan.Content) != "" {
			pc := strings.TrimSpace(c.Plan.Content)
			if len(pc) > 400 {
				pc = pc[:400] + "…"
			}
			if hint != "" {
				hint = hint + "\n\n"
			}
			hint += "Approved plan excerpt:\n" + pc
		}
		if hint == "" {
			hint = "Complete your responsibilities under the approved collaboration plan."
		}

		now := time.Now()
		tasks := make([]CollaborationTask, 0, len(c.Agents))
		for i, ag := range c.Agents {
			tasks = append(tasks, CollaborationTask{
				ID:           uuid.New().String(),
				Title:        fmt.Sprintf("Collaboration workstream %d", i+1),
				Description:  fmt.Sprintf("%s\n\nFocus on your specialty (%s) and coordinate with other participants as needed.", hint, ag.Role),
				AssignedTo:   ag.AgentID,
				AssignedName: ag.AgentName,
				Status:       TaskPending,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}
		c.Tasks = tasks
		c.UpdatedAt = now
		log.Printf("[CollaborationManager] EnsureExecutionTasks: created %d default task(s) for %s", len(tasks), collabID[:8])
		return true, nil
	}

	if assignRoundRobinToUnassignedTasks(c.Tasks, c.Agents) {
		c.UpdatedAt = time.Now()
		log.Printf("[CollaborationManager] EnsureExecutionTasks: assigned participants to unassigned task(s) in %s", collabID[:8])
		return true, nil
	}
	return false, nil
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

// AgentOutOfTurnMentionAllowed is false during planning/review when the bounded
// discussion session is no longer active (e.g. budget_exhausted, timed_out).
// Otherwise @mentions could restart agent chatter after limits were hit.
// Outside planning/reviewing (approved, executing), mentions stay allowed.
func (cm *CollaborationManager) AgentOutOfTurnMentionAllowed(collabID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c == nil {
		return false
	}
	switch c.Phase {
	case PhasePlanning, PhaseReviewing:
		if c.Discussion == nil {
			return false
		}
		return c.Discussion.Status == DiscussionActive
	default:
		return true
	}
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

func (cm *CollaborationManager) isAgentIDLive(agentID string) bool {
	if strings.TrimSpace(agentID) == "" {
		return false
	}
	info, err := cm.hub.GetAgent(agentID)
	return err == nil && info != nil
}

// remapDiscussionAgentIDs replaces stale participant IDs in discussion maps.
func (cm *CollaborationManager) remapDiscussionAgentIDs(d *DiscussionSession, idMap map[string]string) {
	if d == nil || len(idMap) == 0 {
		return
	}
	for i, pid := range d.Participants {
		if nid, ok := idMap[pid]; ok {
			d.Participants[i] = nid
		}
	}
	if d.TurnsThisRound != nil {
		next := make(map[string]int)
		for k, v := range d.TurnsThisRound {
			nk := k
			if nid, ok := idMap[k]; ok {
				nk = nid
			}
			next[nk] += v
		}
		d.TurnsThisRound = next
	}
	if d.Consensus != nil {
		next := make(map[string]ConsensusState)
		for k, v := range d.Consensus {
			nk := k
			if nid, ok := idMap[k]; ok {
				nk = nid
			}
			next[nk] = v
		}
		d.Consensus = next
	}
}

// ReconcileRestoredAgentIDs updates collaboration participant IDs, discussion
// participant lists/maps, and task assignees when persisted UUIDs no longer
// match registered hub agents (e.g. after hub restart). Matching uses display
// name plus agent type so @Cursor and similar CLI agents resume correctly.
func (cm *CollaborationManager) ReconcileRestoredAgentIDs() {
	if cm.hub == nil {
		return
	}
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, c := range cm.collaborations {
		if c == nil {
			continue
		}
		if c.Phase == PhaseCompleted || c.Phase == PhaseCancelled {
			continue
		}
		idMap := make(map[string]string)
		for i := range c.Agents {
			ag := &c.Agents[i]
			if cm.isAgentIDLive(ag.AgentID) {
				continue
			}
			reg := cm.hub.FindLiveAgentByDisplayName(ag.AgentName, ag.AgentType)
			if reg == nil {
				log.Printf("[CollaborationManager] reconcile: no live hub agent for participant @%s (%s) in collab %s",
					ag.AgentName, ag.AgentType, shortCollabID(c.ID))
				continue
			}
			old := ag.AgentID
			if old != "" {
				idMap[old] = reg.ID
			}
			ag.AgentID = reg.ID
			c.UpdatedAt = time.Now()
			log.Printf("[CollaborationManager] reconcile participant @%s: %s -> %s in collab %s",
				ag.AgentName, shortCollabID(old), shortCollabID(reg.ID), shortCollabID(c.ID))
		}
		if len(idMap) > 0 {
			cm.remapDiscussionAgentIDs(c.Discussion, idMap)
		}
		for i := range c.Tasks {
			t := &c.Tasks[i]
			if t.AssignedTo == "" {
				continue
			}
			if cm.isAgentIDLive(t.AssignedTo) {
				continue
			}
			if nid, ok := idMap[t.AssignedTo]; ok {
				t.AssignedTo = nid
				t.UpdatedAt = time.Now()
				continue
			}
			lookupName := strings.TrimSpace(t.AssignedName)
			if lookupName == "" {
				continue
			}
			var typ protocol.AgentType
			for _, p := range c.Agents {
				if strings.EqualFold(p.AgentName, lookupName) {
					typ = p.AgentType
					break
				}
			}
			reg := cm.hub.FindLiveAgentByDisplayName(lookupName, typ)
			if reg != nil {
				t.AssignedTo = reg.ID
				t.UpdatedAt = time.Now()
				log.Printf("[CollaborationManager] reconcile task assignee @%s -> %s in collab %s",
					lookupName, shortCollabID(reg.ID), shortCollabID(c.ID))
			}
		}
	}
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

// PruneTerminalCollaborations removes completed/cancelled collaborations from memory,
// keeping only the most recently updated terminal entries. Returns how many were removed.
func (cm *CollaborationManager) PruneTerminalCollaborations(maxKeep int) int {
	if maxKeep < 0 {
		maxKeep = 0
	}
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var terminal []*Collaboration
	for id, c := range cm.collaborations {
		if c == nil {
			delete(cm.collaborations, id)
			continue
		}
		if c.Phase == PhaseCompleted || c.Phase == PhaseCancelled {
			terminal = append(terminal, c)
		}
	}
	if len(terminal) <= maxKeep {
		return 0
	}
	sort.Slice(terminal, func(i, j int) bool {
		return terminal[i].UpdatedAt.After(terminal[j].UpdatedAt)
	})
	removed := 0
	for _, c := range terminal[maxKeep:] {
		delete(cm.collaborations, c.ID)
		removed++
	}
	return removed
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
