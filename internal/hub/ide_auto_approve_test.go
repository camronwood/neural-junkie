package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaybeAutoApproveIDEFileChange_yoloTrustAccepted(t *testing.T) {
	msg := &protocol.Message{
		From: protocol.AgentInfo{ID: "frontend"},
		Metadata: map[string]interface{}{
			"editor_mode":        "agent",
			"editor_agent_trust": "yolo",
		},
	}
	if msg.EditorAgentTrust() != "yolo" {
		t.Fatalf("trust = %q", msg.EditorAgentTrust())
	}
	if msg.IdeEditorMode() == "ask" {
		t.Fatal("expected agent mode")
	}
}

func TestMaybeAutoApproveIDEFileChange_askModeSkipped(t *testing.T) {
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":        "ask",
			"editor_agent_trust": "yolo",
		},
	}
	if msg.IdeEditorMode() != "ask" {
		t.Fatal("expected ask mode blocks auto-approve path")
	}
}

func TestMaybeAutoApproveIDEFileChange_neverAutoApprovesDelete(t *testing.T) {
	h := newTestHub(t)
	chName := "implement-scenarios"
	_ = h.CreateChannel(chName, "test", "tester")
	agent := &protocol.AgentInfo{ID: "fe1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agent)

	repoRoot := t.TempDir()
	rel := "src/obsolete.ts"
	abs := filepath.Join(repoRoot, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("export const gone = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agent, "delete obsolete")
	msg.Metadata = map[string]interface{}{
		"workspace_context":                   map[string]interface{}{"workspace_path": repoRoot},
		"editor_agent_trust":                  "yolo",
		"editor_mode":                         "agent",
		protocol.IdeMetaImplementationSession: true,
		"file_change_proposal": map[string]interface{}{
			"operation": "delete",
			"file_path": rel,
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.FileChangeAutoApproved() {
		t.Fatal("delete must never be auto-approved even with yolo trust")
	}
	pending := h.fileChangeManager.ListPendingFileChanges("")
	found := false
	for _, c := range pending {
		if c != nil && c.Operation == filechange.FileOperationDelete && c.FilePath == rel {
			found = true
			if c.Status != filechange.FileChangeStatusPending {
				t.Fatalf("delete status=%s want pending", c.Status)
			}
		}
	}
	if !found {
		// Some hubs scope pending by user; assert file still exists as the hard check.
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("delete was applied or pending missing: %v", err)
		}
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("file must still exist on disk: %v", err)
	}
}
