package intent

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Classifier resolves natural-language meaning only. Policy and permissions are
// applied by Router after classification.
type Classifier interface {
	Classify(context.Context, TurnFeatures) (SemanticIntent, error)
	Model() string
}

type Router struct {
	Classifier    Classifier
	MinConfidence float64
	Timeout       time.Duration
}

func NewRouter(classifier Classifier, minConfidence float64) *Router {
	if minConfidence <= 0 || minConfidence > 1 {
		minConfidence = 0.65
	}
	return &Router{
		Classifier:    classifier,
		MinConfidence: minConfidence,
		// Keep this short: SendMessage waits on Resolve before agents see the turn.
		// UI echo happens first, but HTTP send and agent start still wait here.
		// Production semanticTurnRouter overrides from RoutingConfig (default 8s).
		Timeout: 8 * time.Second,
	}
}

// Resolve returns one decision. Structural directives bypass the classifier when
// they determine the action; otherwise classification is local and bounded.
func (r *Router) Resolve(ctx context.Context, features TurnFeatures) TurnDecision {
	if semantic, ok := structuralIntent(features); ok {
		return ResolvePolicy(features, semantic, SourceStructural)
	}
	if r == nil || r.Classifier == nil {
		return classifierFallback(features, "classifier_unavailable")
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	classifyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	semantic, err := r.Classifier.Classify(classifyCtx, features)
	latency := time.Since(started).Milliseconds()
	if err != nil {
		decision := classifierFallback(features, "classifier_error")
		decision.ClassifierModel = r.Classifier.Model()
		decision.ClassifierLatencyMS = latency
		decision.AbstentionReason = err.Error()
		return decision
	}
	applyExplicitSemanticDirectives(features, &semantic)
	normalizeSemanticConsistency(features, &semantic)
	if err := semantic.Validate(); err != nil {
		decision := classifierFallback(features, "classifier_invalid")
		decision.ClassifierModel = r.Classifier.Model()
		decision.ClassifierLatencyMS = latency
		decision.AbstentionReason = err.Error()
		return decision
	}
	if semantic.Confidence < r.MinConfidence {
		decision := classifierFallback(features, "classifier_low_confidence")
		decision.ClassifierModel = r.Classifier.Model()
		decision.ClassifierLatencyMS = latency
		decision.Confidence = semantic.Confidence
		decision.AbstentionReason = fmt.Sprintf("confidence %.3f below %.3f", semantic.Confidence, r.MinConfidence)
		return decision
	}
	decision := ResolvePolicy(features, semantic, SourceLocalModel)
	decision.ClassifierModel = r.Classifier.Model()
	decision.ClassifierLatencyMS = latency
	return decision
}

func normalizeSemanticConsistency(features TurnFeatures, semantic *SemanticIntent) {
	if semantic == nil {
		return
	}
	if semantic.Interaction == InteractionContinuation && features.PendingActionID != "" {
		semantic.RequestedAction = ActionContinue
		semantic.ContinuationTarget = features.PendingActionID
		if features.PendingAction != "" {
			semantic.MutationRequested = mutationForAction(features.PendingAction)
		} else {
			semantic.MutationRequested = MutationWorkspace
		}
	}
	if semantic.RequestedAction == ActionContinue {
		if semantic.ContinuationTarget == "" {
			semantic.ContinuationTarget = features.PendingActionID
		}
		if semantic.MutationRequested == MutationNone && features.PendingAction != "" {
			semantic.MutationRequested = mutationForAction(features.PendingAction)
		}
	}
	switch semantic.RequestedAction {
	case ActionArtifact, ActionImage, ActionMusic:
		semantic.MutationRequested = MutationExternal
	case ActionEdit:
		semantic.MutationRequested = MutationWorkspace
	case ActionAnswer, ActionInspect, ActionPlan, ActionAskUser:
		semantic.MutationRequested = MutationNone
	}
	if (semantic.RecipientType == "" || semantic.RecipientType == "assistant" || semantic.RecipientType == "general") &&
		(semantic.RequestedAction == ActionInspect || semantic.RequestedAction == ActionDebug ||
			semantic.RequestedAction == ActionEdit || semantic.RequestedAction == ActionRun ||
			semantic.RequestedAction == ActionContinue) {
		semantic.RecipientType = recipientForDomain(semantic.Domain)
	}
}

func recipientForDomain(domain string) string {
	switch domain {
	case "code_review":
		// Code review is a turn mode, not a specialist type — leave routing to
		// the addressed/active agent (Assistant default when none is implied).
		return "assistant"
	case "frontend", "backend", "devops", "architecture", "database", "security", "biology", "rust", "cad":
		return domain
	default:
		// Pack-extended domains map 1:1 onto recipient tokens when registered.
		rec := normalizeRecipientToken(domain)
		if CurrentOntology().ValidRecipient(rec) {
			return rec
		}
		return "assistant"
	}
}

func structuralIntent(features TurnFeatures) (SemanticIntent, bool) {
	base := SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   features.ExplicitAction,
		MutationRequested: MutationNone,
		Confidence:        1,
		ReasonCodes:       []string{"explicit_turn_directive"},
	}
	switch {
	case features.IsSlashCommand:
		base.Interaction = InteractionTask
		base.RequestedAction = ActionAnswer
		return base, true
	case features.PendingActionID != "" && features.ReplyTarget != "":
		base.Interaction = InteractionContinuation
		base.RequestedAction = ActionContinue
		base.ContinuationTarget = features.PendingActionID
		base.MutationRequested = mutationForAction(features.PendingAction)
		if base.MutationRequested == MutationNone {
			base.MutationRequested = MutationWorkspace
		}
		return base, true
	default:
		return SemanticIntent{}, false
	}
}

func applyExplicitSemanticDirectives(features TurnFeatures, semantic *SemanticIntent) {
	if semantic == nil {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(features.ComposerMode))
	switch {
	case features.ExplicitAction != "":
		semantic.RequestedAction = features.ExplicitAction
		semantic.MutationRequested = mutationForAction(features.ExplicitAction)
		semantic.ReasonCodes = append(semantic.ReasonCodes, "explicit_action")
	case mode == "plan":
		semantic.RequestedAction = ActionPlan
		semantic.MutationRequested = MutationNone
		semantic.ReasonCodes = append(semantic.ReasonCodes, "explicit_plan_mode")
	case mode == "export":
		semantic.RequestedAction = ActionEdit
		semantic.MutationRequested = MutationWorkspace
		semantic.ReasonCodes = append(semantic.ReasonCodes, "explicit_export_mode")
	}
}

