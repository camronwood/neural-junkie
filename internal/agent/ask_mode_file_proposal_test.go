package agent

import (
	"context"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
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

func TestMaybeSubmitFileChangeFromResponse_mutationNoneStripsParenBlocks(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "will not boot")
	msg.Metadata = map[string]interface{}{
		"editor_mode": "agent",
		"composer_mode": "agent",
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Mutation: intent.MutationNone, Confidence: 0.9, Source: intent.SourceSafeFallback,
	}); err != nil {
		t.Fatal(err)
	}
	resp := "Grounding: loaded files.\n[FILE_CHANGE(src/components/WorkspaceTabBar.tsx)] ```tsx\nexport const X = 1\n```"
	cleaned, proposed, err := a.maybeSubmitFileChangeFromResponse(context.Background(), resp, "dm-test", msg)
	if err != nil {
		t.Fatal(err)
	}
	if proposed {
		t.Fatal("mutation=none must not propose file changes")
	}
	if strings.Contains(cleaned, "FILE_CHANGE") {
		t.Fatalf("expected paren FILE_CHANGE stripped, got %q", cleaned)
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
		switch td.Name {
		case proposeFileEditToolName, searchReplaceToolName, applyPatchToolName:
			t.Fatalf("ask mode must not expose %q", td.Name)
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
	if !toolNamesInclude(a.agentToolDefinitions(editMsg), searchReplaceToolName) {
		t.Fatal("edit mode should expose search_replace")
	}
}

func toolNamesInclude(tools []ai.ClaudeToolDefinition, name string) bool {
	for _, td := range tools {
		if td.Name == name {
			return true
		}
	}
	return false
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

func TestSanitizeAskModeResponse_stripsLooseFileChangeBlock(t *testing.T) {
	t.Parallel()
	raw := "Advice only.\n[FILE_CHANGE]\noperation: edit\npath: core/sample/main.go\n```go\npackage main\n```\n"
	got := sanitizeAskModeResponse(raw)
	if strings.Contains(strings.ToLower(got), "file_change") {
		t.Fatalf("loose block not stripped: %q", got)
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
