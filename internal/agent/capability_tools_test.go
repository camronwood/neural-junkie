package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/mcp"
	biologymcp "github.com/camronwood/neural-junkie/internal/mcp/biology"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCapabilityToolsLazyActivationAndTurnIsolation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	config.InstallTestPack(t, cfg, config.PackLifeSciences)
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	config.SetAppConfig(cfg)
	mcp.SetAppConfig(cfg)
	t.Cleanup(func() {
		config.SetAppConfig(nil)
		mcp.SetAppConfig(nil)
	})

	server, err := biologymcp.NewBiologyMCP()
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Info:      protocol.AgentInfo{ID: "bio-1", Name: "BiologyExpert", Type: protocol.AgentTypeBiology},
		MCPServer: server,
	}
	unrelated := &protocol.Message{ID: "m1", Content: "hello there"}
	if toolNamesInclude(a.agentToolDefinitions(unrelated), "analyze_sequence") {
		t.Fatal("domain schemas should not be loaded for an unrelated turn")
	}
	if _, err := a.executeRequestlessActivationForTest(context.Background(), unrelated, "biology-api"); err != nil {
		t.Fatal(err)
	}
	if !toolNamesInclude(a.agentToolDefinitions(unrelated), "analyze_sequence") {
		t.Fatal("expected biology tool after activation")
	}
	nextTurn := &protocol.Message{ID: "m2", Content: "hello again"}
	if toolNamesInclude(a.agentToolDefinitions(nextTurn), "analyze_sequence") {
		t.Fatal("activation must not leak into another turn")
	}
}

func TestShouldOfferCapabilityTools_presencePing(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		"are you here and ready to help?",
		"are you here and ready to help me?",
		"are you here and ready to hlep?",
		"you still there?",
	} {
		msg := &protocol.Message{Content: content}
		if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionCasual,
			RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
			Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
		}); err != nil {
			t.Fatal(err)
		}
		if shouldOfferCapabilityTools(msg) {
			t.Fatalf("expected capability tools suppressed for %q", content)
		}
		if !isConversationalOnlyTurn(msg) {
			t.Fatalf("expected conversational-only for %q", content)
		}
	}
	if !shouldOfferCapabilityTools(&protocol.Message{Content: "activate biology-api and analyze this FASTA"}) {
		t.Fatal("expected capability tools for a real capability task")
	}
}

func TestShouldOfferCapabilityHandoff_readinessPingEvenIfMisclassifiedAsTask(t *testing.T) {
	t.Parallel()
	// Regression: "Hello, are you ready?" opened a BackendEngineer handoff with an
	// invented Express task. The user text is chat-shaped (no work signals); handoff
	// must stay off even when the stamp wrongly says task/edit.
	for _, content := range []string{
		"Hello, are you ready?",
		"are you ready?",
		"you ready?",
	} {
		msg := &protocol.Message{Content: content}
		if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
			SchemaVersion:   intent.SchemaVersion,
			Interaction:     intent.InteractionTask,
			RequestedAction: intent.ActionEdit,
			Action:          intent.ActionEdit,
			Mutation:        intent.MutationWorkspace,
			Confidence:      1,
			Source:          intent.SourceLocalModel,
		}); err != nil {
			t.Fatal(err)
		}
		if !shouldOfferCapabilityTools(msg) {
			t.Fatalf("activate may still be offered under a bad stamp for %q; handoff is the hard gate", content)
		}
		if shouldOfferCapabilityHandoff(msg) {
			t.Fatalf("expected capability handoff suppressed for readiness ping %q", content)
		}
	}
}

func TestShouldOfferCapabilityHandoff_unstampedWithoutWorkSignals(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "Hello, are you ready?"}
	if shouldOfferCapabilityTools(msg) {
		t.Fatal("unstamped readiness ping must not offer capability tools")
	}
	if shouldOfferCapabilityHandoff(msg) {
		t.Fatal("unstamped readiness ping must not offer capability handoff")
	}
}

func TestShouldOfferCapabilityHandoff_allowsConcreteWork(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "debug the failing auth middleware in auth.go"}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionTask,
		RequestedAction: intent.ActionDebug,
		Action:          intent.ActionDebug,
		Mutation:        intent.MutationWorkspace,
		Confidence:      1,
		Source:          intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !shouldOfferCapabilityHandoff(msg) {
		t.Fatal("expected handoff tools for a concrete debug ask")
	}
}

