package agent

import (
	"context"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ActionIntent is the single action decision made for a turn.
type ActionIntent string

const (
	ActionAnswer   ActionIntent = "answer"
	ActionImage    ActionIntent = "image"
	ActionEdit     ActionIntent = "edit"
	ActionRun      ActionIntent = "run"
	ActionContinue ActionIntent = "continue"
	ActionAskUser  ActionIntent = "ask_user"
)

// MutationLevel describes the strongest side effect expected for a goal.
type MutationLevel string

const (
	MutationNone      MutationLevel = "none"
	MutationExternal  MutationLevel = "external"
	MutationWorkspace MutationLevel = "workspace"
)

// EvidenceKind identifies authoritative outcomes that can support response claims.
type EvidenceKind string

const (
	EvidenceAnswer       EvidenceKind = "answer"
	EvidenceImagePosted  EvidenceKind = "image_posted"
	EvidenceEditProposed EvidenceKind = "edit_proposed"
	EvidenceEditApplied  EvidenceKind = "edit_applied"
	EvidenceCommandRun   EvidenceKind = "command_run"
	EvidenceCommandPass  EvidenceKind = "command_passed"
	EvidenceUserAnswer   EvidenceKind = "user_answer"
)

// TurnGoal is immutable per-turn routing data. Persistence and trust routing can
// consume it later without changing how the goal is derived.
type TurnGoal struct {
	ID                    string         `json:"id"`
	NormalizedRequest     string         `json:"normalized_request"`
	Action                ActionIntent   `json:"action"`
	RequiredCapabilities  []string       `json:"required_capabilities,omitempty"`
	ExpectedEvidence      []EvidenceKind `json:"expected_evidence,omitempty"`
	ContinuationParent    string         `json:"continuation_parent,omitempty"`
	Mutation              MutationLevel  `json:"mutation"`
	ImplementationSession bool           `json:"implementation_session,omitempty"`
	Intent                TurnIntent     `json:"-"`
}

var explicitRunRequestRE = regexp.MustCompile(`(?i)\b(run|execute)\s+(the\s+)?(tests?|test suite|build|command|script|lint|checks?)\b|\b(test|build|lint)\s+(it|this|the project|the code)\b`)
var explicitAskUserRequestRE = regexp.MustCompile(`(?i)^\s*(ask me|question me|clarify with me)\b`)

func deriveTurnGoal(a *Agent, msg *protocol.Message, intent TurnIntent) TurnGoal {
	request := ""
	id := ""
	parent := ""
	if msg != nil {
		request = strings.Join(strings.Fields(strings.TrimSpace(msg.Content)), " ")
		id = strings.TrimSpace(msg.ID)
		parent = strings.TrimSpace(msg.ReplyTo)
	}
	if id == "" {
		id = "turn"
	}
	goal := TurnGoal{
		ID:                 id,
		NormalizedRequest:  request,
		Action:             ActionAnswer,
		ExpectedEvidence:   []EvidenceKind{EvidenceAnswer},
		ContinuationParent: parent,
		Mutation:           MutationNone,
		Intent:             intent,
	}
	activeImplementation := msg != nil && shouldRunImplementationSession(a, msg)
	goal.ImplementationSession = activeImplementation
	activeCollaboration := msg != nil &&
		(msg.GetCollaborationID() != "" || strings.TrimSpace(msg.GetCollaborationPhase()) != "" ||
			msg.Type == protocol.MessageTypeCollabDiscussion)
	switch {
	case activeImplementation:
		goal.Action = ActionEdit
		if userAffirmsPendingImplementation(request) {
			goal.Action = ActionContinue
		}
		goal.RequiredCapabilities = []string{"workspace_edit"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
		goal.Mutation = MutationWorkspace
	case activeCollaboration && UserRequestsGeneratedImage(request):
		// Collaboration orchestration owns active phases; image intent cannot
		// steal the turn from planning, review, or execution.
	case UserRequestsGeneratedImage(request):
		goal.Action = ActionImage
		goal.RequiredCapabilities = []string{generateImageToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceImagePosted}
		goal.Mutation = MutationExternal
	case explicitAskUserRequestRE.MatchString(request):
		goal.Action = ActionAskUser
		goal.RequiredCapabilities = []string{askUserToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceUserAnswer}
	case userAffirmsPendingImplementation(request) && (parent != "" || (a != nil &&
		(channelHasRecentImplementationAsk(a.channelHistory(msg.Channel), msg.ID) ||
			channelHasRecentImplementationActivity(a.channelHistory(msg.Channel), msg.ID, a.Info.ID)))):
		goal.Action = ActionContinue
		goal.RequiredCapabilities = []string{"workspace_edit"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
		goal.Mutation = MutationWorkspace
	case explicitRunRequestRE.MatchString(request):
		goal.Action = ActionRun
		goal.RequiredCapabilities = []string{"run_command"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceCommandRun}
		goal.Mutation = MutationWorkspace
	case userRequestsImplementationForMessage(a, msg) || userRequestsImplementation(request):
		goal.Action = ActionEdit
		goal.RequiredCapabilities = []string{"workspace_edit"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
		goal.Mutation = MutationWorkspace
	}
	return goal
}

func (g TurnGoal) RequiresActionEvidence() bool {
	return g.Action != "" && g.Action != ActionAnswer
}

func turnGoalRunsImplementationSession(goal TurnGoal) bool {
	return goal.ImplementationSession
}

type turnGoalContextKey struct{}

func contextWithTurnGoal(ctx context.Context, goal TurnGoal) context.Context {
	return context.WithValue(ctx, turnGoalContextKey{}, goal)
}

func turnGoalFromContext(ctx context.Context) (TurnGoal, bool) {
	if ctx == nil {
		return TurnGoal{}, false
	}
	goal, ok := ctx.Value(turnGoalContextKey{}).(TurnGoal)
	return goal, ok
}

func turnIntentForContext(ctx context.Context, a *Agent, msg *protocol.Message) TurnIntent {
	if goal, ok := turnGoalFromContext(ctx); ok {
		return goal.Intent
	}
	return a.classifyTurnIntentForMessage(msg)
}
