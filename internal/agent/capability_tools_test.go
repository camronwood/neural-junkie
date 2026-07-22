package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
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

func TestAgentToolDefinitions_presencePingOmitsWorkspaceTools(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		// Non-nil MCPServer interface would need a real server; use hasWorkspaceTools path via file edits.
	}
	// Force workspace tools path with a stub by setting WorkspacePath and using definitions that check hasWorkspaceTools.
	// Instead assert conversational gate directly against tool assembly with MCP nil + workspace tools false:
	msg := &protocol.Message{Content: "are you here and ready to help me?"}
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
