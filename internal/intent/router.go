package intent

import (
	"context"
	"fmt"
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
	case ActionArtifact, ActionImage:
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
		return "code-review"
	case "frontend", "backend", "devops", "architecture", "database", "security", "biology", "rust", "cad":
		return domain
	default:
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
	case features.PendingActionID != "" && (features.ReplyTarget != "" || looksLikeContinuationAffirmation(features.Text)):
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

func looksLikeContinuationAffirmation(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" || len(text) > 80 {
		return false
	}
	phrases := []string{
		"proceed", "go ahead", "keep going", "continue", "do it", "yes", "approved",
		"sounds good", "that works", "please continue", "go for it",
	}
	for _, phrase := range phrases {
		if text == phrase || strings.HasPrefix(text, phrase+" ") || strings.HasPrefix(text, phrase+",") ||
			strings.HasPrefix(text, phrase+"!") || strings.HasPrefix(text, phrase+".") {
			return true
		}
	}
	return false
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

	// Small local classifiers often map "write me a story/ending" onto edit/run.
	// Demote clearly conversational/creative asks before workspace authorization
	// rewrites them into ask_user soft-fails.
	if looksLikeCreativeOrGeneralAnswerRequest(features.Text) {
		switch decision.Action {
		case ActionEdit, ActionRun, ActionContinue, ActionAskUser, ActionDebug, ActionInspect:
			decision.Action = ActionAnswer
			decision.RequestedAction = ActionAnswer
			decision.Mutation = MutationNone
			decision.Retrieval = filterRetrievalTargets(decision.Retrieval, RetrievalCodebase, RetrievalCodeGraph)
			if len(decision.Retrieval) == 0 {
				decision.Retrieval = []RetrievalTarget{RetrievalMemory}
			}
			decision.PolicyOverrides = append(decision.PolicyOverrides, "creative_or_general_answer")
		}
	}

	mode := strings.ToLower(strings.TrimSpace(features.ComposerMode))
	if mode == "ask" || mode == "plan" {
		if decision.Mutation != MutationNone || isMutatingAction(decision.Action) {
			decision.PolicyOverrides = append(decision.PolicyOverrides, "composer_mode_forbids_mutation")
		}
		decision.Mutation = MutationNone
		if mode == "plan" {
			decision.Action = ActionPlan
		} else if isMutatingAction(decision.Action) {
			decision.Action = ActionAnswer
		}
	}
	if decision.Mutation == MutationWorkspace {
		if !features.CanProposeFiles || !features.CanRunImplementation {
			if looksLikeCreativeOrGeneralAnswerRequest(features.Text) {
				decision.Action = ActionAnswer
				decision.Mutation = MutationNone
				decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_mutation_not_authorized_answer")
			} else {
				decision.Action = ActionAskUser
				decision.Mutation = MutationNone
				decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_mutation_not_authorized")
			}
		} else if !features.HasWorkspace {
			if looksLikeCreativeOrGeneralAnswerRequest(features.Text) {
				decision.Action = ActionAnswer
				decision.Mutation = MutationNone
				decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_required_answer")
			} else {
				decision.Action = ActionAskUser
				decision.Mutation = MutationNone
				decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_required")
			}
		}
	}
	if decision.Action == ActionContinue && decision.ContinuationTarget == "" {
		decision.ContinuationTarget = strings.TrimSpace(features.PendingActionID)
		if decision.ContinuationTarget == "" {
			decision.Action = ActionAskUser
			decision.Mutation = MutationNone
			decision.PolicyOverrides = append(decision.PolicyOverrides, "continuation_target_missing")
		}
	}
	// Workspace look/check/git cues must not stay chat-only (Answer + memory).
	// Upgrade to inspect so tooling lands in the prompt and codebase retrieval is required.
	if features.HasWorkspace && looksLikeWorkspaceInspectRequest(features.Text) {
		switch decision.Action {
		case ActionAnswer, ActionAskUser, ActionPlan, "":
			decision.Action = ActionInspect
			decision.RequestedAction = ActionInspect
			decision.Mutation = MutationNone
			if decision.Interaction == InteractionQuestion || decision.Interaction == InteractionCasual || decision.Interaction == "" {
				decision.Interaction = InteractionTask
			}
			decision.PolicyOverrides = append(decision.PolicyOverrides, "workspace_inspect_cue")
		}
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
	decision.PolicyOverrides = normalizeStrings(decision.PolicyOverrides)
	return decision
}

// looksLikeWorkspaceInspectRequest detects user asks to examine workspace/git state
// without necessarily using the word "inspect".
func looksLikeWorkspaceInspectRequest(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	cues := []string{
		"have a look",
		"take a look",
		"go look",
		"go ahead and look",
		"look at the",
		"look into",
		"restore something that broke",
		"see what broke",
		"see what's wrong",
		"see whats wrong",
		"investigate",
		"inspect the",
		"desktop app",
		"app itself is not",
		"app itself does not",
		"tauri",
		"window does not",
		"won't boot",
		"will not boot",
		"does not boot",
	}
	for _, cue := range cues {
		if strings.Contains(t, cue) {
			return true
		}
	}
	return LooksLikeGitInspectRequest(text)
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

// looksLikeCreativeOrGeneralAnswerRequest detects conversational/creative asks that
// should stay ActionAnswer even when a small classifier maps "write" onto edit/run.
func looksLikeCreativeOrGeneralAnswerRequest(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	if looksLikeEngineeringWorkRequest(t) {
		return false
	}
	creative := []string{
		"alternate ending", "alternative ending", "ending to", "fanfic", "fan fiction",
		"write me a story", "write a story", "write me an", "write an essay", "write a poem",
		"write me a poem", "write a song", "write lyrics", "game of thrones", "short story",
		"tell me a joke", "tell me a story", "blog post", "linkedin post", "cover letter",
	}
	for _, cue := range creative {
		if strings.Contains(t, cue) {
			return true
		}
	}
	// Ultra-short clarifications ("what?", "huh?", "why?") after a soft-fail must not
	// stay on run/ask_user/continue and trigger another canned failure.
	fields := strings.Fields(t)
	if len(fields) > 0 && len(fields) <= 3 {
		switch strings.Trim(fields[0], "?.!,") {
		case "what", "huh", "why", "how", "who", "where", "when", "ok", "okay", "thanks", "thank":
			return true
		}
	}
	return false
}

func looksLikeEngineeringWorkRequest(text string) bool {
	t := strings.ToLower(text)
	cues := []string{
		"src/", ".go", ".ts", ".tsx", ".js", ".py", ".rs", ".java",
		"implement ", "refactor ", "unit test", "test suite", "run the test",
		"run tests", "git status", "git log", "git diff", "pull request", " code ",
		"codebase", "compile", "linter", "eslint", "cargo ", "npm ", "make ",
		"deploy", "dockerfile", "kubernetes", "fix the bug", "stack trace",
		"function ", "class ", "module ", "package.json", "cargo.toml",
	}
	for _, cue := range cues {
		if strings.Contains(t, cue) {
			return true
		}
	}
	return false
}

func mutationForAction(action Action) Mutation {
	switch action {
	case ActionEdit, ActionContinue:
		return MutationWorkspace
	case ActionArtifact, ActionImage:
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
	// Still run policy so workspace inspect/git cues are not lost when the classifier
	// times out or fails — otherwise every failed classify stays chat-only memory.
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
