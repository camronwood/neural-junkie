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
