package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollabSchedulerTickDoesNotPanic(t *testing.T) {
	h := newTestHub(t)
	now := time.Now()
	NewCollabScheduler(h).Tick(now)
	h.TickCollabScheduler(now)
}

func TestCollabSchedulerOnPlanningKick(t *testing.T) {
	h := newTestHub(t)
	chName := "sched-kick"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{
		MaxRounds:        1,
		MaxTotalMessages: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Should not panic; may or may not emit handoff depending on silence/turn.
	NewCollabScheduler(h).OnPlanningKick(collab.ID)
	h.KickPlanningDiscussionWatchdog(collab.ID)
}

func TestCollabSchedulerOnPlanningNeedHandoff(t *testing.T) {
	h := newTestHub(t)
	chName := "sched-handoff"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{
		MaxRounds:        1,
		MaxTotalMessages: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !NewCollabScheduler(h).OnPlanningNeedHandoff(collab.ID, "a1") {
		t.Fatal("expected scheduler OnPlanningNeedHandoff")
	}
	msgs, _ := h.GetMessages(chName, 50)
	for _, m := range msgs {
		if m != nil && m.IsFromSystem() && m.Metadata != nil && m.Metadata["collab_turn_handoff"] == true {
			return
		}
	}
	t.Fatal("expected handoff message via scheduler")
}

func TestCollabSchedulerOnGenerationErrorRedispatches(t *testing.T) {
	h := newTestHub(t)
	chName := "sched-generr"
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

	now := time.Now().Add(-time.Minute)
	taskID := "t-sched-generr"
	if err := cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{
			ID:               taskID,
			Title:            "Work",
			Description:      "Write collabs/x/out.md",
			AssignedTo:       "a1",
			AssignedName:     "AgentA",
			Status:           collaboration.TaskInProgress,
			PromptDispatched: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}); err != nil {
		t.Fatal(err)
	}

	before, _ := h.GetMessages(chName, 100)
	nBefore := 0
	for _, m := range before {
		if m != nil && m.Type == protocol.MessageTypeCollabTask {
			nBefore++
		}
	}

	msg := protocol.NewMessage(protocol.MessageTypeAnswer, chName, *a1, "generation failed")
	msg.SetCollaborationID(collab.ID)
	msg.SetTaskID(taskID)
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["generation_error"] = true
	NewCollabScheduler(h).OnGenerationError(collab.ID, msg)

	after, _ := h.GetMessages(chName, 100)
	nAfter := 0
	for _, m := range after {
		if m != nil && m.Type == protocol.MessageTypeCollabTask {
			nAfter++
		}
	}
	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	for _, task := range snap.Tasks {
		if task.ID != taskID {
			continue
		}
		// Free assignee: clear + redispatch stamps PromptDispatched again and adds a task message.
		if nAfter <= nBefore && task.PromptDispatched {
			t.Fatalf("expected redispatch message or cleared flag; tasks=%d→%d PromptDispatched=%v", nBefore, nAfter, task.PromptDispatched)
		}
		return
	}
	t.Fatal("task missing")
}
