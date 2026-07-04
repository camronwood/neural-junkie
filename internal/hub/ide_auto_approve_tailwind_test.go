package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSendMessage_autoApproveTailwindConfigImplementScenario(t *testing.T) {
	h := newTestHub(t)
	chName := "implement-scenarios"
	_ = h.CreateChannel(chName, "test", "tester")

	agent := &protocol.AgentInfo{ID: "fe1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agent)

	repoRoot := t.TempDir()
	tailRel := "tailwind.config.js"
	existing := `/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: { extend: {} },
  plugins: [],
};
`
	newBody := `/** @type {import('tailwindcss').Config} */
export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: { extend: {} },
  plugins: [],
};
`
	if err := os.WriteFile(filepath.Join(repoRoot, tailRel), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agent, "Proposing tailwind edit")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"editor_agent_trust": "auto_apply_edits",
		"editor_mode":        "agent",
		protocol.IdeMetaImplementationSession: true,
		"file_change_proposal": map[string]interface{}{
			"operation":   "edit",
			"file_path":   tailRel,
			"old_content": existing,
			"new_content": newBody,
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if !msg.FileChangeAutoApproved() {
		t.Fatal("expected tailwind.config.js auto-approved on implement-scenarios channel")
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, tailRel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "darkMode") {
		t.Fatalf("expected darkMode on disk, got:\n%s", got)
	}
}