// ResolvePolicy applies deterministic authorization and environment constraints.
func ResolvePolicy(features TurnFeatures, semantic SemanticIntent, source Source) TurnDecision {
	decision := TurnDecision{
		SchemaVersion:      SchemaVersion,
		Interaction:        semantic.Interaction,
		RequestedAction:    semantic.RequestedAction,
		Action:             semantic.RequestedAction,
		Domain:             strings.TrimSpace(semantic.Domain),
		RecipientType:      strings.TrimSpace(semantic.RecipientType),
		Retrieval:          append([]RetrievalTarget(nil), semantic.Retrieval...),
		Mutation:           semantic.MutationRequested,
		ContinuationTarget: strings.TrimSpace(semantic.ContinuationTarget),
		Complexity:         strings.TrimSpace(semantic.Complexity),
		Confidence:         semantic.Confidence,
		Source:             source,
		ReasonCodes:        normalizeStrings(semantic.ReasonCodes),
	}
	if decision.Interaction == "" {
		decision.Interaction = InteractionQuestion
	}
	if decision.RequestedAction == "" {
		decision.RequestedAction = ActionAnswer
		decision.Action = ActionAnswer
	}
	if decision.Mutation == "" {
		decision.Mutation = mutationForAction(decision.Action)
	}

	// Open Neural Canvas in-channel: revisions stay artifact (external), not workspace edit.
	// Structural — driven by open_artifact_* features. Meta questions graduate on
	// classifier reason codes (LooksLike* status/title promote removed).
	hasOpenCanvas := strings.TrimSpace(features.OpenArtifactID) != "" ||
		strings.TrimSpace(features.OpenArtifactRenderer) != ""
	metaCanvasQ := hasOpenCanvas &&
		!LooksLikeCanvasRenameAsk(features.Text) &&
		!LooksLikeOpenCanvasFillAsk(features.Text) &&
		hasMetaCanvasReasonCode(decision.ReasonCodes)
	if hasOpenCanvas && features.ExplicitAction == "" &&
		strings.ToLower(strings.TrimSpace(features.ComposerMode)) != "export" &&
		!pendingActionBlocksCanvasPromote(features) {
		promoteOpenCanvas := false
		if !metaCanvasQ && !PrefersChatOverOpenCanvas(features.Text) {
			switch decision.Action {
			case ActionEdit, ActionRun, ActionInspect, ActionImage,
				ActionAskUser, ActionContinue, ActionAnswer, ActionPlan:
				promoteOpenCanvas = true
			}
		}
		if promoteOpenCanvas {
			decision.Action = ActionArtifact
			decision.RequestedAction = ActionArtifact
			decision.Mutation = MutationExternal
			decision.ContinuationTarget = ""
			if decision.Interaction == InteractionContinuation || decision.Interaction == InteractionCasual ||
				decision.Interaction == InteractionQuestion {
				decision.Interaction = InteractionTask
			}
			if !slices.Contains(decision.Retrieval, RetrievalPriorReference) {
				decision.Retrieval = append(decision.Retrieval, RetrievalPriorReference)
			}
			decision.PolicyOverrides = append(decision.PolicyOverrides, "open_canvas_artifact")
			decision.ReasonCodes = append(decision.ReasonCodes, "durable_artifact")
		}
	}
	// Classifier often stamps artifact/inspect/ask_user on meta questions about an open canvas.
	if metaCanvasQ && features.ExplicitAction == "" && decision.Action != ActionAnswer {
		decision.Action = ActionAnswer
		decision.RequestedAction = ActionAnswer
		decision.Mutation = MutationNone
		decision.ContinuationTarget = ""
		decision.PolicyOverrides = append(decision.PolicyOverrides, "open_canvas_meta_demote")
	}

	// Local classifiers often emit blank_canvas / durable_artifact correctly but stamp
	// ask_user, answer, or continue. Promote those to artifact so Neural Canvas shortcuts
	// can fire — but only when user text corroborates a canvas/map deliverable. Tiny
	// local classifiers spray blank_canvas on unrelated questions (meeting notes, etc.).
	if features.ExplicitAction == "" &&
		strings.ToLower(strings.TrimSpace(features.ComposerMode)) != "export" &&
		!pendingActionBlocksCanvasPromote(features) &&
		shouldPromoteCanvasReasonCodes(decision, features.Text) {
		decision.Action = ActionArtifact
		decision.RequestedAction = ActionArtifact
		decision.Mutation = MutationExternal
		decision.ContinuationTarget = ""
		if decision.Interaction == InteractionContinuation || decision.Interaction == InteractionCasual {
			decision.Interaction = InteractionTask
		}
		decision.PolicyOverrides = append(decision.PolicyOverrides, "canvas_reason_artifact")
		if !slices.Contains(decision.ReasonCodes, "durable_artifact") {
			decision.ReasonCodes = append(decision.ReasonCodes, "durable_artifact")
		}
	}

	// "create a canvas with that information" often stamps answer + advisory_question
	// with no blank_canvas reason. Text create/fill cues are enough to promote.
	if features.ExplicitAction == "" &&
		strings.ToLower(strings.TrimSpace(features.ComposerMode)) != "export" &&
		!pendingActionBlocksCanvasPromote(features) &&
		shouldPromoteCanvasTextAsk(decision, features.Text) {
		decision.Action = ActionArtifact
		decision.RequestedAction = ActionArtifact
		decision.Mutation = MutationExternal
		decision.ContinuationTarget = ""
		if decision.Interaction == InteractionContinuation || decision.Interaction == InteractionCasual ||
			decision.Interaction == InteractionQuestion {
			decision.Interaction = InteractionTask
		}
		decision.PolicyOverrides = append(decision.PolicyOverrides, "canvas_text_artifact")
		if !slices.Contains(decision.ReasonCodes, "durable_artifact") {
			decision.ReasonCodes = append(decision.ReasonCodes, "durable_artifact")
		}
		if !slices.Contains(decision.Retrieval, RetrievalPriorReference) &&
			LooksLikePriorContentCanvasAsk(features.Text) {
			decision.Retrieval = append(decision.Retrieval, RetrievalPriorReference)
		}
	}

	// Chat-surface and meeting-note Q&A stay in-thread even when a canvas tab is
	// focused or the user said "in the chat". No-canvas meeting-note spray still
	// uses spurious_artifact_demote. Explicit fill/create ("put the notes in
	// there") still revises the page.
	if decision.Action == ActionArtifact && features.ExplicitAction == "" &&
		PrefersChatOverOpenCanvas(features.Text) &&
		(hasOpenCanvas || LooksLikeChatSurfaceAsk(features.Text)) {
		decision.Action = ActionAnswer
		decision.RequestedAction = ActionAnswer
		decision.Mutation = MutationNone
		decision.ContinuationTarget = ""
		decision.PolicyOverrides = append(decision.PolicyOverrides, "chat_surface_demote")
	}

	// Tiny local classifiers also stamp ActionArtifact + blank_canvas on unrelated
	// questions. Demote when user text does not corroborate any canvas/map/report ask
	// and there is no open canvas to revise. Skip when policy already promoted from a
	// corroborated reason/continuation path.
	if decision.Action == ActionArtifact && features.ExplicitAction == "" &&
		features.OpenArtifactRenderer == "" && features.OpenArtifactID == "" &&
		!containsString(decision.PolicyOverrides, "canvas_reason_artifact") &&
		!containsString(decision.PolicyOverrides, "canvas_text_artifact") &&
		!containsString(decision.PolicyOverrides, "open_canvas_artifact") &&
		!LooksLikeCanvasDeliverableAsk(features.Text) &&
		!LooksLikeMapsRouteAsk(features.Text) &&
		!LooksLikeWorkspaceReportAsk(features.Text) {
		decision.Action = ActionAnswer
		decision.RequestedAction = ActionAnswer
		decision.Mutation = MutationNone
		decision.ContinuationTarget = ""
		decision.PolicyOverrides = append(decision.PolicyOverrides, "spurious_artifact_demote")
	}

	// ask_user + sprayed blank_canvas on a non-canvas question (e.g. meeting notes)
	// should stay conversational answer, not a question-prompt goal.
	if decision.Action == ActionAskUser && features.ExplicitAction == "" &&
		hasCanvasDeliverableReasonCode(decision.ReasonCodes) &&
		!LooksLikeCanvasDeliverableAsk(features.Text) &&
		!LooksLikeMapsRouteAsk(features.Text) &&
		!LooksLikeWorkspaceReportAsk(features.Text) {
		decision.Action = ActionAnswer
		decision.RequestedAction = ActionAnswer
		decision.Mutation = MutationNone
		decision.ContinuationTarget = ""
		decision.PolicyOverrides = append(decision.PolicyOverrides, "spurious_canvas_ask_user_demote")
	}

	// Project overview / workspace fix / git inspect: reason_codes only (LooksLike
	// positive promote graduated). Canvas/maps text helpers remain demote-only.
	mode := strings.ToLower(strings.TrimSpace(features.ComposerMode))
	if features.ExplicitAction == "" &&
		hasReasonCode(decision.ReasonCodes, "project_overview") &&
		!LooksLikeCanvasDeliverableAsk(features.Text) &&
		!LooksLikeMapsRouteAsk(features.Text) &&
		!looksLikeExplicitCommandRunAsk(features.Text) {
		switch decision.Action {
		case ActionRun, ActionEdit, ActionContinue, ActionAskUser, ActionAnswer, ActionPlan, ActionDebug:
			decision.Action = ActionInspect
			decision.RequestedAction = ActionInspect
			decision.Mutation = MutationNone
			decision.ContinuationTarget = ""
			if decision.Interaction == InteractionContinuation || decision.Interaction == InteractionCasual {
				decision.Interaction = InteractionQuestion
			}
			decision.PolicyOverrides = append(decision.PolicyOverrides, "project_overview_inspect")
		}
	}

	if features.ExplicitAction == "" &&
		mode != "ask" && mode != "plan" &&
		features.HasWorkspace &&
		hasFailureFixReasonCodes(decision.ReasonCodes) &&
		!LooksLikeCanvasDeliverableAsk(features.Text) &&
		!LooksLikeMapsRouteAsk(features.Text) &&
		!hasReasonCode(decision.ReasonCodes, "project_overview") &&
		!LooksLikeProjectOverviewAsk(features.Text) {
		switch decision.Action {
		case ActionAnswer, ActionPlan, ActionInspect, ActionAskUser, ActionRun, ActionContinue:
			decision.Action = ActionDebug
			decision.RequestedAction = ActionDebug
			decision.Mutation = MutationWorkspace
			if decision.Interaction == InteractionCasual || decision.Interaction == InteractionQuestion {
				decision.Interaction = InteractionTask
			}
			if !hasFailureFixReasonCodes(decision.ReasonCodes) {
				decision.ReasonCodes = append(decision.ReasonCodes, "runtime_failure")
			}
			decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_fix_promote")
		case ActionDebug, ActionEdit:
			if decision.Mutation != MutationWorkspace {
				decision.Mutation = MutationWorkspace
				decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_fix_mutation")
			}
			if !hasFailureFixReasonCodes(decision.ReasonCodes) {
				decision.ReasonCodes = append(decision.ReasonCodes, "runtime_failure")
			}
		case ActionArtifact, ActionImage, ActionMusic:
			decision.Action = ActionDebug
			decision.RequestedAction = ActionDebug
			decision.Mutation = MutationWorkspace
			decision.Interaction = InteractionTask
			decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_fix_promote")
		}
	}

	// git_inspect: reason code only (LooksLikeGitInspectRequest promote removed).
	if features.ExplicitAction == "" &&
		mode != "ask" && mode != "plan" &&
		hasReasonCode(decision.ReasonCodes, "git_inspect") {
		switch decision.Action {
		case ActionRun, ActionEdit, ActionDebug, ActionAnswer, ActionAskUser, ActionContinue, ActionPlan:
			decision.Action = ActionInspect
			decision.RequestedAction = ActionInspect
			decision.Mutation = MutationNone
			decision.ContinuationTarget = ""
			decision.PolicyOverrides = append(decision.PolicyOverrides, "git_inspect_promote")
		}
	}

	// Direct implementation asks mis-stamped as plan (no execution) become edit.
	if features.ExplicitAction == "" &&
		mode != "ask" && mode != "plan" &&
		features.HasWorkspace &&
		decision.Action == ActionPlan &&
		LooksLikeDirectImplementationAsk(features.Text) {
		decision.Action = ActionEdit
		decision.RequestedAction = ActionEdit
		decision.Mutation = MutationWorkspace
		decision.Interaction = InteractionTask
		decision.PolicyOverrides = append(decision.PolicyOverrides, "implementation_plan_promote")
	}

	// Prior-content canvas creates need prior_reference even when the stamp is already artifact.
	if decision.Action == ActionArtifact && features.ExplicitAction == "" &&
		LooksLikePriorContentCanvasAsk(features.Text) &&
		!slices.Contains(decision.Retrieval, RetrievalPriorReference) {
		decision.Retrieval = append(decision.Retrieval, RetrievalPriorReference)
	}

	// Repo fact questions ("which package declares X in this repo?") need inspect + codebase,
	// not chat-only answer. Reason code is primary; narrow text fallback until live holds.
	if features.ExplicitAction == "" &&
		features.HasWorkspace &&
		mode != "ask" && mode != "plan" &&
		decision.Action == ActionAnswer &&
		(hasReasonCode(decision.ReasonCodes, "repo_fact") || LooksLikeRepoFactAsk(features.Text)) &&
		!LooksLikeCanvasDeliverableAsk(features.Text) &&
		!LooksLikeProjectOverviewAsk(features.Text) {
		decision.Action = ActionInspect
		decision.RequestedAction = ActionInspect
		decision.Mutation = MutationNone
		decision.ContinuationTarget = ""
		if decision.Interaction == InteractionCasual || decision.Interaction == InteractionClosure {
			decision.Interaction = InteractionQuestion
		}
		if !hasReasonCode(decision.ReasonCodes, "repo_fact") {
			decision.ReasonCodes = append(decision.ReasonCodes, "repo_fact")
		}
		decision.PolicyOverrides = append(decision.PolicyOverrides, "repo_fact_inspect")
	}

	if mode == "ask" || mode == "plan" {
		askForbidsTool := mode == "ask" && decision.Action != ActionAnswer &&
			(isMutatingAction(decision.Action) ||
				decision.Action == ActionRun || decision.Action == ActionDebug ||
				decision.Action == ActionInspect || decision.Action == ActionPlan ||
				decision.Action == ActionContinue || decision.Action == ActionAskUser ||
				decision.Action == ActionEdit || decision.Action == ActionArtifact ||
				decision.Action == ActionImage || decision.Action == ActionMusic)
		if decision.Mutation != MutationNone || isMutatingAction(decision.Action) || askForbidsTool {
			decision.PolicyOverrides = append(decision.PolicyOverrides, "composer_mode_forbids_mutation")
		}
		decision.Mutation = MutationNone
		if mode == "plan" {
			decision.Action = ActionPlan
		} else if isMutatingAction(decision.Action) || askForbidsTool {
			decision.Action = ActionAnswer
			decision.RequestedAction = ActionAnswer
		}
	}
	if decision.Mutation == MutationWorkspace {
		// Unauthorized workspace mutation always demotes to ask_user — no creative-phrase
		// branch that second-guesses the stamped action with text heuristics.
		if !features.CanProposeFiles || !features.CanRunImplementation {
			decision.Action = ActionAskUser
			decision.Mutation = MutationNone
			decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_mutation_not_authorized")
		} else if !features.HasWorkspace {
			decision.Action = ActionAskUser
			decision.Mutation = MutationNone
			decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_required")
		}
	}
	if decision.Action == ActionContinue {
		pendingID := strings.TrimSpace(features.PendingActionID)
		if decision.ContinuationTarget == "" {
			decision.ContinuationTarget = pendingID
		}
		// Soft dialogue follow-ups often stamp continue (sometimes with an invented
		// target) when no pending action exists. Answer in-thread instead.
		if pendingID == "" {
			decision.Action = ActionAnswer
			decision.RequestedAction = ActionAnswer
			decision.Mutation = MutationNone
			decision.ContinuationTarget = ""
			if decision.Interaction == InteractionContinuation {
				decision.Interaction = InteractionQuestion
			}
			decision.PolicyOverrides = append(decision.PolicyOverrides, "continuation_target_missing")
		}
	}

	// Inspect requires a workspace. Without one, treat as ordinary chat answer
	// (e.g. meeting-notes / memory questions mis-stamped as inspect).
	if decision.Action == ActionInspect && features.ExplicitAction == "" && !features.HasWorkspace {
		decision.Action = ActionAnswer
		decision.RequestedAction = ActionAnswer
		decision.Mutation = MutationNone
		decision.PolicyOverrides = append(decision.PolicyOverrides, "inspect_requires_workspace")
	}

	// Tiny classifiers spray ask_user on ordinary chat. Demote when there is no named
	// ambiguity and policy did not force ask_user for auth/workspace gates.
	if decision.Action == ActionAskUser && features.ExplicitAction == "" &&
		!containsString(decision.PolicyOverrides, "workspace_mutation_not_authorized") &&
		!containsString(decision.PolicyOverrides, "workspace_required") &&
		len(normalizeStrings(semantic.Ambiguities)) == 0 &&
		(decision.Interaction == InteractionCasual ||
			decision.Interaction == InteractionClosure ||
			decision.Interaction == InteractionQuestion ||
			decision.Interaction == InteractionContinuation ||
			decision.Interaction == InteractionCorrection) {
		decision.Action = ActionAnswer
		decision.RequestedAction = ActionAnswer
		decision.Mutation = MutationNone
		decision.PolicyOverrides = append(decision.PolicyOverrides, "spurious_ask_user_demote")
	}

	if features.HasWorkspace {
		switch decision.Action {
		case ActionInspect, ActionDebug, ActionEdit, ActionRun, ActionContinue:
			if !containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
				decision.Retrieval = append(decision.Retrieval, RetrievalCodebase)
				decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_requires_codebase_retrieval")
			}
		}
	}
	// Classifiers sometimes pair run with external mutation (creative spray).
	if decision.Action == ActionRun && decision.Mutation == MutationExternal {
		decision.Mutation = MutationNone
		decision.PolicyOverrides = append(decision.PolicyOverrides, "run_mutation_normalize")
	}
	decision.PolicyOverrides = normalizeStrings(decision.PolicyOverrides)
	return decision
}

