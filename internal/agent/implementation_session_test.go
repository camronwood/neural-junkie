package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldRunImplementationSession(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend, Name: "BackendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, "please implement a health check handler")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"ide_route_agent_type":   "backend",
		"implementation_session": true,
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("expected implementation session")
	}
	msg.Metadata["editor_mode"] = "ask"
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("ask mode should not run session")
	}
}

func TestDetectVerifyCommand(t *testing.T) {
	dir := t.TempDir()
	if cmd := detectVerifyCommand(dir); cmd != "" {
		t.Fatalf("empty dir: %q", cmd)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cmd := detectVerifyCommand(dir); cmd != "go test ./..." {
		t.Fatalf("got %q", cmd)
	}
}

func TestAppendUnique(t *testing.T) {
	got := appendUnique([]string{"a"}, []string{"b", "a"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
}
