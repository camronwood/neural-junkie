package agent

import (
	"context"
	"encoding/json"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type stubMCP struct {
	srv *server.MCPServer
}

func (s *stubMCP) Start() error { return nil }
func (s *stubMCP) GetMCPServer() *server.MCPServer {
	return s.srv
}

func TestClaudeToolsFromMCPServer(t *testing.T) {
	srv := server.NewMCPServer("test", "1.0.0")
	srv.AddTool(mcpgo.Tool{
		Name:        "echo",
		Description: "echo input",
		InputSchema: mcpgo.ToolInputSchema{Type: "object"},
	}, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: "ok"}},
		}, nil
	})

	tools := claudeToolsFromMCPServer(srv)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Fatalf("unexpected tool name %q", tools[0].Name)
	}
}

func TestExecuteMCPTool(t *testing.T) {
	srv := server.NewMCPServer("test", "1.0.0")
	srv.AddTool(mcpgo.Tool{
		Name:        "echo",
		Description: "echo",
		InputSchema: mcpgo.ToolInputSchema{Type: "object"},
	}, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: "hello"}},
		}, nil
	})

	out, err := executeMCPTool(context.Background(), srv, "echo", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Fatalf("got %q", out)
	}
}

func TestMcpServerFromInterface(t *testing.T) {
	srv := server.NewMCPServer("test", "1.0.0")
	stub := &stubMCP{srv: srv}
	if mcpServerFromInterface(stub) != srv {
		t.Fatal("expected same server instance")
	}
}
