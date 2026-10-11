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

func TestListDirGlobGrep_pathConfusionTree(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg", "beta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "App.tsx"), []byte("ALPHA_TSX_MARKER"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "App.js"), []byte("DECOY_APP_JS_MARKER"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "alpha", "config.go"), []byte("package alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "beta", "config.go"), []byte("package beta"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := mcp.NewInProcessMCPServer("test-path-confusion", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachTools(srv, func() string { return root })

	list := srv.GetTool("list_dir")
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "list_dir"
	req.Params.Arguments = map[string]any{"path": "src"}
	res, err := list.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("list_dir error: %+v", res)
	}
	text := toolText(res)
	if !strings.Contains(text, "App.tsx") || !strings.Contains(text, "App.js") {
		t.Fatalf("list_dir missing entries:\n%s", text)
	}

	glob := srv.GetTool("glob_file_search")
	req = mcpgo.CallToolRequest{}
	req.Params.Name = "glob_file_search"
	req.Params.Arguments = map[string]any{"pattern": "config.go"}
	res, err = glob.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("glob error: %+v", res)
	}
	text = toolText(res)
	if !strings.Contains(text, "pkg/alpha/config.go") || !strings.Contains(text, "pkg/beta/config.go") {
		t.Fatalf("glob missing configs:\n%s", text)
	}

	grep := srv.GetTool("grep")
	req = mcpgo.CallToolRequest{}
	req.Params.Name = "grep"
	req.Params.Arguments = map[string]any{"pattern": "ALPHA_TSX_MARKER"}
	res, err = grep.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("grep error: %+v", res)
	}
	if !strings.Contains(toolText(res), "App.tsx") {
		t.Fatalf("grep missed App.tsx:\n%s", toolText(res))
	}

	read := srv.GetTool("read_file")
	req = mcpgo.CallToolRequest{}
	req.Params.Name = "read_file"
	req.Params.Arguments = map[string]any{"path": "src/DoesNotExist.tsx"}
	res, err = read.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for missing path")
	}
}

func toolText(res *mcpgo.CallToolResult) string {
	if res == nil {
		return ""
	}
	text := ""
	for _, c := range res.Content {
		if tc, ok := c.(mcpgo.TextContent); ok {
			text += tc.Text
		}
	}
	return text
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
