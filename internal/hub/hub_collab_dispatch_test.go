package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAttachCollaborationDataDoesNotDispatchTasks(t *testing.T) {
	h := NewHub()
	chName := "test-collab-dispatch"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("dispatch test", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("transition executing: %v", err)
	}
	_, _ = cm.EnsureExecutionTasks(collab.ID)
	if _, _, err := cm.AcknowledgeWorkspace(collab.ID); err != nil {
		t.Fatalf("ack workspace: %v", err)
	}

	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	h.dispatchCollabTaskMessages(snap, nil, false)

	before, _ := h.GetMessages(chName, 500)
	taskBefore := countMessageType(before, protocol.MessageTypeCollabTask)

	chat := protocol.NewMessage(protocol.MessageTypeChat, chName, *a1, "working on my task")
	chat.SetCollaborationID(collab.ID)
	chat.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	if err := h.SendMessage(chat); err != nil {
		t.Fatalf("send chat: %v", err)
	}

	after, _ := h.GetMessages(chName, 500)
	taskAfter := countMessageType(after, protocol.MessageTypeCollabTask)
	if taskAfter != taskBefore {
		t.Fatalf("expected collab task count unchanged (%d -> %d)", taskBefore, taskAfter)
	}

	snap2, _ := cm.GetCollaborationSnapshot(collab.ID)
	if !snap2.TasksDispatched {
		t.Fatal("expected TasksDispatched after initial dispatch")
	}
}

func TestDispatchCollabTaskMessagesSkipsWhenAlreadyDispatched(t *testing.T) {
	h := NewHub()
	chName := "test-collab-skip"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("skip", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, _ = cm.TransitionToReviewing(collab.ID)
	_, _ = cm.ApprovePlan(collab.ID)
	_, _ = cm.TransitionToExecuting(collab.ID)
	_, _ = cm.EnsureExecutionTasks(collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	h.dispatchCollabTaskMessages(snap, nil, false)
	msgs, _ := h.GetMessages(chName, 100)
	n1 := countMessageType(msgs, protocol.MessageTypeCollabTask)

	h.dispatchCollabTaskMessages(snap, nil, false)
	msgs, _ = h.GetMessages(chName, 100)
	n2 := countMessageType(msgs, protocol.MessageTypeCollabTask)
	if n2 != n1 {
		t.Fatalf("second dispatch without force should not add tasks: %d -> %d", n1, n2)
	}
}

func countMessageType(msgs []*protocol.Message, typ protocol.MessageType) int {
	n := 0
	for _, m := range msgs {
		if m != nil && m.Type == typ {
			n++
		}
	}
	return n
}
