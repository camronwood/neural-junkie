package collaboration

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CollaborationPhase represents the current lifecycle phase
type CollaborationPhase string

const (
	PhaseDraft     CollaborationPhase = "draft"
	PhasePlanning  CollaborationPhase = "planning"
	PhaseReviewing CollaborationPhase = "reviewing"
	PhaseApproved  CollaborationPhase = "approved"
	PhaseExecuting CollaborationPhase = "executing"
	PhaseCompleted CollaborationPhase = "completed"
	PhaseCancelled CollaborationPhase = "cancelled"
)

// CollaborationSource distinguishes agent-driven planning from user-authored runbooks.
type CollaborationSource string

const (
	SourceDiscussion CollaborationSource = "discussion"
	SourceRunbook    CollaborationSource = "runbook"
)

// TaskStatus represents the status of a collaboration task
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskBlocked    TaskStatus = "blocked"
)

// DiscussionStatus represents the status of a discussion session
type DiscussionStatus string

const (
	DiscussionActive          DiscussionStatus = "active"
	DiscussionConverged       DiscussionStatus = "converged"
	DiscussionBudgetExhausted DiscussionStatus = "budget_exhausted"
	DiscussionTimedOut        DiscussionStatus = "timed_out"
	DiscussionCancelled       DiscussionStatus = "cancelled"
)

// ArtifactStatus represents the status of a shared artifact
type ArtifactStatus string

const (
	ArtifactDraft      ArtifactStatus = "draft"
	ArtifactProposed   ArtifactStatus = "proposed"
	ArtifactApproved   ArtifactStatus = "approved"
	ArtifactSuperseded ArtifactStatus = "superseded"
)

// ConsensusState represents an agent's agreement state
type ConsensusState string

const (
	ConsensusUndecided ConsensusState = "undecided"
	ConsensusAgrees    ConsensusState = "agrees"
	ConsensusDisagrees ConsensusState = "disagrees"
)

// DiscussionConfig holds configurable limits for a discussion session.
// All zero-value fields fall back to package-level defaults.
type DiscussionConfig struct {
	MaxRounds        int           `json:"max_rounds"`
	TurnBudget       int           `json:"turn_budget"`
	MaxTotalMessages int           `json:"max_total_messages"`
	Timeout          time.Duration `json:"timeout"`
}

const (
	DefaultMaxRounds        = 3
	DefaultTurnBudget       = 1
	DefaultMaxTotalMessages = 20
	DefaultTimeout          = 5 * time.Minute

	HardMaxRounds        = 10
	HardMaxTurnBudget    = 3
	HardMaxTotalMessages = 50
	HardMaxTimeout       = 30 * time.Minute

	MaxConcurrentCollaborations = 3
	MaxTasksPerCollaboration    = 10
	HardMaxTasksPerCollaboration = 25
	// MaxExecutionMessages caps agent chat posts during the executing phase.
	MaxExecutionMessages = 100
)

// BlockedUpstreamPolicy controls how blocked upstream tasks affect downstream readiness.
type BlockedUpstreamPolicy string

const (
	BlockedPolicyBlock      BlockedUpstreamPolicy = "block"
	BlockedPolicySkipBranch BlockedUpstreamPolicy = "skip_branch"
	BlockedPolicyFailRun    BlockedUpstreamPolicy = "fail_run"
)

// TaskKind distinguishes agent chat tasks from hub-executed action steps.
type TaskKind string

const (
	TaskKindAgent  TaskKind = "agent"
	TaskKindAction TaskKind = "action"
)

// ExecutionPolicy holds collaboration-level orchestration options.
type ExecutionPolicy struct {
	MaxConcurrentTasks    int                   `json:"max_concurrent_tasks,omitempty"`
	MaxExecutionMessages  int                   `json:"max_execution_messages,omitempty"`
	BlockedUpstreamPolicy BlockedUpstreamPolicy `json:"blocked_upstream_policy,omitempty"`
	StrictTaskStatus      bool                  `json:"strict_task_status,omitempty"`
	HandoffMaxChars       int                   `json:"handoff_max_chars,omitempty"`
}