// LooksLikeGitInspectRequest reports user asks that should be answered with live
// git tool output (status/log/diff/history), not chat speculation.
func LooksLikeGitInspectRequest(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	cues := []string{
		"check git",
		"use git",
		"via git",
		"from git",
		"git to find",
		"git history",
		"git log",
		"git diff",
		"git status",
		"git show",
		"against git",
		"compare with git",
		"known-good",
		"known good",
		"working config",
		"last known good",
		"what changed in git",
		"what did git",
	}
	for _, cue := range cues {
		if strings.Contains(t, cue) {
			return true
		}
	}
	return false
}

// LooksLikeRepoFactAsk reports factual repo questions that need inspect + codebase
// retrieval (dependency ownership, which package/file declares X) — not chat-only answer.
func LooksLikeRepoFactAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	repoCue := strings.Contains(c, "repo") || strings.Contains(c, "codebase") ||
		strings.Contains(c, "workspace") || strings.Contains(c, "package.json") ||
		strings.Contains(c, "go.mod") || strings.Contains(c, "cargo.toml")
	factCue := strings.Contains(c, "which package") || strings.Contains(c, "which file") ||
		strings.Contains(c, "where is") || strings.Contains(c, "where does") ||
		strings.Contains(c, "declares") || strings.Contains(c, "dependency") ||
		strings.Contains(c, "depends on")
	return repoCue && factCue
}

