package hub

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubSemanticRouter struct {
	decision intent.TurnDecision
}

func (s stubSemanticRouter) Resolve(context.Context, intent.TurnFeatures) intent.TurnDecision {
	return s.decision
}

func TestPrepareTurnReturnsContextRequest(t *testing.T) {
	h := NewHub()
	h.SetSemanticTurnRouter(stubSemanticRouter{decision: intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionQuestion,
		RequestedAction: intent.ActionInspect,
		Action:          intent.ActionInspect,
		Retrieval:       []intent.RetrievalTarget{intent.RetrievalCodebase},
		Mutation:        intent.MutationNone,
		Confidence:      0.95,
		Source:          intent.SourceLocalModel,
		ReasonCodes:     []string{"project_overview"},
	}})
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{
		ID: "human-test", Name: "Test", Type: "human",
	}, "please reivew the documents in the workspace")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": "/tmp/ciso",
			"workspace_name": "ciso",
			"file_tree":      "internal/\ngilead-security/\n",
			"open_files":     []interface{}{},
		},
		"context_scope": "hint",
	}
	result, err := h.PrepareTurn(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	if result.PrepareToken == "" {
		t.Fatal("missing prepare token")
	}
	if result.Decision.ContextPlan.Subject != intent.ContextSubjectWorkspaceDocuments {
		t.Fatalf("subject=%s want workspace_documents", result.Decision.ContextPlan.Subject)
	}
	if !result.ContextRequest.IncludeDocumentBodies {
		t.Fatalf("expected document bodies in context_request: %+v", result.ContextRequest)
	}
	pt, ok := h.PeekPreparedTurn(result.PrepareToken)
	if !ok || pt == nil {
		t.Fatal("prepared turn missing")
	}
	consumed, ok := h.ConsumePreparedTurn(result.PrepareToken)
	if !ok || consumed == nil {
		t.Fatal("consume failed")
	}
	if _, ok := h.PeekPreparedTurn(result.PrepareToken); ok {
		t.Fatal("token should be consumed")
	}
}
