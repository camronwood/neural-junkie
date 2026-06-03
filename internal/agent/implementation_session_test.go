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

func TestDetectVerifyCommands_go(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmds := detectVerifyCommands(dir)
	if len(cmds) != 1 || cmds[0] != "go test ./..." {
		t.Fatalf("got %v", cmds)
	}
}

func TestDetectVerifyCommands_nodeBuild(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts":{"build":"vite build","test":"node test.js"}}`)
	writeFile(t, dir, "tsconfig.json", `{}`)
	cmds := detectVerifyCommands(dir)
	if len(cmds) < 2 {
		t.Fatalf("expected build + test, got %v", cmds)
	}
	if cmds[0] != "npm run build" {
		t.Fatalf("first cmd: got %q", cmds[0])
	}
}

func TestGroundingSatisfied(t *testing.T) {
	t.Parallel()
	st := &ImplementationSessionState{StackManifest: &StackManifest{EntryPoint: "src/App.tsx"}}
	if !st.groundingSatisfied() {
		t.Fatal("entry point should satisfy grounding")
	}
	st2 := &ImplementationSessionState{}
	if st2.groundingSatisfied() {
		t.Fatal("empty state should not satisfy grounding")
	}
}

func TestAppendUnique(t *testing.T) {
	got := appendUnique([]string{"a"}, []string{"b", "a"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
}
