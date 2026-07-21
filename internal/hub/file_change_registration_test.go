package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSendMessage_fileChangeRegistrationFailureSurfacesSystemMessage(t *testing.T) {
	h := newTestHub(t)
	chName := "reg-fail"
	_ = h.CreateChannel(chName, "test", "tester")

	agent := &protocol.AgentInfo{ID: "fe1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agent)

	repoRoot := t.TempDir()
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agent, "Proposing edit")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"editor_agent_trust": "auto_apply_edits",
		"editor_mode":        "agent",
		"file_change_proposal": map[string]interface{}{
			"operation":   "edit",
			"file_path":   "src-tauri/vite.config.js",
			"new_content": "export default {}\n",
		},
	}
	if err := h.SendMessage(msg); err == nil {
		t.Fatal("expected registration error for missing edit target")
	}

	h.mu.RLock()
	msgs := h.messages[chName]
	h.mu.RUnlock()

	var sawSystem bool
	for _, m := range msgs {
		if m.Type == protocol.MessageTypeSystemInfo && strings.Contains(m.Content, "was not registered") {
			sawSystem = true
			break
		}
	}
	if !sawSystem {
		t.Fatal("expected system message explaining registration failure")
	}
	if len(h.fileChangeManager.ListPendingFileChanges(agent.ID)) > 0 {
		t.Fatal("expected no pending changes after registration failure")
	}
}

func TestSendMessage_redirectedProposalExecutesResolvedTarget(t *testing.T) {
	h := newTestHub(t)
	chName := "reg-redirect"
	_ = h.CreateChannel(chName, "test", "tester")
	agentInfo := &protocol.AgentInfo{ID: "fe2", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agentInfo)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{"react":"^18"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "App.tsx"), []byte("export default function App() { return null }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	replacement := "export default function App() { return <main>Fixed</main> }\n"
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agentInfo, "Proposing wrong alternate path")
	msg.Metadata = map[string]interface{}{
		"workspace_context":  map[string]interface{}{"workspace_path": root},
		"editor_agent_trust": "auto_apply_edits",
		"editor_mode":        "agent",
		"file_change_proposal": map[string]interface{}{
			"operation": "create", "file_path": "src/App.js", "new_content": replacement,
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "src", "App.tsx"))
	if strings.TrimSpace(string(got)) != strings.TrimSpace(replacement) {
		t.Fatalf("redirected target was not updated: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "src", "App.js")); !os.IsNotExist(err) {
		t.Fatal("proposal executed stale pre-redirect path")
	}
}
