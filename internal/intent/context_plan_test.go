package intent

import "testing"

func TestDeriveContextPlanWorkspaceDocuments(t *testing.T) {
	features := TurnFeatures{HasWorkspace: true}
	decision := TurnDecision{
		SchemaVersion:   SchemaVersion,
		Interaction:     InteractionTask,
		RequestedAction: ActionInspect,
		Action:          ActionInspect,
		Retrieval:       []RetrievalTarget{RetrievalCodebase},
		Mutation:        MutationNone,
		Confidence:      0.9,
		Source:          SourceLocalModel,
		ReasonCodes:     []string{"project_overview"},
	}
	plan, usedFallback := DeriveContextPlan(features, decision, SemanticIntent{})
	if !usedFallback {
		t.Fatal("expected fallback when classifier omitted context fields")
	}
	if plan.Tier != ContextTierOutline {
		t.Fatalf("tier=%s want outline", plan.Tier)
	}
	if plan.Subject != ContextSubjectWorkspaceDocuments {
		t.Fatalf("subject=%s want workspace_documents", plan.Subject)
	}
	if plan.ReviewMode != ReviewModeWorkspace {
		t.Fatalf("review_mode=%s want workspace", plan.ReviewMode)
	}
	req := ContextRequestFromPlan(plan, features)
	if !req.IncludeDocumentBodies || !req.IncludeFileTree {
		t.Fatalf("context request missing document bodies/tree: %+v", req)
	}
}

func TestDeriveContextPlanClassifierProposal(t *testing.T) {
	features := TurnFeatures{HasWorkspace: true}
	decision := TurnDecision{
		SchemaVersion:   SchemaVersion,
		Interaction:     InteractionTask,
		RequestedAction: ActionInspect,
		Action:          ActionInspect,
		Retrieval:       []RetrievalTarget{RetrievalCodebase},
		Mutation:        MutationNone,
		Confidence:      0.95,
		Source:          SourceLocalModel,
	}
	semantic := SemanticIntent{
		ContextTier: ContextTierFocus,
		Subject:     ContextSubjectActiveDocument,
		ReviewMode:  ReviewModeDocument,
	}
	plan, usedFallback := DeriveContextPlan(features, decision, semantic)
	if usedFallback {
		t.Fatal("did not expect fallback when classifier provided full plan")
	}
	if plan.Subject != ContextSubjectActiveDocument || plan.ReviewMode != ReviewModeDocument {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestEnsureContextPlan(t *testing.T) {
	d := TurnDecision{
		SchemaVersion:   SchemaVersion,
		Interaction:     InteractionQuestion,
		RequestedAction: ActionAnswer,
		Action:          ActionAnswer,
		Mutation:        MutationNone,
		Confidence:      1,
		Source:          SourceStructural,
	}
	EnsureContextPlan(&d, TurnFeatures{})
	if err := d.ContextPlan.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveContextPlanActionPlanWithCodebase(t *testing.T) {
	features := TurnFeatures{HasWorkspace: true}
	decision := TurnDecision{
		SchemaVersion:   SchemaVersion,
		Interaction:     InteractionTask,
		RequestedAction: ActionPlan,
		Action:          ActionPlan,
		Retrieval:       []RetrievalTarget{RetrievalCodebase},
		Mutation:        MutationNone,
		Confidence:      1,
		Source:          SourceStructural,
	}
	plan, _ := DeriveContextPlan(features, decision, SemanticIntent{})
	if plan.Tier != ContextTierOutline {
		t.Fatalf("tier=%s want outline", plan.Tier)
	}
	if plan.Subject != ContextSubjectCodebase {
		t.Fatalf("subject=%s want codebase", plan.Subject)
	}
	foundTree := false
	for _, cap := range plan.RequestedCapabilities {
		if cap == "file_tree" {
			foundTree = true
		}
	}
	if !foundTree {
		t.Fatalf("expected file_tree capability, got %v", plan.RequestedCapabilities)
	}
}
