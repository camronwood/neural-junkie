package agent

import (
	"context"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaybeSubmitFileChangeFromResponse_askModeStripsBlocks(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "be-1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend}}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":            "ask",
			"implementation_session": true,
		},
	}
	resp := "Here is a draft.\n[FILE_CHANGE]\noperation: edit\npath: core/sample/main.go\n[/FILE_CHANGE]"
	cleaned, proposed, err := a.maybeSubmitFileChangeFromResponse(context.Background(), resp, "implement-scenarios", msg)
	if err != nil {
		t.Fatal(err)
	}
	if proposed {
		t.Fatal("ask mode must not propose file changes")
	}
	if strings.Contains(cleaned, "[FILE_CHANGE]") {
		t.Fatalf("expected stripped response, got %q", cleaned)
	}
}

func TestAgentToolDefinitions_askModeOmitsProposeFileEdit(t *testing.T) {
	t.Parallel()
	srv := server.NewMCPServer("test", "1.0.0")
	srv.AddTool(mcpgo.Tool{
		Name:        "read_file",
		Description: "read a file",
		InputSchema: mcpgo.ToolInputSchema{Type: "object"},
	}, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: "ok"}},
		}, nil
	})
	a := &Agent{
		Info:      protocol.AgentInfo{ID: "be-1", Type: protocol.AgentTypeBackend},
		MCPServer: &stubMCP{srv: srv},
	}
	askMsg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":            "ask",
			"implementation_session": true,
		},
	}
	for _, td := range a.agentToolDefinitions(askMsg) {
		if td.Name == proposeFileEditToolName {
			t.Fatal("ask mode must not expose propose_file_edit")
		}
	}
	editMsg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":            "edit",
			"implementation_session": true,
		},
	}
	var hasPropose bool
	for _, td := range a.agentToolDefinitions(editMsg) {
		if td.Name == proposeFileEditToolName {
			hasPropose = true
		}
	}
	if !hasPropose {
		t.Fatal("edit mode should expose propose_file_edit when workspace tools are available")
	}
}

func TestExecuteProposeFileEditTool_askModeRejected(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "be-1", Type: protocol.AgentTypeBackend}}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{"editor_mode": "ask"},
	}
	_, err := a.executeProposeFileEditTool(context.Background(), msg, []byte(`{"path":"x.go","content":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("expected read-only error, got %v", err)
	}
}

func TestSanitizeAskModeResponse_stripsMentions(t *testing.T) {
	t.Parallel()
	raw := "Here is advice.\nUse [FILE_CHANGE] or propose_file_edit to add the handler."
	got := sanitizeAskModeResponse(raw)
	lower := strings.ToLower(got)
	if strings.Contains(lower, "[file_change]") || strings.Contains(lower, "propose_file_edit") {
		t.Fatalf("expected stripped mention, got %q", got)
	}
}

func TestShouldUseFileChangeFenceFallback_askMode(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "be-1", Type: protocol.AgentTypeBackend}}
	msg := &protocol.Message{
		Content: "please implement a handler",
		Metadata: map[string]interface{}{
			"editor_mode":            "ask",
			"implementation_session": true,
		},
	}
	if a.shouldUseFileChangeFenceFallback(msg) {
		t.Fatal("ask mode must not use fence fallback even with implementation_session")
	}
}
