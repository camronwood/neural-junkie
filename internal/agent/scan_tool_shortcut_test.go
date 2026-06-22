package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/biology"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSharedScanAnalysisPathFromActiveEditor(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "Camron"}, "summarize_scan_analysis on open file")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": "/proj",
			"active_editor": map[string]interface{}{
				"path":              filepath.Join(abs, "reports", "results.json"),
				"view_mode":         "scan-analysis",
				"scan_analysis_dir": abs,
				"is_active":         true,
			},
		},
	}
	path, ok := sharedScanAnalysisPath(msg)
	if !ok {
		t.Fatal("expected path from active_editor")
	}
	if !scanAnalysisPathExists(path) {
		t.Fatalf("path %q is not analysis export", path)
	}
}

func TestRequestedBiologyScanToolFromTurn_followUp(t *testing.T) {
	hub := &scanShortcutHistoryHub{
		msgs: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-b", protocol.AgentInfo{Name: "Camron", Type: "human"},
				"Use summarize_scan_analysis on the file I have open"),
		},
	}
	a := &Agent{Info: protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology}, Hub: hub}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-b", protocol.AgentInfo{Name: "Camron", Type: "human"}, "its open in my editor now")
	if got := a.requestedBiologyScanToolFromTurn(msg); got != "summarize_scan_analysis" {
		t.Fatalf("got %q want summarize_scan_analysis", got)
	}
}

func TestUserAsksAboutOpenScanFile_noBareSummarize(t *testing.T) {
	if userAsksAboutOpenScanFile("can you summarize IL-6 levels?") {
		t.Fatal("bare summarize should not trigger scan shortcut heuristics")
	}
	if userAsksAboutOpenScanFile("thanks, that helps") {
		t.Fatal("closure should not trigger scan shortcut heuristics")
	}
	if !userAsksAboutOpenScanFile("please summarize_scan_analysis on the open export") {
		t.Fatal("explicit tool name should match")
	}
	if !userAsksAboutOpenScanFile("run scan analysis QC on the plate") {
		t.Fatal("scan analysis phrase should match")
	}
}

func TestResolveBiologyScanTool_unrelatedFollowUp(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology}}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-u-b",
		protocol.AgentInfo{Name: "Camron", Type: "human"},
		"what is the dilution factor for IL-6?",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"scan_analysis": map[string]interface{}{
				"analysis_dir": "/Users/me/Downloads/summary-test",
			},
		},
	}
	if got := a.resolveBiologyScanToolForTurn(msg); got != "" {
		t.Fatalf("got %q want empty for unrelated follow-up", got)
	}
}

func TestResolveBiologyScanToolForOpenFileQuestion(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology}}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-u-b",
		protocol.AgentInfo{Name: "Camron", Type: "human"},
		"whatdo you see in the file I haveopen now?",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": "/Users/me/Downloads/summary-test",
			"scan_analysis": map[string]interface{}{
				"analysis_dir": "/Users/me/Downloads/summary-test",
			},
			"open_files": []interface{}{
				map[string]interface{}{
					"path":      "/Users/me/Downloads/summary-test/reports/results.json",
					"is_active": true,
				},
			},
		},
	}
	if got := a.resolveBiologyScanToolForTurn(msg); got != "summarize_scan_analysis" {
		t.Fatalf("got %q want summarize_scan_analysis", got)
	}
}

func TestTryBiologyScanToolShortcutRunsTool(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(abs, "reports", "results.json")); err != nil {
		t.Skip("scan-analysis fixture missing")
	}

	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	config.SetupTestOfficialPackCatalog(t)
	if err := cfg.InstallPack(config.PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[config.PackLifeSciences] = true
	cfg.SyncAgentsFromPacks()
	mcp.SetAppConfig(cfg)

	bioMCP, err := biology.NewBiologyMCP()
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Info:      protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology},
		MCPServer: bioMCP,
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{Name: "Camron"},
		"please use summarize_scan_analysis on the file I have open",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": abs,
			"scan_analysis": map[string]interface{}{
				"analysis_dir": abs,
				"note":         "viewer active",
			},
			"active_editor": map[string]interface{}{
				"scan_analysis_dir": abs,
				"is_active":         true,
			},
		},
	}

	out, ok := a.tryBiologyScanToolShortcut(context.Background(), msg)
	if !ok {
		t.Fatal("expected shortcut to run summarize_scan_analysis")
	}
	if out == "" {
		t.Fatal("expected tool output")
	}
}

type scanShortcutHistoryHub struct {
	msgs []*protocol.Message
}

func (h *scanShortcutHistoryHub) SendMessage(*protocol.Message) error { return nil }
func (h *scanShortcutHistoryHub) BroadcastDirect(string, *protocol.Message) {
}
func (h *scanShortcutHistoryHub) Subscribe(string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}
func (h *scanShortcutHistoryHub) GetMessages(string, int) ([]*protocol.Message, error) {
	return h.msgs, nil
}
func (h *scanShortcutHistoryHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) {
	return nil, nil
}
func (h *scanShortcutHistoryHub) GetThreadParentAuthor(string) string { return "" }
func (h *scanShortcutHistoryHub) GetCommandHandler() CommandHandlerInterface {
	return nil
}
func (h *scanShortcutHistoryHub) GetAgentChannels(string) []string { return nil }
func (h *scanShortcutHistoryHub) GetChannelType(string) protocol.ChannelType {
	return protocol.ChannelTypeDM
}
func (h *scanShortcutHistoryHub) GetChannelSessionSummary(string) string { return "" }
func (h *scanShortcutHistoryHub) GetThreadMessages(string, int) ([]*protocol.Message, error) {
	return nil, nil
}
func (h *scanShortcutHistoryHub) IsChannelHeld(string) bool            { return false }
func (h *scanShortcutHistoryHub) ImageGenerationEnabled() bool         { return false }
func (h *scanShortcutHistoryHub) GenerateAndPostImage(context.Context, string, protocol.AgentInfo, string, string) error {
	return nil
}
