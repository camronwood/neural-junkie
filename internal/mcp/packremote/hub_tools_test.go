package packremote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
)

func TestLoadHubToolsCatalogAndAttach(t *testing.T) {
	dir := t.TempDir()
	mcpDir := filepath.Join(dir, "assets", "mcp")
	if err := os.MkdirAll(mcpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	catalog := hubToolsFile{
		Version: 1,
		Tools: []HubToolSpec{
			{
				Name:        "browser_screenshot",
				Description: "Capture a screenshot",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{"type": "string"},
					},
					"required": []any{"url"},
				},
			},
		},
	}
	raw, _ := json.Marshal(catalog)
	if err := os.WriteFile(filepath.Join(mcpDir, "tools.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pack.yaml"), []byte("id: web-browser\nversion: \"2.2.0\"\ntitle: Web browser\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tools, err := LoadHubToolsCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "browser_screenshot" {
		t.Fatalf("unexpected tools: %+v", tools)
	}

	var called bool
	srvHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp/call" {
			http.NotFound(w, r)
			return
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"text":"ok from sidecar"}`))
	}))
	t.Cleanup(srvHTTP.Close)

	prevDir := PackDirResolver
	prevBase := PackBaseURLResolver
	PackDirResolver = func(string) (string, error) { return dir, nil }
	PackBaseURLResolver = func(string) string { return srvHTTP.URL }
	t.Cleanup(func() {
		PackDirResolver = prevDir
		PackBaseURLResolver = prevBase
	})

	mcpSrv, err := mcp.NewInProcessMCPServer("test-hub-mcp", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !AttachHubPackTools(mcpSrv, "web-browser") {
		t.Fatal("expected tools attached")
	}
	if mcpSrv.GetTool("browser_screenshot") == nil {
		t.Fatal("missing browser_screenshot")
	}

	hub, err := NewHubPackMCP("web-browser", "Browser")
	if err != nil {
		t.Fatal(err)
	}
	if hub.GetMCPServer().GetTool("browser_screenshot") == nil {
		t.Fatal("hub mcp missing tool")
	}
	_ = called
}
