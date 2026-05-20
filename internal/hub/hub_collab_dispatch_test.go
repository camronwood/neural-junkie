package hub

import (
	"strings"
	"testing"
	"time"

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
	completePlanningRecapForHubTest(t, cm, collab.ID)
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
	approveAndExecuteCollabForTest(t, cm, collab.ID)
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

func TestDispatchWaveDAGOnlyReadyTasks(t *testing.T) {
	h := NewHub()
	chName := "test-collab-dag-wave"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "a3", Name: "AgentC", Type: protocol.AgentTypeRust, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	_ = h.RegisterAgent(a3)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("dag", []string{"a1", "a2", "a3"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	now := collab.CreatedAt
	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "A", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "B", AssignedTo: "a2", AssignedName: "AgentB", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t3", Title: "C", AssignedTo: "a3", AssignedName: "AgentC", Status: collaboration.TaskPending, Dependencies: []string{"t1", "t2"}, CreatedAt: now, UpdatedAt: now},
	}
	if err := cm.SetTasks(collab.ID, tasks); err != nil {
		t.Fatalf("SetTasks: %v", err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	n := h.dispatchReadyCollabTasks(snap, nil, false)
	if n != 2 {
		t.Fatalf("wave1: expected 2 dispatches, got %d", n)
	}
	msgs, _ := h.GetMessages(chName, 100)
	if countMessageType(msgs, protocol.MessageTypeCollabTask) != 2 {
		t.Fatalf("expected 2 collab_task messages in wave1")
	}

	_ = cm.UpdateTaskStatus(collab.ID, "t1", collaboration.TaskCompleted, "done a")
	_ = cm.UpdateTaskStatus(collab.ID, "t2", collaboration.TaskCompleted, "done b")
	snap, _ = cm.GetCollaborationSnapshot(collab.ID)
	n = h.dispatchReadyCollabTasks(snap, nil, false)
	if n != 1 {
		t.Fatalf("wave2: expected 1 dispatch for C, got %d", n)
	}
	msgs, _ = h.GetMessages(chName, 100)
	if countMessageType(msgs, protocol.MessageTypeCollabTask) != 3 {
		t.Fatalf("expected 3 collab_task messages total, got %d", countMessageType(msgs, protocol.MessageTypeCollabTask))
	}
}

func TestSendMessageTaskCompleteDispatchesDependentWave(t *testing.T) {
	h := NewHub()
	chName := "test-collab-complete-wave"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "a3", Name: "AgentC", Type: protocol.AgentTypeRust, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a3)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("wave on complete", []string{"a1", "a3"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	now := collab.CreatedAt
	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "First", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "Second", AssignedTo: "a3", AssignedName: "AgentC", Status: collaboration.TaskPending, Dependencies: []string{"t1"}, CreatedAt: now, UpdatedAt: now},
	}
	if err := cm.SetTasks(collab.ID, tasks); err != nil {
		t.Fatalf("SetTasks: %v", err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if n := h.dispatchReadyCollabTasks(snap, nil, false); n != 1 {
		t.Fatalf("wave1: want 1 dispatch, got %d", n)
	}

	reply := protocol.NewMessage(protocol.MessageTypeAnswer, chName, *a1, "Finished.\nTASK_STATUS: completed\n")
	reply.SetCollaborationID(collab.ID)
	reply.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	reply.SetTaskID("t1")
	if err := h.SendMessage(reply); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	msgs, _ := h.GetMessages(chName, 200)
	if countMessageType(msgs, protocol.MessageTypeCollabTask) != 2 {
		t.Fatalf("expected 2 collab_task after wave2, got %d", countMessageType(msgs, protocol.MessageTypeCollabTask))
	}

	snap2, _ := cm.GetCollaborationSnapshot(collab.ID)
	var t2 *collaboration.CollaborationTask
	for i := range snap2.Tasks {
		if snap2.Tasks[i].ID == "t2" {
			t2 = &snap2.Tasks[i]
			break
		}
	}
	if t2 == nil || !t2.PromptDispatched {
		t.Fatal("dependent task t2 should be prompt-dispatched after t1 completed")
	}
}

func TestDispatchHandoffIncludesUpstreamOutput(t *testing.T) {
	h := NewHub()
	chName := "test-collab-handoff"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, _ := cm.CreateCollaboration("handoff", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	now := time.Now()
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{ID: "t1", Title: "Upstream", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskCompleted, Output: "built the API", CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "Downstream", AssignedTo: "a2", AssignedName: "AgentB", Status: collaboration.TaskPending, Dependencies: []string{"t1"}, CreatedAt: now, UpdatedAt: now},
	})
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	h.dispatchReadyCollabTasks(snap, nil, false)

	msgs, _ := h.GetMessages(chName, 50)
	for _, m := range msgs {
		if m == nil || m.Type != protocol.MessageTypeCollabTask {
			continue
		}
		if m.GetTaskID() == "t2" && !strings.Contains(m.Content, "built the API") {
			t.Fatalf("downstream task prompt should include upstream output, got: %s", m.Content)
		}
	}
}
