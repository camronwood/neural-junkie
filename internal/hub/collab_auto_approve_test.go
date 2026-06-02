package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaybeAutoApproveCollabFileChange_ApprovesUnderCollabsDir(t *testing.T) {
	h := newTestHub(t)
	chName := "auto-approve-test"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Architect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "collabs"), 0755); err != nil {
		t.Fatal(err)
	}

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"write findings",
		[]string{"a1", "a2"},
		chName,
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{SourceRepoPath: repoRoot},
	)
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	rel := collaboration.ProjectCollabRelPath(collab.ID) + "/findings.md"
	task := collaboration.CollaborationTask{
		ID:           "t1",
		Title:        "Write findings",
		Description:  "Write " + rel,
		AssignedTo:   "a1",
		AssignedName: "Assistant",
		Status:       collaboration.TaskInProgress,
	}
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{task})

	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *a1, "Proposing file change")
	msg.SetCollaborationID(collab.ID)
	msg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"file_change_proposal": map[string]interface{}{
			"operation":   "create",
			"file_path":   rel,
			"new_content": "# Findings\n- one\n- two\n- three\n",
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("expected auto-approved file at %s: %v", abs, err)
	}

	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.Tasks[0].Status != collaboration.TaskCompleted {
		t.Fatalf("expected task completed after auto-approve, got %s", after.Tasks[0].Status)
	}

	for _, p := range h.fileChangeManager.ListPendingFileChanges("a1") {
		if p != nil && p.Status == filechange.FileChangeStatusPending {
			t.Fatalf("expected no pending changes after auto-approve, still have %s", p.ID)
		}
	}
}

func TestCollabAutoApproveDeliverablesDisabled(t *testing.T) {
	h := newTestHub(t)
	cfg := config.DefaultConfig()
	falseVal := false
	cfg.Collaboration.AutoApproveDeliverables = &falseVal
	h.commandHandler.SetProviderRegistry(cfg, nil)

	chName := "auto-approve-off"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Architect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	repoRoot := t.TempDir()
	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"write findings",
		[]string{"a1", "a2"},
		chName,
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{SourceRepoPath: repoRoot},
	)
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)

	rel := collaboration.ProjectCollabRelPath(collab.ID) + "/findings.md"
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *a1, "Proposing file change")
	msg.SetCollaborationID(collab.ID)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"file_change_proposal": map[string]interface{}{
			"operation":   "create",
			"file_path":   rel,
			"new_content": "hello",
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); err == nil {
		t.Fatal("expected file not written when auto-approve disabled")
	}
	if pending := h.fileChangeManager.ListPendingFileChanges("a1"); len(pending) == 0 {
		t.Fatal("expected pending file change when auto-approve disabled")
	}
}

func TestIsPathUnderCollabDeliverable(t *testing.T) {
	root := t.TempDir()
	collabID := "abc-123"
	prefix := collaboration.ProjectCollabRelPath(collabID)
	okPath := filepath.Join(root, filepath.FromSlash(prefix+"/findings.md"))
	if !isPathUnderCollabDeliverable(collabID, root, okPath) {
		t.Fatal("expected deliverable path under collabs/<id>")
	}
	outside := filepath.Join(root, "src/main.go")
	if isPathUnderCollabDeliverable(collabID, root, outside) {
		t.Fatal("expected outside path to be rejected")
	}
}
