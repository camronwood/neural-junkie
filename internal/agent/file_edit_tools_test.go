package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func workspaceMCPServer(t *testing.T) *server.MCPServer {
	t.Helper()
	srv := server.NewMCPServer("test-workspace", "1.0.0")
	srv.AddTool(mcpgo.Tool{
		Name:        "read_file",
		Description: "read",
		InputSchema: mcpgo.ToolInputSchema{Type: "object"},
	}, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: "ok"}},
		}, nil
	})
	return srv
}

func TestAgentToolDefinitions_editModeIncludesPatchTools(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info:      protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend},
		MCPServer: &stubMCP{srv: workspaceMCPServer(t)},
	}
	editMsg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":            "agent",
			"implementation_session": true,
		},
	}
	names := map[string]bool{}
	for _, td := range a.agentToolDefinitions(editMsg) {
		names[td.Name] = true
	}
	for _, want := range []string{searchReplaceToolName, applyPatchToolName, proposeFileEditToolName} {
		if !names[want] {
			t.Fatalf("missing tool %q in agent mode; got %v", want, names)
		}
	}
}

func TestAgentToolDefinitions_askModeOmitsPatchTools(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info:      protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend},
		MCPServer: &stubMCP{srv: workspaceMCPServer(t)},
	}
	askMsg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":            "ask",
			"implementation_session": true,
		},
	}
	for _, td := range a.agentToolDefinitions(askMsg) {
		switch td.Name {
		case searchReplaceToolName, applyPatchToolName, proposeFileEditToolName:
			t.Fatalf("ask mode must not expose %q", td.Name)
		}
	}
}

func TestExecuteSearchReplaceTool_proposesEdit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var sent int
	a := &Agent{
		Info:          protocol.AgentInfo{ID: "be-1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		Hub:           countingHub{inner: shouldRespondTestHub{}, count: &sent},
		MCPServer:     &stubMCP{srv: workspaceMCPServer(t)},
		WorkspacePath: dir,
		Context:       &ConversationContext{CurrentChannel: "test"},
	}
	msg := &protocol.Message{
		Channel: "test",
		Metadata: map[string]interface{}{
			"editor_mode": "agent",
			"workspace_context": map[string]interface{}{
				"workspace_path": dir,
			},
		},
	}
	input := []byte(`{"path":"main.go","old_string":"func main() {}","new_string":"func main() { println(1) }"}`)
	result, err := a.executeSearchReplaceTool(context.Background(), msg, input)
	if err != nil {
		t.Fatal(err)
	}
	if result == "" || sent == 0 {
		t.Fatalf("expected proposal, result=%q sent=%d", result, sent)
	}
}

type countingHub struct {
	inner shouldRespondTestHub
	count *int
}

func (c countingHub) SendMessage(msg *protocol.Message) error {
	*c.count++
	return c.inner.SendMessage(msg)
}

func (c countingHub) BroadcastDirect(channelName string, msg *protocol.Message) {
	c.inner.BroadcastDirect(channelName, msg)
}
func (c countingHub) Subscribe(channelName string) (chan *protocol.Message, error) {
	return c.inner.Subscribe(channelName)
}
func (c countingHub) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	return c.inner.GetMessages(channelName, limit)
}
func (c countingHub) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	return c.inner.GetChannelAgents(channelName)
}
func (c countingHub) GetThreadParentAuthor(threadID string) string { return c.inner.GetThreadParentAuthor(threadID) }
func (c countingHub) GetCommandHandler() CommandHandlerInterface   { return c.inner.GetCommandHandler() }
func (c countingHub) GetAgentChannels(agentID string) []string     { return c.inner.GetAgentChannels(agentID) }
func (c countingHub) GetChannelType(channelName string) protocol.ChannelType {
	return c.inner.GetChannelType(channelName)
}
func (c countingHub) GetChannelSessionSummary(channel string) string { return c.inner.GetChannelSessionSummary(channel) }
func (c countingHub) GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error) {
	return c.inner.GetThreadMessages(threadID, limit)
}
func (c countingHub) IsChannelHeld(channel string) bool { return c.inner.IsChannelHeld(channel) }
func (c countingHub) ImageGenerationEnabled() bool      { return c.inner.ImageGenerationEnabled() }
func (c countingHub) GenerateAndPostImage(ctx context.Context, channel string, from protocol.AgentInfo, prompt, size string) error {
	return c.inner.GenerateAndPostImage(ctx, channel, from, prompt, size)
}
func (c countingHub) MusicGenerationEnabled() bool { return c.inner.MusicGenerationEnabled() }
func (c countingHub) GenerateAndPostMusic(ctx context.Context, channel string, from protocol.AgentInfo, req MusicGenerateRequest) error {
	return c.inner.GenerateAndPostMusic(ctx, channel, from, req)
}
func (c countingHub) ExtractAndPostMusicStems(ctx context.Context, channel string, from protocol.AgentInfo, req MusicExtractRequest) error {
	if h, ok := c.inner.(interface {
		ExtractAndPostMusicStems(context.Context, string, protocol.AgentInfo, MusicExtractRequest) error
	}); ok {
		return h.ExtractAndPostMusicStems(ctx, channel, from, req)
	}
	return nil
}
func (c countingHub) AskUserQuestion(string, string, string, string, []string) (string, error) {
	return "", nil
}


