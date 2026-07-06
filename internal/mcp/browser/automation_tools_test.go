package browser

import (
	"testing"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
)

func TestBrowserMCPRegistersAutomationTools(t *testing.T) {
	srv, err := mcp.NewInProcessMCPServer("test-browser-mcp", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	AttachAutomationTools(srv)
	for _, want := range []string{
		"browser_screenshot",
		"browser_navigate",
		"browser_click",
		"browser_fill",
		"browser_a11y_audit",
		"browser_metrics",
	} {
		if srv.GetTool(want) == nil {
			t.Fatalf("missing tool %q", want)
		}
	}
}
