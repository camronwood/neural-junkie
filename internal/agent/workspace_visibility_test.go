package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestTryWorkspaceVisibilityResponse(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "BackendEngineer", Type: protocol.AgentTypeBackend}}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-u-be",
		protocol.AgentInfo{Name: "User", Type: "human"},
		"can you see my workspace I have open?",
	)
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeOutline,
		"workspace_context": map[string]interface{}{
			"workspace_name": "neural-junkie",
			"workspace_path": "/Users/me/neural-junkie",
			"file_tree":      "desktop/\ninternal/\n",
			"open_files": []interface{}{
				map[string]interface{}{
					"path":      "/Users/me/neural-junkie/desktop/src/App.tsx",
					"language":  "typescript",
					"is_active": true,
				},
			},
		},
	}
	out, ok := a.tryWorkspaceVisibilityResponse(msg)
	if !ok {
		t.Fatal("expected visibility shortcut")
	}
	for _, want := range []string{"Yes", "neural-junkie", "outline", "App.tsx"} {
		if !strings.Contains(out, want) {
			t.Fatalf("reply missing %q:\n%s", want, out)
		}
	}
}

func TestUserAsksAboutWorkspaceVisibility_phrase(t *testing.T) {
	if !userAsksAboutWorkspaceVisibility("can you see my workspace I have open?") {
		t.Fatal("expected match for workspace I have open")
	}
}

func TestLooksLikeIgnoresWorkspaceVisibility(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"}, "can you see my workspace?")
	bad := "The best answer is to use golang.org/x/themes. go get golang.org/x/themes"
	if !looksLikeIgnoresWorkspaceVisibility(msg, bad) {
		t.Fatal("expected ignore detection")
	}
	good := "Yes — I have workspace context. Project: neural-junkie"
	if looksLikeIgnoresWorkspaceVisibility(msg, good) {
		t.Fatal("expected valid visibility answer")
	}
}
