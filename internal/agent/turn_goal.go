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
	applyChatModeAdvisoryGoal(msg, &goal)
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
		if intent.LooksLikeGitInspectRequest(request) {
			goal.RequiredCapabilities = append(goal.RequiredCapabilities, "run_command")
			goal.ExpectedEvidence = []EvidenceKind{EvidenceCommandRun}
		}
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
	// Explicit image requests outrank weak Answer/AskUser classifications so chat
	// cover-art turns still take the generate_image shortcut.
	if UserRequestsGeneratedImage(request) &&
		(goal.Action == ActionAnswer || goal.Action == ActionAskUser || goal.Action == ActionPlan) {
		goal.Action = ActionImage
		goal.RequiredCapabilities = []string{generateImageToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceImagePosted}
		goal.Mutation = MutationExternal
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	// Respect explicit implementation_session metadata even when RequiresWorkspace
	// was not stamped (semantic may classify as edit without setting the flag).
	explicitSession := msg != nil && msg.ImplementationSession()
	// conversation_mode=chat without an explicit session stays advisory Answer —
	// do not let semantic Edit/Debug drive workspace-mutation goals.
	if msg != nil && ConversationModeFromMessage(msg) == ConversationModeChat && !explicitSession &&
		!msg.IdeEditorModeIsExport() {
		switch goal.Action {
		case ActionEdit, ActionDebug, ActionContinue, ActionRun, ActionInspect, ActionPlan:
			goal.Action = ActionAnswer
			goal.RequiredCapabilities = nil
			goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
			goal.Mutation = MutationNone
		}
	}
	// Scenario/IDE asked for an implementation session, but the classifier often
	// under-calls boot-fix pastes as inspect/run/answer + mutation=none. Upgrade those
	// when the paste clearly needs a fix. Leave plain ActionAnswer alone so
	// artifact/canvas/chat classifications without fix cues still stick.
	if explicitSession && caps.CanRunImplSession && msg != nil &&
		!msg.IdeEditorModeIsAsk() && !msg.IdeEditorModeIsPlan() {
		bootOrFix := messageHasBootOrBuildError(request) || messageImpliesFixLikeIntent(request, nil)
		switch goal.Action {
		case ActionInspect, ActionPlan:
			if bootOrFix {
				goal.Action = ActionDebug
				goal.RequiredCapabilities = []string{"workspace_read", "workspace_edit", "run_command"}
				goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied, EvidenceCommandRun}
			} else {
				goal.Action = ActionEdit
				goal.RequiredCapabilities = []string{"workspace_edit"}
				goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
			}
			goal.Mutation = MutationWorkspace
		case ActionAnswer:
			// SoftArch/vite and selection-extract pastes often classify as Answer.
			// With an explicit implementation_session, upgrade to a workspace mutation
			// action so turnGoalRunsImplementationSession matches shouldRunImplementationSession.
			if UserRequestsArtifact(request) {
				break
			}
			if bootOrFix {
				goal.Action = ActionDebug
				goal.RequiredCapabilities = []string{"workspace_read", "workspace_edit", "run_command"}
				goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied, EvidenceCommandRun}
			} else {
				goal.Action = ActionEdit
				goal.RequiredCapabilities = []string{"workspace_edit"}
				goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
			}
			goal.Mutation = MutationWorkspace
		case ActionRun:
			// Classifier often stamps run+mutation=none for boot-fix pastes; hub already
			// treats Run as impl-worthy when implementation_session is set.
			if goal.Mutation != MutationWorkspace {
				goal.Mutation = MutationWorkspace
			}
			if bootOrFix {
				goal.Action = ActionDebug
				goal.RequiredCapabilities = []string{"workspace_read", "workspace_edit", "run_command"}
				goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied, EvidenceCommandRun}
			}
		}
	}
	goal.ImplementationSession = (goal.Action == ActionDebug || goal.Action == ActionEdit ||
		goal.Action == ActionContinue || goal.Action == ActionRun) &&
		caps.CanRunImplSession && (caps.RequiresWorkspace || explicitSession)
	if msg != nil && (msg.IdeEditorModeIsAsk() || msg.IdeEditorModeIsPlan()) {
		goal.ImplementationSession = false
	}
	if goal.ImplementationSession {
		goal.Intent = IntentTask
	}
	applyChatModeAdvisoryGoal(msg, &goal)
	return goal
}

// applyChatModeAdvisoryGoal forces advisory conversation_mode=chat turns and
// hypothetical design questions to Answer. Hub may stamp implementation_session for
// semantic Edit; advisory questions still stay conversational unless export mode.
func applyChatModeAdvisoryGoal(msg *protocol.Message, goal *TurnGoal) {
	if msg == nil || goal == nil {
		return
	}
	if msg.IdeEditorModeIsExport() {
		return
	}
	advisoryQuestion := isAdvisoryImplementationQuestion(goal.NormalizedRequest) ||
		isAdvisoryImplementationQuestion(msg.Content)
	advisoryChat := ConversationModeFromMessage(msg) == ConversationModeChat &&
		!msg.ImplementationSession()
	// Hub-stamped implementation_session must not veto advisory questions — the stamp
	// is semantic promotion, not an explicit client "ship it" opt-in.
	if advisoryQuestion {
		advisoryChat = true
	}
	if !advisoryChat && !advisoryQuestion {
		return
	}
	request := goal.NormalizedRequest
	if strings.TrimSpace(request) == "" && msg != nil {
		request = msg.Content
	}
	// Keep real image/artifact asks; demote classifier false-positives (e.g. "Design a
	// theme settings flow" classified as ActionImage) so chat scenarios get prose.
	switch goal.Action {
	case ActionImage:
		if UserRequestsGeneratedImage(request) {
			return
		}
	case ActionArtifact:
		if UserRequestsArtifact(request) {
			return
		}
	}
	goal.Action = ActionAnswer
	goal.Mutation = MutationNone
	goal.RequiredCapabilities = nil
	goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
	goal.ImplementationSession = false
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
	if g.Action == "" || g.Action == ActionAnswer || g.Action == ActionPlan {
		return false
	}
	if g.Action == ActionInspect {
		for _, kind := range g.ExpectedEvidence {
			if kind == EvidenceCommandRun {
				return true
			}
		}
		return false
	}
	return true
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
