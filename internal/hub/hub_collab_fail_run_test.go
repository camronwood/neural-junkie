package hub

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestFailRunPolicyCancelsOnBlockedTaskReply(t *testing.T) {
	h := NewHub()
	chName := "general"
	_ = h.CreateChannel(chName, "General", "")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)

	now := time.Now()
	policy := collaboration.ExecutionPolicy{BlockedUpstreamPolicy: collaboration.BlockedPolicyFailRun}
	result, err := h.CreateRunbookSession(RunbookCreateRequest{
		Description: "fail run",
		AgentIDs:    []string{"a1"},
		Channel:     chName,
		CreatedBy:   "tester",
		Tasks: []collaboration.CollaborationTask{
			{ID: "t1", Title: "Work", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := h.UpdateRunbookSession(result.CollaborationID, collaboration.RunbookUpdatePayload{ExecutionPolicy: &policy}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.SubmitRunbookForReview(result.CollaborationID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.StartRunbook(result.CollaborationID); err != nil {
		t.Fatal(err)
	}
	if err := h.AcknowledgeCollaborationWorkspace(result.CollaborationID, ""); err != nil {
		t.Fatal(err)
	}

	ch := result.CollaborationChannel
	reply := protocol.NewMessage(protocol.MessageTypeAnswer, ch, *a1, "TASK_STATUS: blocked\nNeed credentials.")
	reply.SetCollaborationID(result.CollaborationID)
	reply.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	reply.SetTaskID("t1")
	if err := h.SendMessage(reply); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	snap, err := h.GetCollaborationManager().GetCollaborationSnapshot(result.CollaborationID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Phase != collaboration.PhaseCancelled {
		t.Fatalf("phase = %s, want cancelled", snap.Phase)
	}

	msgs, _ := h.GetMessages(ch, 50)
	found := false
	for _, m := range msgs {
		if m != nil && strings.Contains(m.Content, "Run stopped") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected fail_run system broadcast")
	}
}
