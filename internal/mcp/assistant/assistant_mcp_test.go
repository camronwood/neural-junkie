package assistant

import "testing"

func TestNewAssistantMCPRegistersWorkspaceTools(t *testing.T) {
	srv, err := NewAssistantMCP()
	if err != nil {
		t.Fatal(err)
	}
	srv.AttachWorkspaceTools(func() string { return t.TempDir() })
	srv.AttachWebTools()
	mcpSrv := srv.GetMCPServer()
	for _, name := range []string{"read_file", "grep", "glob_file_search", "list_dir", "semantic_search", "web_search", "fetch_url"} {
		if mcpSrv.GetTool(name) == nil {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}
}
