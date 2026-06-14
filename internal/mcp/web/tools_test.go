package web

import (
	"context"
	"testing"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func TestAttachToolsRegistersWebTools(t *testing.T) {
	srv, err := mcp.NewInProcessMCPServer("test-web-mcp", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachTools(srv)
	for _, name := range []string{"web_search", "fetch_url"} {
		if srv.GetTool(name) == nil {
			t.Fatalf("expected tool %q", name)
		}
	}
}

func TestFetchURLBlocksPrivateHost(t *testing.T) {
	srv, err := mcp.NewInProcessMCPServer("test-web-mcp", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachTools(srv)
	st := srv.GetTool("fetch_url")
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "fetch_url"
	req.Params.Arguments = map[string]any{"url": "http://127.0.0.1:9999/"}
	result, err := st.Handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected SSRF error result, got %+v", result)
	}
}

func TestWebSearchNotConfigured(t *testing.T) {
	mcp.SetAppConfig(nil)
	t.Cleanup(func() { mcp.SetAppConfig(nil) })

	srv, err := mcp.NewInProcessMCPServer("test-web-mcp", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachTools(srv)
	st := srv.GetTool("web_search")
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "web_search"
	req.Params.Arguments = map[string]any{"query": "test"}
	result, err := st.Handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected not-configured error, got %+v", result)
	}
}
