package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func TestSemanticDebugDecisionDrivesGoalAndKnowledgeWithoutPhrases(t *testing.T) {
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-user-frontend",
		protocol.AgentInfo{ID: "user", Name: "Camron", Type: "human"},
		"The process exits before the interface appears. Diagnose and repair the project.",
	)
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode:         "agent",
		protocol.IdeMetaImplementationSession: true,
		"workspace_context":                   map[string]interface{}{"workspace_path": "/tmp/project"},
	}
	decision := intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionDebug, Action: intent.ActionDebug,
		RecipientType: "frontend", Retrieval: []intent.RetrievalTarget{intent.RetrievalCodebase},
		Mutation: intent.MutationNone, Confidence: 0.92, Source: intent.SourceLocalModel,
	}
	if err := protocol.StampTurnDecision(msg, decision); err != nil {
		t.Fatal(err)
	}

	goal := deriveTurnGoal(a, msg, IntentSubstantive)
	if goal.Action != ActionDebug || !goal.ImplementationSession || goal.Intent != IntentTask {
		t.Fatalf("goal=%+v", goal)
	}
	plan := a.effectiveKnowledgePlan(msg, goal.Intent)
	if !plan.Has(routing.RouteCodebase) || plan.Reason != "semantic_decision" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestSemanticArtifactDecisionCannotEnterImplementation(t *testing.T) {
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-user-frontend",
		protocol.AgentInfo{ID: "user", Name: "Camron", Type: "human"},
		"Present the architecture as a durable visual beside this conversation.",
	)
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode:         "agent",
		protocol.IdeMetaImplementationSession: true,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 0.95, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact || goal.ImplementationSession {
		t.Fatalf("goal=%+v", goal)
	}
}

func TestSemanticAskModePolicyPreventsFileChanges(t *testing.T) {
	decision := intent.ResolvePolicy(intent.TurnFeatures{
		ComposerMode: "ask", HasWorkspace: true,
		CanProposeFiles: true, CanRunImplementation: true,
	}, intent.SemanticIntent{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionEdit, MutationRequested: intent.MutationWorkspace,
		Confidence: 0.99,
	}, intent.SourceLocalModel)
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "user", Type: "human"}, "Make changes")
	msg.Metadata = map[string]interface{}{protocol.TurnMetaComposerMode: "ask"}
	if err := protocol.StampTurnDecision(msg, decision); err != nil {
		t.Fatal(err)
	}
	goal := deriveTurnGoal(nil, msg, IntentTask)
	if goal.Action != ActionAnswer || goal.Mutation != MutationNone || goal.ImplementationSession {
		t.Fatalf("goal=%+v", goal)
	}
}