// Normalized returns defaults for zero fields.
func (p ExecutionPolicy) Normalized(source CollaborationSource) ExecutionPolicy {
	out := p
	if out.BlockedUpstreamPolicy == "" {
		out.BlockedUpstreamPolicy = BlockedPolicyBlock
	}
	if source == SourceRunbook && !out.StrictTaskStatus {
		// zero value means unset; runbooks default strict
		out.StrictTaskStatus = true
	}
	if out.HandoffMaxChars <= 0 {
		out.HandoffMaxChars = 2000
	} else if out.HandoffMaxChars > 32000 {
		out.HandoffMaxChars = 32000
	}
	return out
}

// MaxExecutionMessagesLimit returns the effective cap for agent messages during execution.
func (c *Collaboration) MaxExecutionMessagesLimit() int {
	if c == nil {
		return MaxExecutionMessages
	}
	if c.ExecutionPolicy.MaxExecutionMessages > 0 {
		return c.ExecutionPolicy.MaxExecutionMessages
	}
	return MaxExecutionMessages
}

// GraphLayoutNode stores a task node's position in the graph editor.
type GraphLayoutNode struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GraphLayout maps task IDs to canvas positions (persisted on collaboration).
type GraphLayout map[string]GraphLayoutNode

// TaskExecutionOptions configures per-task agent execution.
type TaskExecutionOptions struct {
	ProviderID       string   `json:"provider_id,omitempty"`
	RequiresApproval bool     `json:"requires_approval,omitempty"`
	MaxRetries       int      `json:"max_retries,omitempty"`
	TimeoutSeconds   int      `json:"timeout_seconds,omitempty"`
	ContextPaths     []string `json:"context_paths,omitempty"`
}

// TaskActionSpec defines a hub-executed action step.
type TaskActionSpec struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// EdgeCondition gates an incoming dependency edge.
type EdgeCondition struct {
	Mode     string `json:"mode"` // always | on_status | on_output
	Status   string `json:"status,omitempty"`
	Contains string `json:"contains,omitempty"`
	Regex    string `json:"regex,omitempty"`
}

// DependencyEdge is a conditioned dependency from upstream to this task.
type DependencyEdge struct {
	FromTaskID string         `json:"from_task_id"`
	Condition  *EdgeCondition `json:"condition,omitempty"`
}

// DependencyGroup expresses OR/AND join semantics over upstream tasks.
type DependencyGroup struct {
	Mode    string   `json:"mode"` // all | any
	TaskIDs []string `json:"task_ids"`
}

// Normalized returns a copy with all zero-value fields replaced by defaults
// and all values clamped to hard maximums.
func (dc DiscussionConfig) Normalized() DiscussionConfig {
	out := dc
	if out.MaxRounds <= 0 {
		out.MaxRounds = DefaultMaxRounds
	} else if out.MaxRounds > HardMaxRounds {
		out.MaxRounds = HardMaxRounds
	}
	if out.TurnBudget <= 0 {
		out.TurnBudget = DefaultTurnBudget
	} else if out.TurnBudget > HardMaxTurnBudget {
		out.TurnBudget = HardMaxTurnBudget
	}
	if out.MaxTotalMessages <= 0 {
		out.MaxTotalMessages = DefaultMaxTotalMessages
	} else if out.MaxTotalMessages > HardMaxTotalMessages {
		out.MaxTotalMessages = HardMaxTotalMessages
	}
	if out.Timeout <= 0 {
		out.Timeout = DefaultTimeout
	} else if out.Timeout > HardMaxTimeout {
		out.Timeout = HardMaxTimeout
	}
	return out
}

// ExecutionMode selects where collaboration execution writes files.
type ExecutionMode string

const (
	ExecutionModeSandbox  ExecutionMode = "sandbox"
	ExecutionModeWorktree ExecutionMode = "worktree"
)

// CreateOptions configures optional collaboration creation parameters.
type CreateOptions struct {
	ExecutionMode  ExecutionMode
	SourceRepoPath string // absolute git repo root; optional until workspace ack
	Source         CollaborationSource
	InitialTasks   []CollaborationTask
	SkipDiscussion bool
}