func TestExecuteRequestCapabilityHelpTool_refusesReadinessPingDespiteConcreteTask(t *testing.T) {
	t.Parallel()
	ctx := withCapabilityHandoffTurnState(context.Background())
	a := &Agent{Info: protocol.AgentInfo{ID: "a1", Name: "Code-Expert-VibeCoding", Type: protocol.AgentTypeExpert}}
	msg := &protocol.Message{
		ID:      "m1",
		Channel: "dm-camron-code-expert-vibecoding",
		From:    protocol.AgentInfo{Name: "Camron"},
		Content: "Hello, are you ready?",
	}
	// Invented concrete task like the live failure — must still refuse based on user turn.
	out, err := a.executeRequestCapabilityHelpTool(ctx, msg, []byte(
		`{"capability_id":"software-development/sd-mcp-sidecar","task":"Initialize a new project with package.json and src/index.js, then create a basic Express server that logs Hello Vibe to the console on startup."}`,
	))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "not appropriate") && !strings.Contains(strings.ToLower(out), "locally") {
		t.Fatalf("expected user-turn refusal, got %q", out)
	}
	if st := capabilityHandoffTurnStateFromContext(ctx); st == nil || st.count != 0 {
		t.Fatal("readiness refusal must not consume the per-turn budget")
	}
}

func TestAgentToolDefinitions_presencePingOmitsWorkspaceTools(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		// Non-nil MCPServer interface would need a real server; use hasWorkspaceTools path via file edits.
	}
	// Force workspace tools path with a stub by setting WorkspacePath and using definitions that check hasWorkspaceTools.
	// Instead assert conversational gate directly against tool assembly with MCP nil + workspace tools false:
	msg := &protocol.Message{Content: "are you here and ready to help me?"}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionCasual,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	tools := a.agentToolDefinitions(msg)
	for _, td := range tools {
		switch td.Name {
		case "read_file", "run_command", "list_dir", "glob", "search_replace", "propose_file_edit",
			activateCapabilityToolName, requestCapabilityHelpToolName, askUserToolName:
			t.Fatalf("presence ping must not expose tool %q", td.Name)
		}
	}
}

func (a *Agent) executeRequestlessActivationForTest(_ context.Context, msg *protocol.Message, id string) (string, error) {
	return a.executeActivateCapabilityTool(msg, []byte(`{"capability_id":"`+id+`"}`))
}

func TestIsConversationalOnlyTurn_meetingNotesStayChatOnly(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "can you give me a summary of the notes from my last meeting/"}
	if !isConversationalOnlyTurn(msg) {
		t.Fatal("expected meeting-note questions to stay conversational-only")
	}
	if shouldOfferCapabilityTools(msg) {
		t.Fatal("expected capability tools suppressed for meeting-note questions")
	}
}

func TestIsConversationalOnlyTurn_casualAskUserStaysChatOnly(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "what should I prioritize today?"}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionCasual,
		RequestedAction: intent.ActionAskUser,
		Action:          intent.ActionAskUser,
		Mutation:        intent.MutationNone,
		Confidence:      1,
		Source:          intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !isConversationalOnlyTurn(msg) {
		t.Fatal("expected casual+ask_user misclassification to stay conversational-only")
	}
}

func TestIsConversationalOnlyTurn_contextScopeNone(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{
		Content: "Correction: refer to the component as ThemeSettings, not SettingsPanel.",
		Metadata: map[string]interface{}{
			MetadataConversationMode: ConversationModeChat,
			MetadataContextScope:     ContextScopeNone,
			"workspace_context": map[string]interface{}{
				"workspace_path": "/tmp/fixture",
			},
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionCorrection,
		RequestedAction: intent.ActionEdit,
		Action:          intent.ActionEdit,
		Mutation:        intent.MutationWorkspace,
		Retrieval:       []intent.RetrievalTarget{intent.RetrievalCodebase},
		Confidence:      1,
		Source:          intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !isConversationalOnlyTurn(msg) {
		t.Fatal("context_scope=none must stay conversational-only even for InteractionCorrection")
	}
}

func TestShouldAugmentPromptWithWorkspace_contextScopeNone(t *testing.T) {
	t.Parallel()
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	a.WorkspacePath = "/tmp/sticky"
	msg := &protocol.Message{
		Content: "Correction: refer to the component as ThemeSettings.",
		Metadata: map[string]interface{}{
			MetadataConversationMode: ConversationModeChat,
			MetadataContextScope:     ContextScopeNone,
			"workspace_context": map[string]interface{}{
				"workspace_path": "/tmp/fixture",
				"file_tree":      "README.md\n",
			},
		},
	}
	if a.shouldAugmentPromptWithWorkspace(IntentTask, msg) {
		t.Fatal("context_scope=none must not augment prompt with workspace files")
	}
}
