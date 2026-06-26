package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestLooksLikePlaceholderDeliverableContent(t *testing.T) {
	cases := []struct {
		content string
		want    bool
	}{
		{"# Report\n\n- **File:** [Insert File Name]\n", true},
		{"# Feature\n\n[Feature Name]\n\n[Brief description of feature]\n", true},
		{"# Findings\n\n- grounded in README\n", false},
		{"", false},
		{"Lorem ipsum dolor sit amet", true},
	}
	for _, tc := range cases {
		if got := looksLikePlaceholderDeliverableContent(tc.content); got != tc.want {
			t.Fatalf("looksLikePlaceholderDeliverableContent(%q) = %v, want %v", tc.content, got, tc.want)
		}
	}
}

func TestRegisterFileChangeProposal_rejectsCancelledCollab(t *testing.T) {
	h := newTestHub(t)
	chName := "cancelled-collab"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend, Status: "active"}
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
	if _, err := cm.CancelCollaboration(collab.ID); err != nil {
		t.Fatal(err)
	}

	rel := collaboration.ProjectCollabRelPath(collab.ID) + "/draft.md"
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *a1, "Proposing file change")
	msg.SetCollaborationID(collab.ID)
	msg.SetCollaborationPhase(string(collaboration.PhaseCancelled))
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"file_change_proposal": map[string]interface{}{
			"operation":   "create",
			"file_path":   rel,
			"new_content": "# Draft\n\nDone.\n",
		},
	}
	if err := h.SendMessage(msg); err == nil {
		t.Fatalf("expected SendMessage error for cancelled collab proposal, got nil")
	}

	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); !os.IsNotExist(err) {
		t.Fatalf("expected no file on disk after cancelled collab proposal, stat err=%v", err)
	}
	if len(h.fileChangeManager.ListPendingFileChanges("a1")) > 0 {
		t.Fatal("expected no pending file changes after rejected proposal")
	}
}

func TestRegisterFileChangeProposal_rejectsAskMode(t *testing.T) {
	h := newTestHub(t)
	chName := "ask-mode"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)

	repoRoot := t.TempDir()
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *a1, "Proposing file change")
	msg.Metadata = map[string]interface{}{
		protocol.IdeMetaEditorMode: "ask",
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"file_change_proposal": map[string]interface{}{
			"operation":   "edit",
			"file_path":   "core/sample/main.go",
			"new_content": "package main\n",
		},
	}
	if err := h.registerFileChangeProposal(msg, msg.Metadata["file_change_proposal"]); err == nil {
		t.Fatal("expected ask-mode proposal rejection")
	}
}

func TestRegisterFileChangeProposal_rejectsPlaceholderContent(t *testing.T) {
	h := newTestHub(t)
	chName := "placeholder-collab"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend, Status: "active"}
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
	msg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"file_change_proposal": map[string]interface{}{
			"operation":   "create",
			"file_path":   rel,
			"new_content": "# Findings\n\n- **File:** [Insert File Name]\n",
		},
	}
	if err := h.SendMessage(msg); err == nil {
		t.Fatalf("expected SendMessage error for placeholder proposal, got nil")
	}

	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); !os.IsNotExist(err) {
		t.Fatalf("expected placeholder proposal to be rejected, stat err=%v", err)
	}
	for _, p := range h.fileChangeManager.ListPendingFileChanges("a1") {
		if p != nil && p.Status == filechange.FileChangeStatusPending {
			t.Fatalf("unexpected pending change %s", p.ID)
		}
	}
}
