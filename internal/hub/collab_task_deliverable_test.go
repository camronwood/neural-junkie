package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollabTaskDeliverableSatisfied_ignoresPlanStubs(t *testing.T) {
	h := newTestHub(t)
	root := t.TempDir()
	stubPath := filepath.Join(root, "findings.md")
	_ = os.WriteFile(stubPath, []byte("# findings.md\n\n_Initial stub created when the plan was approved. Replace with task output._\n"), 0644)

	snap := &collaboration.Collaboration{
		WorkingDirectory: root,
	}
	task := &collaboration.CollaborationTask{
		Description: "Write collabs/test/findings.md",
	}
	if h.collabTaskDeliverableSatisfied(snap, task, nil) {
		t.Fatal("plan stub file should not satisfy deliverable check")
	}
}

func TestMaybeWarnPrematureTaskCompletion_BlocksWithoutProposal(t *testing.T) {
	h := newTestHub(t)
	chName := "deliverable-guard"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"write doc",
		[]string{"a1", "a2"},
		chName,
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{SourceRepoPath: t.TempDir()},
	)
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _ = cm.EnsureExecutionTasks(collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if len(snap.Tasks) == 0 {
		t.Fatal("expected tasks")
	}
	task := snap.Tasks[0]
	task.Title = "Write findings"
	task.Description = "Write collabs/test/findings.md with three bullets"
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{task})

	reply := protocol.NewMessage(protocol.MessageTypeAnswer, chName, *a1, "All done.\nTASK_STATUS: completed\n")
	reply.SetCollaborationID(collab.ID)
	reply.SetTaskID(task.ID)

	h.maybeUpdateTaskStatus(reply, collab.ID)

	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.Tasks[0].Status == collaboration.TaskCompleted {
		t.Fatalf("expected task to stay open without file deliverable, got %s", after.Tasks[0].Status)
	}
}

func TestMaybeWarnPrematureTaskCompletion_AllowsWithFileChange(t *testing.T) {
	h := newTestHub(t)
	chName := "deliverable-ok"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	repoRoot := t.TempDir()
	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("write doc", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{}, collaboration.CreateOptions{SourceRepoPath: repoRoot})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _ = cm.EnsureExecutionTasks(collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if len(snap.Tasks) == 0 {
		t.Fatal("expected tasks")
	}
	task := snap.Tasks[0]
	rel := collaboration.ProjectCollabRelPath(collab.ID) + "/findings.md"
	task.Description = "Write " + rel
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{task})

	dest := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("# Findings\n\n- one\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reply := protocol.NewMessage(protocol.MessageTypeAnswer, chName, *a1, "Here.\n[FILE_CHANGE]\npath: "+rel+"\nTASK_STATUS: completed\n")
	reply.SetCollaborationID(collab.ID)
	reply.SetTaskID(task.ID)

	h.maybeUpdateTaskStatus(reply, collab.ID)

	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.Tasks[0].Status != collaboration.TaskCompleted {
		t.Fatalf("expected completed with on-disk deliverable, got %s", after.Tasks[0].Status)
	}
}

func TestMaybeWarnPrematureTaskCompletion_BlocksFileChangeTextOnly(t *testing.T) {
	h := newTestHub(t)
	chName := "deliverable-text-only"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("write doc", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _ = cm.EnsureExecutionTasks(collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if len(snap.Tasks) == 0 {
		t.Fatal("expected tasks")
	}
	task := snap.Tasks[0]
	task.Description = "Write collabs/test/findings.md"
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{task})

	reply := protocol.NewMessage(protocol.MessageTypeAnswer, chName, *a1, "Here.\n[FILE_CHANGE]\npath: collabs/test/findings.md\nTASK_STATUS: completed\n")
	reply.SetCollaborationID(collab.ID)
	reply.SetTaskID(task.ID)

	h.maybeUpdateTaskStatus(reply, collab.ID)

	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.Tasks[0].Status == collaboration.TaskCompleted {
		t.Fatalf("expected task to stay open when only [FILE_CHANGE] text is present, got %s", after.Tasks[0].Status)
	}
}
