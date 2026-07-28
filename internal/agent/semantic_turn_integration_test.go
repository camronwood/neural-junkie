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

func TestSemanticGitInspectRequiresCommandEvidence(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-user-frontend",
		protocol.AgentInfo{ID: "user", Name: "Camron", Type: "human"},
		"yeah please check git and see if we can find the working config",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionInspect, Action: intent.ActionInspect,
		Retrieval: []intent.RetrievalTarget{intent.RetrievalCodebase},
		Mutation:  intent.MutationNone, Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	goal := deriveTurnGoal(nil, msg, IntentSubstantive)
	if goal.Action != ActionInspect {
		t.Fatalf("action=%s, want inspect", goal.Action)
	}
	if !goal.RequiresActionEvidence() {
		t.Fatal("git inspect should require command evidence")
	}
	foundRun := false
	for _, capName := range goal.RequiredCapabilities {
		if capName == "run_command" {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Fatalf("capabilities=%v, want run_command", goal.RequiredCapabilities)
	}
	if len(goal.ExpectedEvidence) != 1 || goal.ExpectedEvidence[0] != EvidenceCommandRun {
		t.Fatalf("evidence=%v, want command_run", goal.ExpectedEvidence)
	}
}

func TestStampedAnswerDecisionIgnoresArtifactPhrases(t *testing.T) {
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-user-frontend",
		protocol.AgentInfo{ID: "user", Name: "Camron", Type: "human"},
		"please create a canvas of the architecture",
	)
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode:         "agent",
		protocol.IdeMetaImplementationSession: true,
		MetadataConversationMode:              ConversationModeCode,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Mutation: intent.MutationNone, Confidence: 0.95, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}

	if artifactToolsEnabledForMessage(msg) {
		t.Fatal("ActionAnswer must not enable artifact tools despite canvas phrases")
	}
	if userRequestsImplementationForMessage(a, msg) {
		t.Fatal("a stamped ActionAnswer must not itself report implementation intent")
	}
	// Stamp-first: an explicit IDE implementation_session flag is a structural signal
	// that is no longer second-guessed by scanning the message for canvas/artifact
	// phrasing (the old UserRequestsArtifact veto) — it still drives the session gate. // phrase-migration-shim
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("explicit implementation_session metadata must still be honored for ActionAnswer")
	}
	if EffectiveConversationMode(msg, protocol.ChannelTypeDM) != ConversationModeChat {
		t.Fatalf("conversation mode=%q, want chat for ActionAnswer", EffectiveConversationMode(msg, protocol.ChannelTypeDM))
	}
}

func TestStampedInspectDecisionIgnoresCodeReviewPhrases(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm",
		protocol.AgentInfo{ID: "user", Type: "human"},
		"code review this project please: /tmp/example-repo",
	)
	if !userRequestsCodeReview(msg.Content) {
		t.Fatal("fixture must phrase-match code review")
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		RecipientType: "assistant", Mutation: intent.MutationNone,
		Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if userRequestsCodeReviewForMessage(msg) {
		t.Fatal("stamped ActionAnswer must not treat review phrases as code-review intent")
	}
}

// TestStampedEditDecisionTrustedDespiteCanvasPhrasing documents the stamp-first
// replacement for the old neuralCanvasIsSecondaryToCodeChange phrase override: a
// classifier that stamps ActionEdit is trusted as-is, even when the message text
// happens to mention "Neural Canvas" / "Mermaid diagram". If the classifier means
// a durable artifact, it must stamp ActionArtifact directly — routing no longer
// re-derives that from message phrasing.
func TestStampedEditDecisionTrustedDespiteCanvasPhrasing(t *testing.T) {
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-user-assistant",
		protocol.AgentInfo{ID: "user", Type: "human"},
		"Create a Neural Canvas Mermaid diagram of this architecture",
	)
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode:         "agent",
		protocol.IdeMetaImplementationSession: true,
		"workspace_context":                   map[string]interface{}{"workspace_path": "/tmp/project"},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionEdit, Action: intent.ActionEdit,
		RecipientType: "assistant", Mutation: intent.MutationWorkspace,
		Confidence: 0.9, Source: intent.SourceLocalModel,
		Retrieval:  []intent.RetrievalTarget{intent.RetrievalCodebase},
	}); err != nil {
		t.Fatal(err)
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionEdit || !goal.ImplementationSession {
		t.Fatalf("goal=%+v, want stamped edit trusted despite canvas phrasing", goal)
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("stamped ActionEdit must run implementation session regardless of canvas phrasing")
	}
	if artifactToolsEnabledForMessage(msg) {
		t.Fatal("ActionEdit must not enable artifact tools despite canvas phrasing")
	}
	if !userRequestsImplementationForMessage(a, msg) {
		t.Fatal("stamped ActionEdit must report implementation intent")
	}
}

func TestStampedEditDecisionDrivesAssistantImplGateWithoutPhrases(t *testing.T) {
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-user-assistant",
		protocol.AgentInfo{ID: "user", Type: "human"},
		"please create a canvas after you fix the typo in the greeting",
	)
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode:         "agent",
		protocol.IdeMetaImplementationSession: true,
		"workspace_context":                   map[string]interface{}{"workspace_path": "/tmp/project"},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionEdit, Action: intent.ActionEdit,
		RecipientType: "assistant", Mutation: intent.MutationWorkspace,
		Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !assistantAllowsImplementationSession(a, msg) {
		t.Fatal("stamped ActionEdit must allow Assistant impl session without phrase match")
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("stamped ActionEdit must run implementation session")
	}
	if artifactToolsEnabledForMessage(msg) {
		t.Fatal("ActionEdit must not enable artifact tools despite canvas phrases")
	}
}

