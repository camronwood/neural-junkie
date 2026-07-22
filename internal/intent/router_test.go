package intent

import (
	"context"
	"errors"
	"testing"
)

type testClassifier struct {
	intent SemanticIntent
	err    error
	calls  int
}

func (c *testClassifier) Classify(context.Context, TurnFeatures) (SemanticIntent, error) {
	c.calls++
	return c.intent, c.err
}

func (c *testClassifier) Model() string { return "local-test" }

func TestRouterPlanDirectiveSurvivesClassifierFailure(t *testing.T) {
	classifier := &testClassifier{err: errors.New("unavailable")}
	decision := NewRouter(classifier, 0.6).Resolve(context.Background(), TurnFeatures{
		Text:         "What should we change?",
		ComposerMode: "plan",
		HasWorkspace: true,
	})
	if classifier.calls != 1 {
		t.Fatalf("classifier calls=%d, want 1", classifier.calls)
	}
	if decision.Action != ActionPlan || decision.Mutation != MutationNone || decision.Source != SourceStructural {
		t.Fatalf("decision=%+v, want structural plan", decision)
	}
}

func TestRouterUsesClassifierForNaturalLanguage(t *testing.T) {
	classifier := &testClassifier{intent: SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionDebug,
		Retrieval:         []RetrievalTarget{RetrievalCodebase},
		MutationRequested: MutationNone,
		Confidence:        0.91,
		ReasonCodes:       []string{"runtime_failure"},
	}}
	decision := NewRouter(classifier, 0.6).Resolve(context.Background(), TurnFeatures{
		Text:                 "The program fails during startup; diagnose it.",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	})
	if classifier.calls != 1 {
		t.Fatalf("classifier calls=%d, want 1", classifier.calls)
	}
	if decision.Action != ActionDebug || decision.Source != SourceLocalModel || decision.ClassifierModel != "local-test" {
		t.Fatalf("decision=%+v, want local debug", decision)
	}
}

func TestPolicyNeverLetsAskModeMutate(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		ComposerMode:         "ask",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.99,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, ask mode authorized mutation", decision)
	}
}

func TestPolicyRequiresWorkspaceForEdit(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		ComposerMode:         "agent",
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionAskUser || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, want workspace clarification", decision)
	}
}

func TestPolicyAddsCodebaseRetrievalForWorkspaceEdits(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		Domain:            "frontend",
		RecipientType:     "frontend",
		MutationRequested: MutationWorkspace,
		Confidence:        0.95,
	}, SourceLocalModel)
	if decision.Action != ActionEdit || decision.Mutation != MutationWorkspace {
		t.Fatalf("decision=%+v, want workspace edit", decision)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
		t.Fatalf("retrieval=%v, want codebase", decision.Retrieval)
	}
}

func TestLowConfidenceAbstainsSafely(t *testing.T) {
	classifier := &testClassifier{intent: SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.2,
	}}
	decision := NewRouter(classifier, 0.7).Resolve(context.Background(), TurnFeatures{Text: "ambiguous"})
	if decision.Action != ActionAnswer || decision.Mutation != MutationNone || decision.Source != SourceSafeFallback {
		t.Fatalf("decision=%+v, want safe fallback", decision)
	}
}

func TestSemanticIntentRejectsInvalidEnumAndConfidence(t *testing.T) {
	base := SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.5,
	}
	invalid := base
	invalid.RequestedAction = Action("destroy")
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid action accepted")
	}
	invalid = base
	invalid.Confidence = 1.1
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid confidence accepted")
	}
}

func TestSemanticConsistencyUsesTypedContinuationState(t *testing.T) {
	semantic := SemanticIntent{
		SchemaVersion: SchemaVersion, Interaction: InteractionContinuation,
		RequestedAction: ActionRun, MutationRequested: MutationNone,
		Confidence: 0.95,
	}
	features := TurnFeatures{PendingActionID: "goal-1", PendingAction: ActionEdit}
	normalizeSemanticConsistency(features, &semantic)
	if semantic.RequestedAction != ActionContinue || semantic.ContinuationTarget != "goal-1" ||
		semantic.MutationRequested != MutationWorkspace {
		t.Fatalf("semantic=%+v", semantic)
	}
}

func TestSemanticConsistencyDerivesArtifactMutationAndRecipient(t *testing.T) {
	artifact := SemanticIntent{
		SchemaVersion: SchemaVersion, Interaction: InteractionTask,
		RequestedAction: ActionArtifact, MutationRequested: MutationNone, Confidence: 0.9,
	}
	normalizeSemanticConsistency(TurnFeatures{}, &artifact)
	if artifact.MutationRequested != MutationExternal {
		t.Fatalf("artifact=%+v", artifact)
	}
	debug := SemanticIntent{
		SchemaVersion: SchemaVersion, Interaction: InteractionTask,
		RequestedAction: ActionDebug, Domain: "frontend", RecipientType: "assistant",
		MutationRequested: MutationWorkspace, Confidence: 0.9,
	}
	normalizeSemanticConsistency(TurnFeatures{}, &debug)
	if debug.RecipientType != "frontend" {
		t.Fatalf("debug=%+v", debug)
	}
}

