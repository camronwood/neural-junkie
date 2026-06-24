package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestIsProtectedWorkspaceFile(t *testing.T) {
	t.Parallel()
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"implement-scenarios",
		protocol.AgentInfo{ID: "user", Name: "camronwood", Type: protocol.AgentTypeGeneral},
		"In @file:src/App.tsx ONLY add a subtitle. Do NOT modify tailwind.config.js or package.json.",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"unchanged_files": []interface{}{"tailwind.config.js", "package.json"},
		},
	}
	if !isProtectedWorkspaceFile(msg, "tailwind.config.js") {
		t.Fatal("expected tailwind.config.js protected via metadata")
	}
	if !isProtectedWorkspaceFile(msg, "package.json") {
		t.Fatal("expected package.json protected via metadata")
	}
	if isProtectedWorkspaceFile(msg, "src/App.tsx") {
		t.Fatal("App.tsx should remain editable")
	}
}
