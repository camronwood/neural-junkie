package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func TestListDirTool_hidesScenarioBaseline(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".scenario-baseline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".scenario-baseline", "Makefile"), []byte("seed:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := mcp.NewInProcessMCPServer("test-list-dir", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachTools(srv, func() string { return root })
	st := srv.GetTool("list_dir")
	if st == nil {
		t.Fatal("list_dir not registered")
	}
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "list_dir"
	req.Params.Arguments = map[string]any{"path": "."}
	res, err := st.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res)
	}
	text := ""
	for _, c := range res.Content {
		if tc, ok := c.(mcpgo.TextContent); ok {
			text += tc.Text
		}
	}
	if strings.Contains(text, ".scenario-baseline") {
		t.Fatalf("list_dir leaked scenario baseline:\n%s", text)
	}
	if !strings.Contains(text, "Makefile") {
		t.Fatalf("expected live Makefile in listing:\n%s", text)
	}
}

func TestReadFileTool_rejectsScenarioBaseline(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".scenario-baseline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".scenario-baseline", "Makefile"), []byte("seed:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := mcp.NewInProcessMCPServer("test-read-baseline", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachTools(srv, func() string { return root })
	st := srv.GetTool("read_file")
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "read_file"
	req.Params.Arguments = map[string]any{"path": ".scenario-baseline/Makefile"}
	res, err := st.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error reading harness seed path")
	}
}

func TestReadFileTool_stripsRedundantFixturePrefix(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "react-vite-corrupt-appjs")
	if err := os.MkdirAll(filepath.Join(fixture, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "src", "App.js"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := mcp.NewInProcessMCPServer("test-workspace", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachTools(srv, func() string { return fixture })
	st := srv.GetTool("read_file")
	if st == nil {
		t.Fatal("read_file not registered")
	}
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "read_file"
	req.Params.Arguments = map[string]any{"path": "scenarios/fixtures/react-vite-corrupt-appjs/src/App.js"}
	res, err := st.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res)
	}
}

func TestSemanticSearchTool_usesWorkspaceRootFromContext(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello workspace"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := mcp.NewInProcessMCPServer("test-workspace-ctx", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	// RootResolver empty — root must come from request context.
	AttachTools(srv, func() string { return "" })
	st := srv.GetTool("semantic_search")
	if st == nil {
		t.Fatal("semantic_search not registered")
	}
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "semantic_search"
	req.Params.Arguments = map[string]any{"query": "hello"}

	emptyRes, err := st.Handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if emptyRes == nil || !emptyRes.IsError {
		t.Fatal("expected error when workspace root not set")
	}

	ctx := shared.ContextWithWorkspaceRoot(context.Background(), root)
	res, err := st.Handler(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error with context root: %+v", res)
	}
}