// CollaborationAgent pairs an agent identity with its role inside
// a particular collaboration.
type CollaborationAgent struct {
	AgentID   string             `json:"agent_id"`
	AgentName string             `json:"agent_name"`
	AgentType protocol.AgentType `json:"agent_type"`
	Expertise []string           `json:"expertise"`
	Role      string             `json:"role"`
}

// Collaboration is the top-level orchestration unit that tracks a
// multi-agent work session from planning through execution.
type Collaboration struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Phase       CollaborationPhase   `json:"phase"`
	Source      CollaborationSource  `json:"source,omitempty"`
	Agents      []CollaborationAgent `json:"agents"`
	Plan        *SharedArtifact      `json:"plan,omitempty"`
	Tasks       []CollaborationTask  `json:"tasks,omitempty"`
	Discussion  *DiscussionSession   `json:"discussion,omitempty"`
	Channel     string               `json:"channel"`
	ThreadID    string               `json:"thread_id"`
	CreatedBy   string               `json:"created_by"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	Config      DiscussionConfig     `json:"config"`
	// ExecutionMode is sandbox (default) or worktree (git worktree under assets root).
	ExecutionMode ExecutionMode `json:"execution_mode,omitempty"`
	// SourceRepoPath is the git repository to branch from in worktree mode.
	SourceRepoPath string `json:"source_repo_path,omitempty"`
	// WorktreeBranch is the branch created for worktree execution (e.g. nj/collab-abc12345).
	WorktreeBranch string `json:"worktree_branch,omitempty"`
	// WorkingDirectory is an absolute path created when execution starts; agents
	// and the desktop app use it as the collaboration execution workspace root.
	WorkingDirectory string `json:"working_directory,omitempty"`
	// WorkspaceAcknowledged is set after the user confirms the sandbox in the
	// desktop app (or via /ack-collab-workspace). Until then, task messages are
	// not sent to agents so file/command steps do not race ahead of setup.
	WorkspaceAcknowledged bool `json:"workspace_acknowledged,omitempty"`
	// TasksDispatched is set after initial collaboration_task prompts were sent.
	// Prevents re-dispatch on every channel message during execution.
	TasksDispatched bool `json:"tasks_dispatched,omitempty"`
	// ExecutionMessageCount tracks agent chat posts during the executing phase.
	ExecutionMessageCount int `json:"execution_message_count,omitempty"`
	ExecutionPolicy       ExecutionPolicy `json:"execution_policy,omitempty"`
	GraphLayout           GraphLayout     `json:"graph_layout,omitempty"`
	DispatchPaused        bool            `json:"dispatch_paused,omitempty"`
	// PlanningDiscussion is a snapshot of the planning-phase discussion taken when execution starts.
	PlanningDiscussion *DiscussionSession `json:"planning_discussion,omitempty"`
	PlanningRecap         string `json:"planning_recap,omitempty"`
	SessionRecap          string `json:"session_recap,omitempty"`
	PlanningRecapStatus   string `json:"planning_recap_status,omitempty"` // pending|complete|failed
	SessionRecapStatus    string `json:"session_recap_status,omitempty"`
	PlanningRecapAgentID  string `json:"planning_recap_agent_id,omitempty"`
	SessionRecapAgentID   string `json:"session_recap_agent_id,omitempty"`
	// AwaitingFinalize is set when all work is done but the final recap has not finished.
	AwaitingFinalize      bool   `json:"awaiting_finalize,omitempty"`
	FinalizeReason        string `json:"finalize_reason,omitempty"`
	FinalizeChannel       string `json:"finalize_channel,omitempty"`
	FinalizeMarkOpenTasks bool   `json:"finalize_mark_open_tasks,omitempty"`
}

// Recap status values stored on Collaboration.
const (
	RecapStatusPending  = "pending"
	RecapStatusComplete = "complete"
	RecapStatusFailed   = "failed"
)

// RecapKind distinguishes pre-approval vs end-of-session recaps.
type RecapKind string

const (
	RecapKindPreApproval RecapKind = "pre_approval"
	RecapKindFinal       RecapKind = "final"
)

// EffectiveExecutionPolicy returns normalized execution policy for this collaboration.
func (c *Collaboration) EffectiveExecutionPolicy() ExecutionPolicy {
	if c == nil {
		return ExecutionPolicy{BlockedUpstreamPolicy: BlockedPolicyBlock, HandoffMaxChars: 2000}
	}
	return c.ExecutionPolicy.Normalized(c.Source)
}

// DiscussionBudgetEnforced is true while the plan is still being negotiated
// (planning / user review). After approval, execution-phase discussion is not capped.
func (c *Collaboration) DiscussionBudgetEnforced() bool {
	if c == nil {
		return true
	}
	switch c.Phase {
	case PhaseDraft, PhasePlanning, PhaseReviewing:
		return true
	default:
		return false
	}
}

// CollaborationTask is a unit of work assigned to a specific agent
// within a collaboration. Tasks are produced during the planning phase
// and executed after user approval.
type CollaborationTask struct {
	ID                 string               `json:"id"`
	Title              string               `json:"title"`
	Description        string               `json:"description"`
	AssignedTo         string               `json:"assigned_to"`
	AssignedName       string               `json:"assigned_name"`
	Kind               TaskKind             `json:"kind,omitempty"`
	Action             *TaskActionSpec      `json:"action,omitempty"`
	Options            *TaskExecutionOptions `json:"options,omitempty"`
	Status             TaskStatus           `json:"status"`
	Dependencies       []string             `json:"dependencies,omitempty"`
	DependencyEdges    []DependencyEdge     `json:"dependency_edges,omitempty"`
	DependencyGroups   []DependencyGroup    `json:"dependency_groups,omitempty"`
	PromptDispatched   bool                 `json:"prompt_dispatched,omitempty"`
	AwaitingApproval   bool                 `json:"awaiting_approval,omitempty"`
	SkippedDueToBlocked bool                `json:"skipped_due_to_blocked,omitempty"`
	Output             string               `json:"output,omitempty"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