// LooksLikeDirectImplementationAsk reports concrete "do the work" asks that must not
// stay as plan. Explicit planning phrasing is excluded.
func LooksLikeDirectImplementationAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	if strings.Contains(c, "propose a plan") || strings.Contains(c, "how should we") ||
		strings.Contains(c, "what's the approach") || strings.Contains(c, "what is the approach") ||
		strings.Contains(c, "draft a plan") || strings.HasPrefix(c, "plan ") {
		return false
	}
	verbs := []string{
		"add a ", "add an ", "create a ", "create an ", "implement ", "write a ", "write an ",
		"build a ", "build an ", "insert a ", "insert an ",
	}
	for _, v := range verbs {
		if strings.HasPrefix(c, v) || strings.Contains(c, " "+v) {
			return true
		}
	}
	return false
}

// shouldPromoteCanvasReasonCodes reports mis-stamped turns where the classifier
// named a canvas/map deliverable via reason_codes but chose the wrong action.
// Reason codes alone are not enough — qwen2.5:3b sprays blank_canvas on unrelated
// asks — so user text must corroborate the deliverable kind.
func shouldPromoteCanvasReasonCodes(decision TurnDecision, text string) bool {
	switch decision.Action {
	case ActionEdit, ActionDebug, ActionArtifact, ActionImage, ActionMusic:
		return false
	}
	var blank, durable, report, maps bool
	for _, code := range decision.ReasonCodes {
		switch strings.ToLower(strings.TrimSpace(code)) {
		case "blank_canvas":
			blank = true
		case "durable_artifact":
			durable = true
		case "workspace_report":
			report = true
		case "maps_route":
			maps = true
		}
	}
	if maps && LooksLikeMapsRouteAsk(text) {
		return true
	}
	if report && LooksLikeWorkspaceReportAsk(text) {
		return true
	}
	if blank || durable {
		if LooksLikeCanvasDeliverableAsk(text) {
			return true
		}
		// "ok please do that now" after a blank-canvas create: text has no "canvas",
		// but continuation + blank_canvas without junk failure codes is enough.
		if blank && decision.Interaction == InteractionContinuation &&
			!hasSpuriousClassifierFailureCodes(decision.ReasonCodes) {
			return true
		}
	}
	return false
}

