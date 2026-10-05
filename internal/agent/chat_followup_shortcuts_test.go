package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestTryConfusedFollowUpResponse(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "What?")
	out, ok := tryConfusedFollowUpResponse(msg)
	if !ok || !strings.Contains(out, "clarify") {
		t.Fatalf("expected confused follow-up shortcut, got (%q, %v)", out, ok)
	}
	for _, banned := range []string{"I want to add theme", "The user wants to add theme", "successfully added"} {
		if strings.Contains(out, banned) {
			t.Fatalf("shortcut echoed banned %q: %s", banned, out)
		}
	}
	theme := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "I want to add theme support")
	if _, ok := tryConfusedFollowUpResponse(theme); ok {
		t.Fatal("theme request must not use confused-follow-up shortcut")
	}
}

func TestTryOpenFileFactResponse_packageDeclaration(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "core", "sample")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package main\n\nfunc HelloWorld() {}\nfunc main() { HelloWorld() }\n"
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Info: protocol.AgentInfo{Name: "BackendEngineer", Type: protocol.AgentTypeBackend}}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-u-be",
		protocol.AgentInfo{Name: "User", Type: "human"},
		"What Go package declaration is at the top of that file?",
	)
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeFocus,
		"workspace_path":     dir,
		"workspace_context": map[string]interface{}{
			"workspace_path": dir,
			"open_files": []interface{}{
				map[string]interface{}{
					"path":      "core/sample/main.go",
					"language":  "go",
					"is_active": true,
					"content":   body,
				},
			},
		},
	}
	out, ok := a.tryOpenFileFactResponse(msg)
	if !ok {
		t.Fatal("expected open-file fact shortcut")
	}
	for _, want := range []string{"package main", "core/sample", "HelloWorld"} {
		if !strings.Contains(out, want) {
			t.Fatalf("reply missing %q:\n%s", want, out)
		}
	}
}

func TestUserAsksReferencedFileFact(t *testing.T) {
	if !userAsksReferencedFileFact("What Go package declaration is at the top of that file?") {
		t.Fatal("expected match")
	}
	if userAsksReferencedFileFact("can you see my workspace?") {
		t.Fatal("workspace visibility is not a file-fact ask")
	}
}