// EffectiveKind returns agent when unset.
func (t CollaborationTask) EffectiveKind() TaskKind {
	if t.Kind == "" || t.Kind == TaskKindAgent {
		return TaskKindAgent
	}
	return t.Kind
}

// DiscussionSession is a bounded multi-agent conversation with
// round-robin turn-taking and hard budget limits.
type DiscussionSession struct {
	ID                string           `json:"id"`
	CollaborationID   string           `json:"collaboration_id"`
	Topic             string           `json:"topic"`
	Participants      []string         `json:"participants"`
	MaxRounds         int              `json:"max_rounds"`
	CurrentRound      int              `json:"current_round"`
	TurnBudget        int              `json:"turn_budget"`
	TotalMessageCount int              `json:"total_message_count"`
	MaxTotalMessages  int              `json:"max_total_messages"`
	Status            DiscussionStatus `json:"status"`
	Timeout           time.Duration    `json:"timeout"`
	StartedAt         time.Time        `json:"started_at"`

	// CurrentTurnIndex tracks which participant should speak next
	CurrentTurnIndex int `json:"current_turn_index"`

	// TurnsThisRound tracks how many messages each agent has sent in the current round
	TurnsThisRound map[string]int `json:"turns_this_round"`

	// Consensus tracks each participant's agreement state
	Consensus map[string]ConsensusState `json:"consensus"`

	// Messages collects all discussion messages for summarisation
	Messages []*protocol.Message `json:"messages,omitempty"`
}

// ArtifactEdit records a single edit to a shared artifact.
type ArtifactEdit struct {
	EditorID   string    `json:"editor_id"`
	EditorName string    `json:"editor_name"`
	Content    string    `json:"content"`
	Version    int       `json:"version"`
	Timestamp  time.Time `json:"timestamp"`
}

// SharedArtifact is a collaboratively-built document (typically a plan)
// that agents propose edits to during the planning phase.
type SharedArtifact struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	Version     int            `json:"version"`
	EditHistory []ArtifactEdit `json:"edit_history,omitempty"`
	Status      ArtifactStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