func hasSpuriousClassifierFailureCodes(codes []string) bool {
	return hasFailureFixReasonCodes(codes)
}

func hasFailureFixReasonCodes(codes []string) bool {
	for _, code := range codes {
		switch strings.ToLower(strings.TrimSpace(code)) {
		case "startup_failure", "runtime_failure", "build_failure":
			return true
		}
	}
	return false
}

func hasMetaCanvasReasonCode(codes []string) bool {
	return hasReasonCode(codes, "canvas_meta_question") || hasReasonCode(codes, "canvas_title_question")
}

func hasReasonCode(codes []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return false
	}
	for _, code := range codes {
		if strings.EqualFold(strings.TrimSpace(code), want) {
			return true
		}
	}
	return false
}

func hasCanvasDeliverableReasonCode(codes []string) bool {
	for _, code := range codes {
		switch strings.ToLower(strings.TrimSpace(code)) {
		case "blank_canvas", "durable_artifact", "workspace_report", "maps_route":
			return true
		}
	}
	return false
}

// PrefersChatOverOpenCanvas reports turns that must stay in-thread even when a
// Neural Canvas tab is focused. Explicit canvas fill/create still revises the page.
func PrefersChatOverOpenCanvas(text string) bool {
	if LooksLikeChatSurfaceAsk(text) {
		return true
	}
	if LooksLikeMeetingNotesAsk(text) &&
		!LooksLikeOpenCanvasFillAsk(text) &&
		!LooksLikeCanvasDeliverableAsk(text) {
		return true
	}
	return false
}

// LooksLikeChatSurfaceAsk reports the user wants the reply in the chat thread,
// not as a Neural Canvas artifact.
func LooksLikeChatSurfaceAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	cues := []string{
		"in the chat", "in chat", "in this chat", "in this thread",
		"as text", "as a message", "just text", "just the text",
		"show me text", "show me the text",
		"don't use the canvas", "dont use the canvas",
		"without a canvas", "without the canvas",
		"no canvas", "not on the canvas", "not in the canvas",
		"don't create a canvas", "dont create a canvas",
		"don't make a canvas", "dont make a canvas",
	}
	for _, cue := range cues {
		if strings.Contains(c, cue) {
			return true
		}
	}
	return false
}

