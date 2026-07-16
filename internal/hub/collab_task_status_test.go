package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestPlanHandoffDoesNotCompleteFileDeliverableWithoutFile(t *testing.T) {
	h := newTestHub(t)
	chName := "plan-handoff-gate"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("write doc", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{}, collaboration.CreateOptions{
		SourceRepoPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	now := time.Now()
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{
			ID:           "t-findings",
			Title:        "Write findings",
			Description:  "Write collabs/test/findings.md summarizing results",
			AssignedTo:   "a1",
			AssignedName: "AgentA",
			Status:       collaboration.TaskInProgress,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "t-chat",
			Title:        "Discuss approach",
			Description:  "Share thoughts in chat only",
			AssignedTo:   "a2",
			AssignedName: "AgentB",
			Status:       collaboration.TaskInProgress,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	})

	handoff := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		chName,
		*a1,
		"Handoff:\nTask 1 (@AgentA) — Complete\nTask 2 (@AgentB) — Complete\n",
	)
	handoff.SetCollaborationID(collab.ID)
	handoff.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	h.maybeSyncTaskStatusFromPlanHandoff(handoff, collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	byID := map[string]collaboration.CollaborationTask{}
	for _, task := range snap.Tasks {
		byID[task.ID] = task
	}
	if byID["t-findings"].Status == collaboration.TaskCompleted {
		t.Fatal("file deliverable must not complete from plan-handoff prose without file")
	}
	if byID["t-chat"].Status != collaboration.TaskCompleted {
		t.Fatalf("non-file task should complete from handoff, got %s", byID["t-chat"].Status)
	}
}