func TestPolicyUpgradesLookCueToInspect(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "ok go ahead and have a look",
		ComposerMode: "agent",
		HasWorkspace: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.9,
	}, SourceLocalModel)
	if decision.Action != ActionInspect {
		t.Fatalf("action=%s, want inspect", decision.Action)
	}
	if !containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
		t.Fatalf("retrieval=%v, want codebase", decision.Retrieval)
	}
	found := false
	for _, o := range decision.PolicyOverrides {
		if o == "workspace_inspect_cue" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("overrides=%v, want workspace_inspect_cue", decision.PolicyOverrides)
	}
}

func TestPolicyUpgradesGitHistoryCueToInspect(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "check the current state against git history to see if we can restore something that broke",
		HasWorkspace: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionAnswer,
		MutationRequested: MutationNone,
		Confidence:        0.88,
	}, SourceLocalModel)
	if decision.Action != ActionInspect {
		t.Fatalf("action=%s, want inspect", decision.Action)
	}
}

func TestLooksLikeWorkspaceInspectRequest(t *testing.T) {
	if !looksLikeWorkspaceInspectRequest("ok go ahead and have a look") {
		t.Fatal("expected inspect cue")
	}
	if !looksLikeWorkspaceInspectRequest("yeah please check git and see if we can find it") {
		t.Fatal("expected check git cue")
	}
	if !looksLikeWorkspaceInspectRequest("can we use git to find what the working config was?") {
		t.Fatal("expected use git cue")
	}
	if !LooksLikeGitInspectRequest("yeah please check git and see if we can find it") {
		t.Fatal("expected LooksLikeGitInspectRequest")
	}
	if LooksLikeGitInspectRequest("ok go ahead and have a look") {
		t.Fatal("generic look cue should not force git tools")
	}
	if looksLikeWorkspaceInspectRequest("thanks, that helps") {
		t.Fatal("did not expect inspect cue")
	}
}

func TestSafeFallbackStillUpgradesGitInspectCue(t *testing.T) {
	decision := safeFallback(TurnFeatures{
		Text:              "yeah please check git and see if we can find it",
		HasWorkspace:      true,
		ExplicitRecipient: "frontend",
		ComposerMode:      "agent",
	}, "classifier_error")
	if decision.Action != ActionInspect {
		t.Fatalf("action=%s, want inspect after classifier fallback", decision.Action)
	}
	foundCodebase := false
	for _, r := range decision.Retrieval {
		if r == RetrievalCodebase {
			foundCodebase = true
			break
		}
	}
	if !foundCodebase {
		t.Fatalf("retrieval=%v, want codebase", decision.Retrieval)
	}
	if decision.Source != SourceSafeFallback {
		t.Fatalf("source=%s, want safe_fallback", decision.Source)
	}
}

func TestPolicyDemotesCreativeWriteToAnswer(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:                 "Can you write me an alternet ending to game of thrones?",
		ComposerMode:         "agent",
		HasWorkspace:         true,
		CanProposeFiles:      true,
		CanRunImplementation: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionTask,
		RequestedAction:   ActionEdit,
		MutationRequested: MutationWorkspace,
		Confidence:        0.95,
		Retrieval:         []RetrievalTarget{RetrievalCodebase},
	}, SourceLocalModel)
	if decision.Action != ActionAnswer || decision.Mutation != MutationNone {
		t.Fatalf("decision=%+v, want answer for creative write", decision)
	}
	if containsRetrievalTarget(decision.Retrieval, RetrievalCodebase) {
		t.Fatalf("retrieval=%v, creative answer should drop codebase", decision.Retrieval)
	}
}

func TestPolicyDemotesShortClarificationFromAskUser(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "what?",
		ComposerMode: "agent",
		HasWorkspace: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionContinuation,
		RequestedAction:   ActionAskUser,
		MutationRequested: MutationNone,
		Confidence:        0.8,
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer for short clarification", decision.Action)
	}
}

func TestPolicyPresenceCheckStripsPriorReference(t *testing.T) {
	decision := ResolvePolicy(TurnFeatures{
		Text:         "are you there?",
		ComposerMode: "agent",
		HasWorkspace: true,
	}, SemanticIntent{
		SchemaVersion:     SchemaVersion,
		Interaction:       InteractionQuestion,
		RequestedAction:   ActionInspect,
		MutationRequested: MutationNone,
		Confidence:        0.95,
		Retrieval:         []RetrievalTarget{RetrievalPriorReference},
	}, SourceLocalModel)
	if decision.Action != ActionAnswer {
		t.Fatalf("action=%s, want answer for presence check", decision.Action)
	}
	if containsRetrievalTarget(decision.Retrieval, RetrievalPriorReference) {
		t.Fatalf("retrieval=%v, presence check must drop prior_reference", decision.Retrieval)
	}
	found := false
	for _, o := range decision.PolicyOverrides {
		if o == "presence_check" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("overrides=%v, want presence_check", decision.PolicyOverrides)
	}
}
