package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
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

func TestFormatCallToolResult_compressesLargeGrep(t *testing.T) {
	enabled := true
	ai.SetHubRuntimeOptions(
		config.PerformanceConfig{
			ContextCompressEnabled:      &enabled,
			ContextCompressMaxToolBytes: 2000,
		},
		config.OllamaConfig{},
	)
	contextcompress.SetDefaultStore(contextcompress.NewStore(50, 60, ""))
	t.Cleanup(func() { contextcompress.SetDefaultStore(nil) })

	var lines []string
	for i := 0; i < 400; i++ {
		lines = append(lines, "file.go:1:match")
	}
	raw := strings.Join(lines, "\n")
	ctx := contextcompress.WithCompressContext(context.Background(), "test-ch", "call-1")
	out := formatCallToolResult(ctx, "grep", &mcpgo.CallToolResult{
		Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: raw}},
	})
	if len(out) >= len(raw) {
		t.Fatalf("expected compression, got len %d", len(out))
	}
	if !strings.Contains(out, "ctx-") {
		t.Fatal("expected ref marker")
	}
}
