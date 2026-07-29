package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestWithSharedWebSearchAllowlist(t *testing.T) {
	t.Parallel()
	if got := withSharedWebSearchAllowlist(nil); got != nil {
		t.Fatalf("empty allowlist should stay empty (all tools), got %v", got)
	}
	got := withSharedWebSearchAllowlist([]string{"maps_geocode", "maps_route"})
	want := map[string]bool{"maps_geocode": true, "maps_route": true, "web_search": true, "fetch_url": true}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, name := range got {
		if !want[name] {
			t.Fatalf("unexpected %q in %v", name, got)
		}
	}
}

func TestEnsureAgentWebSearchTools_createsSharedMCP(t *testing.T) {
	t.Parallel()
	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeArena, "Arena", nil, ai.NewMockProvider(), hub)
	if ag.MCPServer != nil {
		t.Fatal("expected no MCP before ensure")
	}
	ensureAgentWebSearchTools(ag)
	if ag.MCPServer == nil {
		t.Fatal("expected shared web MCP")
	}
	srv := mcpServerFromInterface(ag.MCPServer)
	if srv == nil || srv.GetTool("web_search") == nil || srv.GetTool("fetch_url") == nil {
		t.Fatal("expected web_search and fetch_url tools")
	}
	// Idempotent
	ensureAgentWebSearchTools(ag)
}
