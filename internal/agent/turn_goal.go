package agent

import (
	"context"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ActionIntent is the single action decision made for a turn.
type ActionIntent string

const (
	ActionAnswer   ActionIntent = "answer"
	ActionInspect  ActionIntent = "inspect"
	ActionPlan     ActionIntent = "plan"
	ActionDebug    ActionIntent = "debug"
	ActionArtifact ActionIntent = "artifact"
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
	EvidenceAnswer          EvidenceKind = "answer"
	EvidenceArtifactCreated EvidenceKind = "artifact_created"
	EvidenceImagePosted     EvidenceKind = "image_posted"
	EvidenceEditProposed    EvidenceKind = "edit_proposed"
	EvidenceEditApplied     EvidenceKind = "edit_applied"
	EvidenceCommandRun      EvidenceKind = "command_run"
	EvidenceCommandPass     EvidenceKind = "command_passed"
	EvidenceUserAnswer      EvidenceKind = "user_answer"
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
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		return deriveTurnGoalFromDecision(msg, decision)
	}
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
	artifactRequested := UserRequestsArtifact(request)
	artifactContinuationID := ""
	if !artifactRequested && msg != nil && a != nil && userAffirmsPendingImplementation(request) {
		artifactContinuationID = pendingArtifactRequestID(a.channelHistory(msg.Channel), msg.ID)
		artifactRequested = artifactContinuationID != ""
		if artifactRequested {
			goal.ContinuationParent = artifactContinuationID
		}
	}
	activeImplementation := msg != nil && !artifactRequested && shouldRunImplementationSession(a, msg)
	goal.ImplementationSession = activeImplementation
	if activeImplementation {
		goal.Intent = IntentTask
	}
	activeCollaboration := msg != nil &&
		(msg.GetCollaborationID() != "" || strings.TrimSpace(msg.GetCollaborationPhase()) != "" ||
			msg.Type == protocol.MessageTypeCollabDiscussion)
	switch {
	case artifactRequested && !activeCollaboration:
		goal.Action = ActionArtifact
		goal.RequiredCapabilities = []string{createArtifactToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceArtifactCreated}
		goal.Mutation = MutationExternal
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

func deriveTurnGoalFromDecision(msg *protocol.Message, decision intent.TurnDecision) TurnGoal {
	request := ""
	id := "turn"
	if msg != nil {
		request = strings.Join(strings.Fields(strings.TrimSpace(msg.Content)), " ")
		if strings.TrimSpace(msg.ID) != "" {
			id = msg.ID
		}
	}
	goal := TurnGoal{
		ID:                 id,
		NormalizedRequest:  request,
		Action:             actionIntentFromSemantic(decision.Action),
		ContinuationParent: decision.ContinuationTarget,
		Mutation:           mutationLevelFromSemantic(decision.Mutation),
		Intent:             turnIntentFromSemantic(decision.Interaction),
	}
	switch goal.Action {
	case ActionInspect:
		goal.RequiredCapabilities = []string{"workspace_read"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
	case ActionPlan:
		goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
	case ActionDebug:
		goal.RequiredCapabilities = []string{"workspace_read", "run_command"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceCommandRun}
	case ActionEdit, ActionContinue:
		goal.RequiredCapabilities = []string{"workspace_edit"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
	case ActionRun:
		goal.RequiredCapabilities = []string{"run_command"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceCommandRun}
	case ActionArtifact:
		goal.RequiredCapabilities = []string{createArtifactToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceArtifactCreated}
	case ActionImage:
		goal.RequiredCapabilities = []string{generateImageToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceImagePosted}
	case ActionAskUser:
		goal.RequiredCapabilities = []string{askUserToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceUserAnswer}
	default:
		goal.Action = ActionAnswer
		goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	goal.ImplementationSession = (goal.Action == ActionDebug || goal.Action == ActionEdit || goal.Action == ActionContinue) &&
		caps.CanRunImplSession && caps.RequiresWorkspace
	if goal.ImplementationSession {
		goal.Intent = IntentTask
	}
	return goal
}

func actionIntentFromSemantic(action intent.Action) ActionIntent {
	switch action {
	case intent.ActionInspect:
		return ActionInspect
	case intent.ActionPlan:
		return ActionPlan
	case intent.ActionDebug:
		return ActionDebug
	case intent.ActionEdit:
		return ActionEdit
	case intent.ActionRun:
		return ActionRun
	case intent.ActionContinue:
		return ActionContinue
	case intent.ActionArtifact:
		return ActionArtifact
	case intent.ActionImage:
		return ActionImage
	case intent.ActionAskUser:
		return ActionAskUser
	default:
		return ActionAnswer
	}
}

func mutationLevelFromSemantic(mutation intent.Mutation) MutationLevel {
	switch mutation {
	case intent.MutationExternal:
		return MutationExternal
	case intent.MutationWorkspace:
		return MutationWorkspace
	default:
		return MutationNone
	}
}

func turnIntentFromSemantic(interaction intent.InteractionKind) TurnIntent {
	switch interaction {
	case intent.InteractionClosure:
		return IntentClosure
	case intent.InteractionCasual:
		return IntentLowSignal
	case intent.InteractionTask, intent.InteractionContinuation, intent.InteractionCorrection:
		return IntentTask
	default:
		return IntentSubstantive
	}
}

func (g TurnGoal) RequiresActionEvidence() bool {
	return g.Action != "" && g.Action != ActionAnswer && g.Action != ActionInspect && g.Action != ActionPlan
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