// LooksLikeMeetingNotesAsk reports questions about synced meeting notes.
func LooksLikeMeetingNotesAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	cues := []string{
		"meeting notes", "meeting note", "meeting summary",
		"summarize notes", "summarise notes",
		"last meeting", "recent meeting", "today's meeting", "todays meeting",
		"notes from today", "notes from my last", "notes from the last",
	}
	for _, cue := range cues {
		if strings.Contains(c, cue) {
			return true
		}
	}
	return strings.Contains(c, "meeting") && strings.Contains(c, "notes")
}

// LooksLikeCanvasDeliverableAsk reports user text that asks to create/open a Neural
// Canvas or related durable visual artifact (not meeting notes, chat Q&A, etc.).
func LooksLikeCanvasDeliverableAsk(text string) bool {
	c := normalizeCanvasTypos(strings.ToLower(strings.TrimSpace(text)))
	if c == "" {
		return false
	}
	if strings.Contains(c, "neural canvas") || strings.Contains(c, "canvas") ||
		strings.Contains(c, "artifact") || strings.Contains(c, "mermaid") ||
		strings.Contains(c, "diagram") || strings.Contains(c, "timeline") {
		return true
	}
	if strings.Contains(c, "chart") && !strings.Contains(c, "org chart") {
		return true
	}
	return false
}

// normalizeCanvasTypos maps common misspellings onto "canvas" so create asks still route.
func normalizeCanvasTypos(text string) string {
	if text == "" {
		return text
	}
	replacer := strings.NewReplacer(
		"canvans", "canvas",
		"canvass", "canvas",
		"canvus", "canvas",
		"canva ", "canvas ",
	)
	return replacer.Replace(text)
}

// pendingActionBlocksCanvasPromote keeps bare continuations ("ok") from stealing an
// in-flight edit/debug turn — but explicit canvas create/fill asks still promote.
func pendingActionBlocksCanvasPromote(features TurnFeatures) bool {
	if features.PendingAction != ActionEdit && features.PendingAction != ActionDebug {
		return false
	}
	return !LooksLikeCanvasCreateOrFillAsk(features.Text)
}

// LooksLikeCanvasCreateOrFillAsk reports explicit create/fill/post canvas asks.
func LooksLikeCanvasCreateOrFillAsk(text string) bool {
	if LooksLikeCanvasStatusQuestion(text) || LooksLikeCanvasTitleQuestion(text) {
		return false
	}
	if LooksLikeMapsRouteAsk(text) || LooksLikeWorkspaceReportAsk(text) {
		return true
	}
	if !LooksLikeCanvasDeliverableAsk(text) {
		return false
	}
	c := normalizeCanvasTypos(strings.ToLower(strings.TrimSpace(text)))
	for _, verb := range []string{
		"create", "make", "generate", "build", "produce", "render",
		"open a", "open new", "add ", "put ", "fill", "post ",
	} {
		if strings.Contains(c, verb) {
			return true
		}
	}
	return strings.Contains(c, "new canvas") || strings.Contains(c, "blank canvas") ||
		strings.Contains(c, "new neural canvas") || LooksLikePriorContentCanvasAsk(text)
}

// LooksLikeMapsRouteAsk reports geographic map/route/directions asks.
func LooksLikeMapsRouteAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	return strings.Contains(c, "map") || strings.Contains(c, "route") ||
		strings.Contains(c, "directions") || strings.Contains(c, "drive from") ||
		strings.Contains(c, "driving from") || strings.Contains(c, "walk from") ||
		strings.Contains(c, "walking from")
}

// LooksLikeWorkspaceReportAsk reports asks for a durable project/workspace report.
func LooksLikeWorkspaceReportAsk(text string) bool {
	c := normalizeOverviewTypos(normalizeCanvasTypos(strings.ToLower(strings.TrimSpace(text))))
	if c == "" {
		return false
	}
	if !(strings.Contains(c, "report") || strings.Contains(c, "writeup") ||
		strings.Contains(c, "write-up") || strings.Contains(c, "brief") ||
		strings.Contains(c, "summary") || strings.Contains(c, "summarize")) {
		return false
	}
	// Chat-only "summarize the project" is inspect, not a durable report artifact.
	if !strings.Contains(c, "report") && !strings.Contains(c, "writeup") &&
		!strings.Contains(c, "write-up") && !strings.Contains(c, "brief") &&
		!strings.Contains(c, "canvas") && !strings.Contains(c, "artifact") {
		return false
	}
	return strings.Contains(c, "project") || strings.Contains(c, "workspace") ||
		strings.Contains(c, "repo") || strings.Contains(c, "codebase") ||
		strings.Contains(c, "canvas") || strings.Contains(c, "artifact")
}

// LooksLikeWorkspaceFixAsk reports imperative requests to fix/repair a broken app
// or workspace failure. Advisory "how would you fix…" questions return false.
func LooksLikeWorkspaceFixAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	advisory := []string{
		"how would you", "how should i", "how do i", "how can i", "what would you",
		"what's the best way", "whats the best way", "could you explain how",
	}
	for _, cue := range advisory {
		if strings.Contains(c, cue) {
			return false
		}
	}
	fixCues := []string{
		"fix the app", "fix this app", "fix it", "fix the bug", "fix this",
		"repair the", "repair it", "sort out the", "sort this out", "sort it out",
		"can you fix", "please fix", "fix this for me", "make it work",
		"get it working", "get the app working",
	}
	for _, cue := range fixCues {
		if strings.Contains(c, cue) {
			return true
		}
	}
	failureCues := []string{
		"broken", "not working", "doesn't work", "does not work", "won't boot",
		"wont boot", "not booting", "won't start", "wont start", "blank screen",
		"white screen", "crash", "crashing", "failing to start", "failed to start",
		"exits before", "never reaches the ui", "something is wrong with this project",
		"something is wrong with the project",
	}
	hasFailure := false
	for _, cue := range failureCues {
		if strings.Contains(c, cue) {
			hasFailure = true
			break
		}
	}
	if !hasFailure {
		return false
	}
	repairVerbs := []string{"fix", "repair", "patch", "resolve", "sort out", "diagnose"}
	for _, verb := range repairVerbs {
		if strings.Contains(c, verb) {
			return true
		}
	}
	return false
}

