package workspace

import (
	"os"
	"path/filepath"
	"testing"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func TestReadFileTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := mcp.NewInProcessMCPServer("test-workspace", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	root := dir
	AttachTools(srv, func() string { return root })
	st := srv.GetTool("read_file")
	if st == nil {
		t.Fatal("read_file not registered")
	}
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "read_file"
	req.Params.Arguments = map[string]any{"path": "hello.txt"}
	res, err := st.Handler(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res)
	}
}

func TestListDirTool(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a"), 0o644)
	srv, _ := mcp.NewInProcessMCPServer("test-workspace", "1.0.0")
	AttachTools(srv, func() string { return dir })
	st := srv.GetTool("list_dir")
	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]any{"path": "."}
	res, err := st.Handler(t.Context(), req)
	if err != nil || res.IsError {
		t.Fatalf("list_dir failed: %v %+v", err, res)
	}
}
