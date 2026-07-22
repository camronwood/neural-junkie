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
		"are you here and ready to hlep?",
		"you still there?",
	} {
		msg := &protocol.Message{Content: content}
		if shouldOfferCapabilityTools(msg) {
			t.Fatalf("expected capability tools suppressed for %q", content)
		}
	}
	if !shouldOfferCapabilityTools(&protocol.Message{Content: "activate biology-api and analyze this FASTA"}) {
		t.Fatal("expected capability tools for a real capability task")
	}
}

func (a *Agent) executeRequestlessActivationForTest(_ context.Context, msg *protocol.Message, id string) (string, error) {
	return a.executeActivateCapabilityTool(msg, []byte(`{"capability_id":"`+id+`"}`))
}
