package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestTickCollaborationIdleWatchdog_DispatchesReadyPending(t *testing.T) {
	h := newTestHub(t)
	chName := "watchdog-dispatch"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	now := time.Now()
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{
			ID:           "t1",
			Title:        "Work",
			Description:  "Do work",
			AssignedTo:   "a1",
			AssignedName: "AgentA",
			Status:       collaboration.TaskPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	})

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.TasksDispatched {
		t.Fatal("expected no dispatch yet")
	}

	h.TickCollaborationIdleWatchdog(now.Add(time.Second))

	msgs, _ := h.GetMessages(chName, 50)
	if countMessageType(msgs, protocol.MessageTypeCollabTask) == 0 {
		t.Fatal("expected watchdog to dispatch ready pending task")
	}
}

func TestCollabWatchdogRedispatchCount(t *testing.T) {
	h := NewHub()
	key := "collab:task"
	if got := h.collabWatchdogRedispatchCount(key); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
	h.collabWatchdogBumpRedispatch(key)
	if got := h.collabWatchdogRedispatchCount(key); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}
