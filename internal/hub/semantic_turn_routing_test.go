package hub

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type semanticRouterFunc func(context.Context, intent.TurnFeatures) intent.TurnDecision

func (fn semanticRouterFunc) Resolve(ctx context.Context, features intent.TurnFeatures) intent.TurnDecision {
	return fn(ctx, features)
}

func TestResolveSemanticTurnStampsCanonicalImplementation(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic", "", "", protocol.ChannelTypePublic, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(_ context.Context, features intent.TurnFeatures) intent.TurnDecision {
		if !features.HasWorkspace || features.ComposerMode != "agent" {
			t.Fatalf("features=%+v", features)
		}
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
			RequestedAction: intent.ActionEdit, Action: intent.ActionEdit,
			RecipientType: "frontend", Retrieval: []intent.RetrievalTarget{intent.RetrievalCodebase},
			Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "semantic", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "Repair the startup failure.")
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode: "agent",
		"workspace_context":           map[string]interface{}{"workspace_path": "/tmp/project"},
	}

	h.resolveSemanticTurn(context.Background(), msg)
	decision, ok := protocol.ExtractTurnDecision(msg)
	if !ok || decision.Action != intent.ActionEdit {
		t.Fatalf("decision=%+v ok=%v", decision, ok)
	}
	if !msg.ImplementationSession() || msg.IdeRouteAgentType() != "frontend" {
		t.Fatalf("metadata=%v", msg.Metadata)
	}
}

func TestResolveSemanticTurnPreservesExplicitMentionRouting(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-mention", "", "", protocol.ChannelTypePublic, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
			RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
			RecipientType: "frontend", Mutation: intent.MutationNone,
			Confidence: 1, Source: intent.SourceLocalModel,
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "semantic-mention", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "@BackendEngineer explain this")
	msg.Mentions = []string{"backend-id"}
	msg.Metadata = map[string]interface{}{protocol.IdeMetaRouteAgentType: "backend"}
	h.resolveSemanticTurn(context.Background(), msg)
	if msg.IdeRouteAgentType() != "backend" {
		t.Fatalf("explicit route overwritten: %v", msg.Metadata)
	}
}

func TestResolveSemanticTurnRejectsClientAuthoredDecision(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-spoof", "", "", protocol.ChannelTypePublic, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
			RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
			RecipientType: "assistant", Mutation: intent.MutationNone,
			Confidence: 1, Source: intent.SourceLocalModel,
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "semantic-spoof", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "Explain the project.")
	msg.Metadata = map[string]interface{}{}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionEdit, Action: intent.ActionEdit,
		RecipientType: "frontend", Mutation: intent.MutationWorkspace,
		Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	h.resolveSemanticTurn(context.Background(), msg)
	decision, ok := protocol.ExtractTurnDecision(msg)
	if !ok || decision.Action != intent.ActionAnswer || decision.Mutation != intent.MutationNone {
		t.Fatalf("client decision survived: %+v", decision)
	}
}

func TestResolveSemanticTurnPhraseMismatchCannotOverrideAction(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-phrase", "", "", protocol.ChannelTypePublic, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
			RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
			RecipientType: "assistant", Mutation: intent.MutationNone,
			Confidence: 1, Source: intent.SourceLocalModel,
		}
	}))
	// Canvas phrases would trigger artifact heuristics without a stamped decision.
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "semantic-phrase", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "please create a canvas of the architecture")
	msg.Metadata = map[string]interface{}{protocol.TurnMetaComposerMode: "agent"}
	h.resolveSemanticTurn(context.Background(), msg)
	decision, ok := protocol.ExtractTurnDecision(msg)
	if !ok || decision.Action != intent.ActionAnswer || decision.Mutation != intent.MutationNone {
		t.Fatalf("decision=%+v ok=%v", decision, ok)
	}
	if msg.ImplementationSession() {
		t.Fatal("ActionAnswer must not stamp implementation_session")
	}
	if msg.IdeRouteAgentType() != "assistant" {
		t.Fatalf("recipient route=%q, want assistant", msg.IdeRouteAgentType())
	}
}
