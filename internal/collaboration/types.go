package collaboration

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CollaborationPhase represents the current lifecycle phase
type CollaborationPhase string

const (
	PhasePlanning  CollaborationPhase = "planning"
	PhaseReviewing CollaborationPhase = "reviewing"
	PhaseApproved  CollaborationPhase = "approved"
	PhaseExecuting CollaborationPhase = "executing"
	PhaseCompleted CollaborationPhase = "completed"
	PhaseCancelled CollaborationPhase = "cancelled"
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
	MaxRounds          int           `json:"max_rounds"`
	TurnBudget         int           `json:"turn_budget"`
	MaxTotalMessages   int           `json:"max_total_messages"`
	Timeout            time.Duration `json:"timeout"`
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
)

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
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Phase       CollaborationPhase `json:"phase"`
	Agents      []CollaborationAgent `json:"agents"`
	Plan        *SharedArtifact    `json:"plan,omitempty"`
	Tasks       []CollaborationTask `json:"tasks,omitempty"`
	Discussion  *DiscussionSession `json:"discussion,omitempty"`
	Channel     string             `json:"channel"`
	ThreadID    string             `json:"thread_id"`
	CreatedBy   string             `json:"created_by"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Config      DiscussionConfig   `json:"config"`
}

// CollaborationTask is a unit of work assigned to a specific agent
// within a collaboration. Tasks are produced during the planning phase
// and executed after user approval.
type CollaborationTask struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	AssignedTo   string     `json:"assigned_to"`
	AssignedName string     `json:"assigned_name"`
	Status       TaskStatus `json:"status"`
	Dependencies []string   `json:"dependencies,omitempty"`
	Output       string     `json:"output,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
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