// LooksLikeProjectOverviewAsk reports chat asks to review/summarize/explain the
// open project, workspace, repo, or codebase (not canvas deliverables).
func LooksLikeProjectOverviewAsk(text string) bool {
	c := normalizeOverviewTypos(normalizeCanvasTypos(strings.ToLower(strings.TrimSpace(text))))
	if c == "" {
		return false
	}
	if LooksLikeCanvasCreateOrFillAsk(text) || LooksLikeMapsRouteAsk(text) {
		return false
	}
	verbs := []string{
		"summarize", "summary", "review", "overview", "describe", "explain",
		"walk me through", "walk through", "what is this project", "what's this project",
		"what does this project", "tell me about the project", "tell me about this project",
	}
	hasVerb := false
	for _, verb := range verbs {
		if strings.Contains(c, verb) {
			hasVerb = true
			break
		}
	}
	if !hasVerb {
		return false
	}
	objects := []string{
		"project", "workspace", "repo", "repository", "codebase",
		"i have open", "have open", "open project", "this project",
	}
	for _, object := range objects {
		if strings.Contains(c, object) {
			return true
		}
	}
	return false
}

func normalizeOverviewTypos(text string) string {
	if text == "" {
		return text
	}
	return strings.NewReplacer(
		"summerize", "summarize",
		"sumarize", "summarize",
		"summarise", "summarize",
	).Replace(text)
}

func looksLikeExplicitCommandRunAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	cues := []string{
		"run the tests", "run tests", "npm run", "cargo run", "make run",
		"execute the", "run the command", "run this command", "run the build",
		"run the app", "start the server", "launch the",
	}
	for _, cue := range cues {
		if strings.Contains(c, cue) {
			return true
		}
	}
	return false
}

// LooksLikePriorContentCanvasAsk reports puts of earlier chat content onto a canvas
// ("create a canvas with that information", "add that to a neural canvas").
func LooksLikePriorContentCanvasAsk(text string) bool {
	c := normalizeCanvasTypos(strings.ToLower(strings.TrimSpace(text)))
	if c == "" {
		return false
	}
	if !(strings.Contains(c, "canvas") || strings.Contains(c, "artifact")) {
		return false
	}
	cues := []string{
		"with that", "that information", "this information", "that summary",
		"those notes", "that content", "add that", "put that", "from that",
		"what you just", "what you wrote", "the above", "previous reply",
		"prior reply", "into a canvas", "into the canvas", "to a neural canvas",
		"to the neural canvas", "on a neural canvas", "on the neural canvas",
		"with this information", "with this",
	}
	for _, cue := range cues {
		if strings.Contains(c, cue) {
			return true
		}
	}
	return false
}

// shouldPromoteCanvasTextAsk promotes create/fill canvas asks that the classifier
// stamped as answer/ask_user without a blank_canvas reason code.
func shouldPromoteCanvasTextAsk(decision TurnDecision, text string) bool {
	switch decision.Action {
	case ActionDebug, ActionArtifact, ActionImage, ActionMusic:
		return false
	case ActionEdit:
		// Typo "canvans" often stamps edit; promote pure canvas creates, but keep
		// mixed "fix then canvas" turns as workspace edit.
		return LooksLikeCanvasCreateOrFillAsk(text) && !looksLikeMixedEditThenCanvasAsk(text)
	}
	return LooksLikeCanvasCreateOrFillAsk(text)
}

func looksLikeMixedEditThenCanvasAsk(text string) bool {
	c := normalizeCanvasTypos(strings.ToLower(strings.TrimSpace(text)))
	if c == "" || !strings.Contains(c, "canvas") {
		return false
	}
	return strings.Contains(c, "fix") || strings.Contains(c, "typo") ||
		strings.Contains(c, "implement") || strings.Contains(c, "after you") ||
		strings.Contains(c, "then create") || strings.Contains(c, ".go") ||
		strings.Contains(c, ".ts") || strings.Contains(c, ".rs")
}

func containsRetrievalTarget(values []RetrievalTarget, target RetrievalTarget) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func filterRetrievalTargets(values []RetrievalTarget, drop ...RetrievalTarget) []RetrievalTarget {
	if len(values) == 0 {
		return nil
	}
	deny := map[RetrievalTarget]bool{}
	for _, target := range drop {
		deny[target] = true
	}
	out := make([]RetrievalTarget, 0, len(values))
	for _, value := range values {
		if deny[value] {
			continue
		}
		out = append(out, value)
	}
	return out
}

// LooksLikeOpenCanvasFillAsk reports add/put/fill instructions that should revise
// an open Neural Canvas when open_artifact_* features are set. Used only as a
// structural companion to open-canvas policy — not as primary turn routing.
func LooksLikeOpenCanvasFillAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" || LooksLikeCanvasStatusQuestion(c) || LooksLikeCanvasTitleQuestion(c) {
		return false
	}
	hasVerb := strings.Contains(c, "add ") || strings.Contains(c, "put ") ||
		strings.Contains(c, "fill") || strings.Contains(c, "include ") ||
		strings.Contains(c, "insert ") || strings.Contains(c, "embed ") ||
		strings.Contains(c, "write ") || strings.Contains(c, "update ") ||
		strings.Contains(c, "rename ") || strings.Contains(c, "call it ") ||
		strings.Contains(c, "title it ") || strings.Contains(c, "name it ") ||
		LooksLikeCanvasRenameAsk(c)
	if !hasVerb {
		return false
	}
	return strings.Contains(c, "canvas") || strings.Contains(c, "the page") ||
		strings.Contains(c, "the document") || strings.Contains(c, " in there") ||
		strings.Contains(c, "on there") || strings.HasSuffix(c, " there") ||
		strings.Contains(c, "to it") || strings.Contains(c, "in it") ||
		LooksLikeCanvasRenameAsk(c) || LooksLikeOpenCanvasReviseAsk(c)
}

var openCanvasOrdinalItemRE = regexp.MustCompile(`(?i)\b(\d+(st|nd|rd|th)|first|second|third|fourth|fifth)\s+(list\s+)?item\b`)

