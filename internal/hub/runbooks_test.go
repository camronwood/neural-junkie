package hub

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func registerRunbookTestAgent(t *testing.T, h *Hub, id, name string, agentType protocol.AgentType) {
	t.Helper()
	ag := &protocol.AgentInfo{ID: id, Name: name, Type: agentType, Status: "active"}
	if err := h.RegisterAgent(ag); err != nil {
		t.Fatalf("RegisterAgent %s: %v", id, err)
	}
}

func createReviewingRunbook(t *testing.T, h *Hub, chName string) (collabID, collabChannel string) {
	t.Helper()
	registerRunbookTestAgent(t, h, "a1", "RustExpert", protocol.AgentTypeRust)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{
			ID: "t1", Title: "Implement", Description: "Build the feature",
			AssignedTo: "a1", AssignedName: "RustExpert",
			Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now,
		},
	}
	result, err := h.CreateRunbookSession(RunbookCreateRequest{
		Description: "Regression runbook",
		AgentIDs:    []string{"a1"},
		Channel:     chName,
		CreatedBy:   "tester",
		Tasks:       tasks,
	})
	if err != nil {
		t.Fatalf("CreateRunbookSession: %v", err)
	}
	if _, err := h.SubmitRunbookForReview(result.CollaborationID); err != nil {
		t.Fatalf("SubmitRunbookForReview: %v", err)
	}
	return result.CollaborationID, result.CollaborationChannel
}

func countCollabTaskMessages(msgs []*protocol.Message) int {
	n := 0
	for _, m := range msgs {
		if m != nil && m.Type == protocol.MessageTypeCollabTask {
			n++
		}
	}
	return n
}

func findCollabStatusContaining(msgs []*protocol.Message, needle string) *protocol.Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m == nil || m.Type != protocol.MessageTypeCollabStatus {
			continue
		}
		if strings.Contains(m.Content, needle) {
			return m
		}
	}
	return nil
}

func TestStartRunbookSandboxDefersDispatchUntilWorkspaceAck(t *testing.T) {
	h := NewHub()
	chName := "runbook-defer-dispatch"
	_ = h.CreateChannel(chName, "Runbook defer", "")

	collabID, collabCh := createReviewingRunbook(t, h, chName)

	started, err := h.StartRunbook(collabID)
	if err != nil {
		t.Fatalf("StartRunbook: %v", err)
	}
	if started.Phase != collaboration.PhaseExecuting {
		t.Fatalf("phase = %s", started.Phase)
	}
	if strings.TrimSpace(started.WorkingDirectory) == "" {
		t.Fatal("expected sandbox working_directory after start")
	}
	if started.WorkspaceAcknowledged {
		t.Fatal("sandbox runbook should not auto-ack without bound project repo")
	}

	msgs, _ := h.GetMessages(collabCh, 200)
	if countCollabTaskMessages(msgs) != 0 {
		t.Fatalf("expected 0 collab_task before workspace ack, got %d", countCollabTaskMessages(msgs))
	}
	status := findCollabStatusContaining(msgs, "Waiting for workspace confirmation")
	if status == nil {
		t.Fatal("expected execution status mentioning workspace confirmation")
	}
	if !strings.Contains(status.Content, "Runbook execution started") {
		t.Fatalf("status should mention runbook start, got: %s", status.Content)
	}
}

func TestStartRunbookDispatchesAfterWorkspaceAck(t *testing.T) {
	h := NewHub()
	chName := "runbook-dispatch-after-ack"
	_ = h.CreateChannel(chName, "Runbook ack", "")

	collabID, collabCh := createReviewingRunbook(t, h, chName)
	if _, err := h.StartRunbook(collabID); err != nil {
		t.Fatalf("StartRunbook: %v", err)
	}
	if err := h.AcknowledgeCollaborationWorkspace(collabID, ""); err != nil {
		t.Fatalf("AcknowledgeCollaborationWorkspace: %v", err)
	}

	msgs, _ := h.GetMessages(collabCh, 200)
	if countCollabTaskMessages(msgs) != 1 {
		t.Fatalf("expected 1 collab_task after workspace ack, got %d", countCollabTaskMessages(msgs))
	}

	snap, err := h.GetRunbookSnapshot(collabID)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.WorkspaceAcknowledged {
		t.Fatal("expected workspace_acknowledged after ack")
	}
	for _, task := range snap.Tasks {
		if task.ID == "t1" && !task.PromptDispatched {
			t.Fatal("expected task prompt dispatched after workspace ack")
		}
	}
}

func TestStartRunbookAutoAcksAndDispatchesWithBoundRepo(t *testing.T) {
	h := NewHub()
	chName := "runbook-auto-ack"
	_ = h.CreateChannel(chName, "Runbook auto", "")

	registerRunbookTestAgent(t, h, "a1", "RustExpert", protocol.AgentTypeRust)
	repoDir := t.TempDir()

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{
			ID: "t1", Title: "Patch", AssignedTo: "a1", AssignedName: "RustExpert",
			Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now,
		},
	}

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateRunbook(
		"Bound repo runbook",
		[]string{"a1"},
		chName,
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{
			SourceRepoPath: repoDir,
			InitialTasks:   tasks,
		},
	)
	if err != nil {
		t.Fatalf("CreateRunbook: %v", err)
	}
	collabID := collab.ID
	collabCh := "collab-" + collabID
	h.CreateChannelWithType(
		collabCh,
		collab.Title,
		chName,
		protocol.ChannelTypeCollaboration,
		"tester",
	)
	if err := cm.BindCollaborationChannel(collabID, collabCh); err != nil {
		t.Fatalf("BindCollaborationChannel: %v", err)
	}
	if _, err := cm.SubmitRunbook(collabID); err != nil {
		t.Fatalf("SubmitRunbook: %v", err)
	}

	started, err := h.StartRunbook(collabID)
	if err != nil {
		t.Fatalf("StartRunbook: %v", err)
	}
	if !started.WorkspaceAcknowledged {
		t.Fatal("expected auto workspace ack for sandbox with bound source repo")
	}

	msgs, _ := h.GetMessages(collabCh, 200)
	if countCollabTaskMessages(msgs) != 1 {
		t.Fatalf("expected 1 collab_task after auto-ack start, got %d", countCollabTaskMessages(msgs))
	}
	status := findCollabStatusContaining(msgs, "Tasks dispatched")
	if status == nil {
		t.Fatal("expected status noting tasks were dispatched")
	}
}
