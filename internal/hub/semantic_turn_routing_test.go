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

func TestResolveSemanticTurnPreservesExplicitIdeRouteWithoutMentions(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-ide-route", "", "", protocol.ChannelTypePublic, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
			RequestedAction: intent.ActionInspect, Action: intent.ActionInspect,
			RecipientType: "assistant", Mutation: intent.MutationNone,
			Confidence: 1, Source: intent.SourceLocalModel,
			ReasonCodes: []string{"runtime_failure"},
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "semantic-ide-route", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "the app won't boot — make start-all fails")
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode:         "agent",
		protocol.IdeMetaRouteAgentType:        "frontend",
		protocol.IdeMetaImplementationSession: true,
		"workspace_context":                   map[string]interface{}{"workspace_path": "/tmp/fixture"},
	}
	h.resolveSemanticTurn(context.Background(), msg)
	if msg.IdeRouteAgentType() != "frontend" {
		t.Fatalf("scenario ide_route_agent_type overwritten to %q", msg.IdeRouteAgentType())
	}
	if !msg.ImplementationSession() {
		t.Fatal("expected implementation_session preserved")
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

func TestResolveSemanticTurnSkipsSystemInfoJoin(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-join", "", "", protocol.ChannelTypePublic, "user")
	called := false
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		called = true
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionContinuation,
			RequestedAction: intent.ActionContinue, Action: intent.ActionContinue,
			Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeSystemInfo, "semantic-join", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "Camron has joined the chat")
	h.resolveSemanticTurn(context.Background(), msg)
	if called {
		t.Fatal("classifier must not run for system_info join announcements")
	}
	if _, ok := protocol.ExtractTurnDecision(msg); ok {
		t.Fatal("system_info must not receive a stamped turn decision")
	}
	if msg.ImplementationSession() {
		t.Fatal("system_info must not stamp implementation_session")
	}
}

func TestResolveSemanticTurnContinueWithoutWorkspaceMutationSkipsImplSession(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-continue", "", "", protocol.ChannelTypePublic, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionContinuation,
			RequestedAction: intent.ActionContinue, Action: intent.ActionContinue,
			RecipientType: "assistant", Mutation: intent.MutationNone,
			Confidence: 1, Source: intent.SourceLocalModel,
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeChat, "semantic-continue", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "ok continue")
	msg.Metadata = map[string]interface{}{protocol.TurnMetaComposerMode: "agent"}
	h.resolveSemanticTurn(context.Background(), msg)
	if msg.ImplementationSession() {
		t.Fatal("continue with mutation=none must not stamp implementation_session")
	}
}

func TestResolveSemanticTurnChatModeDoesNotStampImplSession(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-chat", "", "", protocol.ChannelTypeDM, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
			RequestedAction: intent.ActionEdit, Action: intent.ActionEdit,
			RecipientType: "frontend", Mutation: intent.MutationWorkspace,
			Confidence: 0.9, Source: intent.SourceLocalModel,
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "semantic-chat", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "Design a theme settings flow with ThemeSettings.")
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode: "agent",
		"conversation_mode":           "chat",
		"workspace_context":           map[string]interface{}{"workspace_path": "/tmp/project"},
	}
	h.resolveSemanticTurn(context.Background(), msg)
	if msg.ImplementationSession() {
		t.Fatal("conversation_mode=chat must not stamp implementation_session from semantic Edit")
	}
	decision, ok := protocol.ExtractTurnDecision(msg)
	if !ok || decision.Action != intent.ActionAnswer || decision.Mutation != intent.MutationNone {
		t.Fatalf("chat advisory decision=%+v", decision)
	}
	gov, ok := protocol.ExtractTurnGovernance(msg)
	if !ok || gov.CanRunImplSession || gov.RequiresWorkspace {
		t.Fatalf("chat advisory governance=%+v", gov)
	}
}

func TestResolveSemanticTurnAdvisoryQuestionDoesNotStampImplSession(t *testing.T) {
	h := NewHub()
	h.CreateChannelWithType("semantic-advisory", "", "", protocol.ChannelTypeDM, "user")
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
			RequestedAction: intent.ActionEdit, Action: intent.ActionEdit,
			RecipientType: "backend", Mutation: intent.MutationWorkspace,
			Confidence: 0.9, Source: intent.SourceLocalModel,
		}
	}))
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "semantic-advisory", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "go deeper on the approach — what would you implement first?")
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode: "agent",
		"conversation_mode":           "code",
		"workspace_context":           map[string]interface{}{"workspace_path": "/tmp/project"},
	}
	h.resolveSemanticTurn(context.Background(), msg)
	if msg.ImplementationSession() {
		t.Fatal("advisory question must not stamp implementation_session")
	}
	decision, ok := protocol.ExtractTurnDecision(msg)
	if !ok || decision.Action != intent.ActionAnswer || decision.Mutation != intent.MutationNone {
		t.Fatalf("advisory decision=%+v", decision)
	}
}