// LooksLikeOpenCanvasReviseAsk reports content edits for an already-open canvas
// that omit the word "canvas" — e.g. "the 3rd item is Arrive in Florida",
// "ok add a 3rd list item, arrive in florida".
func LooksLikeOpenCanvasReviseAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" || LooksLikeCanvasStatusQuestion(c) || LooksLikeCanvasTitleQuestion(c) {
		return false
	}
	// Workspace file edits must stay edit, not canvas revise.
	if strings.Contains(c, ".go") || strings.Contains(c, ".ts") || strings.Contains(c, ".tsx") ||
		strings.Contains(c, ".js") || strings.Contains(c, ".rs") || strings.Contains(c, ".py") {
		return false
	}
	if openCanvasOrdinalItemRE.MatchString(c) {
		return true
	}
	listObject := strings.Contains(c, "item") || strings.Contains(c, "bullet") ||
		strings.Contains(c, "step") || strings.Contains(c, "point") ||
		strings.Contains(c, "list") || strings.Contains(c, "section")
	listVerb := strings.Contains(c, "add ") || strings.Contains(c, "insert ") ||
		strings.Contains(c, "include ") || strings.Contains(c, "append ") ||
		strings.Contains(c, "change ") || strings.Contains(c, "update ") ||
		strings.Contains(c, "replace ") || strings.Contains(c, "fix ") ||
		strings.Contains(c, "remove ") || strings.Contains(c, "delete ")
	if listObject && listVerb {
		return true
	}
	// Declarative corrections: "the 3rd item is X", "item 3 should be X"
	if strings.Contains(c, "item") && (strings.Contains(c, " is ") ||
		strings.Contains(c, " should be ") || strings.Contains(c, " to ")) {
		return true
	}
	return false
}

// LooksLikeCanvasStatusQuestion reports meta questions about whether the open
// canvas was updated — keep these as answer, not artifact creates.
func LooksLikeCanvasStatusQuestion(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	if LooksLikeCanvasTitleQuestion(c) {
		return true
	}
	if strings.Contains(c, "did you") || strings.Contains(c, "have you") ||
		strings.Contains(c, "was the") || strings.Contains(c, "did the") {
		return strings.Contains(c, "canvas") || strings.Contains(c, "update") ||
			strings.Contains(c, "the info") || strings.Contains(c, "the page")
	}
	if strings.Contains(c, "what's on") || strings.Contains(c, "what is on") ||
		strings.Contains(c, "show me the canvas") {
		return true
	}
	return false
}

// LooksLikeCanvasTitleQuestion reports asks about why/how the open canvas was named.
func LooksLikeCanvasTitleQuestion(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" {
		return false
	}
	if strings.Contains(c, "why did you name") || strings.Contains(c, "why did you call") ||
		strings.Contains(c, "why is it called") || strings.Contains(c, "why is it named") ||
		strings.Contains(c, "why name it") || strings.Contains(c, "why titled") ||
		strings.Contains(c, "why the title") || strings.Contains(c, "why that title") ||
		strings.Contains(c, "why did you title") || strings.Contains(c, "why named it") ||
		strings.Contains(c, "why call it") {
		return true
	}
	if strings.Contains(c, "why") &&
		(strings.Contains(c, "named it") || strings.Contains(c, "name it") ||
			strings.Contains(c, "call it") || strings.Contains(c, "called it") ||
			strings.Contains(c, "title")) {
		return true
	}
	return false
}

// LooksLikeCanvasRenameAsk reports an instruction to rename the open canvas.
func LooksLikeCanvasRenameAsk(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" || LooksLikeCanvasTitleQuestion(c) {
		return false
	}
	if strings.Contains(c, "rename") || strings.Contains(c, "retitle") ||
		strings.Contains(c, "call it ") || strings.Contains(c, "title it ") ||
		strings.Contains(c, "name it ") || strings.Contains(c, "name this ") ||
		strings.Contains(c, "call this ") || strings.Contains(c, "change the title") ||
		strings.Contains(c, "change its title") || strings.Contains(c, "set the title") ||
		strings.Contains(c, "title should be") {
		return true
	}
	// "The title of the document should be …" / "title of the page is …"
	if strings.Contains(c, "the title of") &&
		(strings.Contains(c, "should be") || strings.Contains(c, " to ")) {
		return true
	}
	return false
}

// LooksLikeAdvisoryImplementationQuestion is deprecated. Advisory turns are
// classified by the semantic router (reason_codes including advisory_question).
//
// Deprecated: always returns false. Do not add new call sites.
func LooksLikeAdvisoryImplementationQuestion(text string) bool {
	_ = text
	return false
}

// LooksLikePresenceCheck is deprecated. Presence/casual turns are classified by
// the semantic router (interaction=casual, action=answer).
//
// Deprecated: always returns false. Do not add new call sites.
func LooksLikePresenceCheck(text string) bool {
	_ = text
	return false
}

func mutationForAction(action Action) Mutation {
	switch action {
	case ActionEdit, ActionContinue:
		return MutationWorkspace
	case ActionArtifact, ActionImage, ActionMusic:
		return MutationExternal
	default:
		return MutationNone
	}
}

func isMutatingAction(action Action) bool {
	return mutationForAction(action) != MutationNone
}

func safeFallback(features TurnFeatures, reason string) TurnDecision {
	recipient := strings.TrimSpace(features.ExplicitRecipient)
	if recipient == "" {
		recipient = "assistant"
	}
	semantic := SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0,
		ReasonCodes:       []string{reason},
		RecipientType:     recipient,
		Retrieval:         []RetrievalTarget{RetrievalMemory},
	}
	decision := ResolvePolicy(features, semantic, SourceSafeFallback)
	decision.Confidence = 0
	decision.AbstentionReason = reason
	if reason != "" && !containsString(decision.ReasonCodes, reason) {
		decision.ReasonCodes = append([]string{reason}, decision.ReasonCodes...)
	}
	return decision
}

func classifierFallback(features TurnFeatures, reason string) TurnDecision {
	mode := strings.ToLower(strings.TrimSpace(features.ComposerMode))
	if features.ExplicitAction == "" && mode != "plan" && mode != "export" {
		return safeFallback(features, reason)
	}
	semantic := SemanticIntent{
		SchemaVersion: SchemaVersion, Interaction: InteractionTask,
		RequestedAction: ActionAnswer, MutationRequested: MutationNone,
		Confidence: 1, ReasonCodes: []string{reason},
		RecipientType: strings.TrimSpace(features.ExplicitRecipient),
	}
	applyExplicitSemanticDirectives(features, &semantic)
	decision := ResolvePolicy(features, semantic, SourceStructural)
	decision.AbstentionReason = reason
	return decision
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
