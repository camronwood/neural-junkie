package agent

import (
	"context"
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
	ActionMusic    ActionIntent = "music"
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
	EvidenceMusicPosted     EvidenceKind = "music_posted"
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

func deriveTurnGoal(a *Agent, msg *protocol.Message, intent TurnIntent) TurnGoal {
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		goal := deriveTurnGoalFromDecision(msg, decision)
		syncTurnGoalImplementationSession(a, msg, &goal)
		return goal
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
	// No canonical semantic decision was stamped for this turn. Without a stamp,
	// the only authoritative signals left are structural: an active implementation
	// session or explicit composer export mode. shouldRunImplementationSession is
	// the single source of truth for that gate (see implementation_session.go).
	activeImplementation := msg != nil && shouldRunImplementationSession(a, msg)
	goal.ImplementationSession = activeImplementation
	if activeImplementation {
		goal.Intent = IntentTask
		goal.Action = ActionEdit
		goal.RequiredCapabilities = []string{"workspace_edit"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
		goal.Mutation = MutationWorkspace
	} else if parent != "" && a != nil && a.Hub != nil {
		// Structural continuation mirror of intent.structuralIntent's
		// PendingActionID != "" && ReplyTarget != "" rule: an explicit reply to a
		// message the hub still tracks as an open conversation goal, not phrase matching.
		if resolver, ok := a.Hub.(conversationGoalResolver); ok &&
			resolver.ResolveConversationGoalID(msg.Channel, "") != "" {
			goal.Intent = IntentTask
			goal.Action = ActionContinue
			goal.RequiredCapabilities = []string{"workspace_edit"}
			goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
			goal.Mutation = MutationWorkspace
		}
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
		goal.RequiredCapabilities = []string{"workspace_read", "run_command"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceCommandRun}
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
		if decisionHasReason(decision, "maps_route") {
			goal.RequiredCapabilities = []string{mapsCreateToolName}
		} else {
			goal.RequiredCapabilities = []string{createArtifactToolName, updateArtifactToolName}
		}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceArtifactCreated}
	case ActionImage:
		goal.RequiredCapabilities = []string{generateImageToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceImagePosted}
	case ActionMusic:
		goal.RequiredCapabilities = []string{generateMusicToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceMusicPosted}
		goal.Mutation = MutationExternal
	case ActionAskUser:
		goal.RequiredCapabilities = []string{askUserToolName}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceUserAnswer}
	default:
		goal.Action = ActionAnswer
		goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
	}
	if goal.Action == ActionArtifact {
		goal.Mutation = MutationExternal
		goal.ImplementationSession = false
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	// Respect explicit implementation_session metadata even when RequiresWorkspace
	// was not stamped (semantic may classify as edit without setting the flag).
	explicitSession := msg != nil && msg.ImplementationSession()
	// conversation_mode=chat without an explicit session stays advisory Answer —
	// do not let semantic Edit/Debug drive workspace-mutation goals.
	// Exception: stamped workspace fix/repair (debug|edit + workspace mutation with
	// failure reason codes, or clear fix-the-app wording) must not be demoted.
	if msg != nil && ConversationModeFromMessage(msg) == ConversationModeChat && !explicitSession &&
		!msg.IdeEditorModeIsExport() && !turnGoalPreservesWorkspaceFix(msg, goal) {
		switch goal.Action {
		case ActionEdit, ActionDebug, ActionContinue, ActionRun, ActionInspect, ActionPlan:
			goal.Action = ActionAnswer
			goal.RequiredCapabilities = nil
			goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
			goal.Mutation = MutationNone
		}
	}
	// Trust the stamped action — explicit implementation_session metadata only
	// promotes a turn to a running implementation session when the classifier
	// already selected a workspace mutation action (Edit/Debug/Continue/Run).
	// It must never phrase-match its way from Inspect/Plan/Answer into Debug/Edit.
	goal.ImplementationSession = (goal.Action == ActionDebug || goal.Action == ActionEdit ||
		goal.Action == ActionContinue || goal.Action == ActionRun) &&
		caps.CanRunImplSession && (caps.RequiresWorkspace || explicitSession)
	if msg != nil && (msg.IdeEditorModeIsAsk() || msg.IdeEditorModeIsPlan()) {
		goal.ImplementationSession = false
	}
	if goal.ImplementationSession {
		goal.Intent = IntentTask
		goal.Mutation = MutationWorkspace
	}
	applyChatModeAdvisoryGoal(msg, &goal)
	return goal
}

// applyChatModeAdvisoryGoal forces advisory conversation_mode=chat turns to Answer.
// Hub may stamp implementation_session for semantic Edit; chat-mode turns still stay
// conversational unless export mode. Stamped creative-media actions (image/artifact/
// music) and workspace fix/repair turns are authoritative and are never demoted.
func applyChatModeAdvisoryGoal(msg *protocol.Message, goal *TurnGoal) {
	if msg == nil || goal == nil {
		return
	}
	if msg.IdeEditorModeIsExport() {
		return
	}
	if scenarioHarnessRequestsImplementationSession(msg) {
		return
	}
	advisoryChat := ConversationModeFromMessage(msg) == ConversationModeChat &&
		!msg.ImplementationSession()
	if !advisoryChat {
		return
	}
	switch goal.Action {
	case ActionImage, ActionArtifact, ActionMusic:
		return
	}
	if turnGoalPreservesWorkspaceFix(msg, *goal) {
		return
	}
	goal.Action = ActionAnswer
	goal.Mutation = MutationNone
	goal.RequiredCapabilities = nil
	goal.ExpectedEvidence = []EvidenceKind{EvidenceAnswer}
	goal.ImplementationSession = false
}

// turnGoalPreservesWorkspaceFix reports debug/edit workspace-mutation turns that
// must survive conversation_mode=chat demotion (fix/repair/broken asks).
func turnGoalPreservesWorkspaceFix(msg *protocol.Message, goal TurnGoal) bool {
	if goal.Mutation != MutationWorkspace {
		return false
	}
	switch goal.Action {
	case ActionDebug, ActionEdit, ActionContinue, ActionRun:
	default:
		return false
	}
	if msg != nil && intent.LooksLikeWorkspaceFixAsk(msg.Content) {
		return true
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		for _, code := range decision.ReasonCodes {
			switch strings.ToLower(strings.TrimSpace(code)) {
			case "startup_failure", "runtime_failure", "build_failure", "boot_failure":
				return true
			}
		}
	}
	return false
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
	case intent.ActionMusic:
		return ActionMusic
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

// syncTurnGoalImplementationSession aligns TurnGoal with shouldRunImplementationSession so
// turn_pipeline stepGenerate and post_process metadata match the session gate.
func syncTurnGoalImplementationSession(a *Agent, msg *protocol.Message, goal *TurnGoal) {
	if a == nil || msg == nil || goal == nil {
		return
	}
	if goal.Action == ActionArtifact || goal.Action == ActionImage || goal.Action == ActionMusic {
		return
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		switch decision.Action {
		case intent.ActionArtifact, intent.ActionImage, intent.ActionMusic:
			return
		}
	}
	if !shouldRunImplementationSession(a, msg) {
		return
	}
	goal.ImplementationSession = true
	goal.Intent = IntentTask
	switch goal.Action {
	case ActionAnswer, ActionPlan, ActionInspect, ActionAskUser:
		goal.Action = ActionEdit
		goal.Mutation = MutationWorkspace
		goal.RequiredCapabilities = []string{"workspace_edit"}
		goal.ExpectedEvidence = []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}
	}
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
