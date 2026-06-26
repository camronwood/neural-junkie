package hub

import (
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
