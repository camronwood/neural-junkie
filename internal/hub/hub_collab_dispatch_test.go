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

func TestSendMessageInCollabChannelInheritsCollaborationMetadata(t *testing.T) {
	h := NewHub()

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("plan", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	chName := "collab-" + collab.ID
	h.CreateChannelWithType(chName, "collab", "general", protocol.ChannelTypeCollaboration, "tester")
	if err := cm.BindCollaborationChannel(collab.ID, chName); err != nil {
		t.Fatalf("bind channel: %v", err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		chName,
		protocol.AgentInfo{ID: "human-camronwood", Name: "camronwood", Type: protocol.AgentTypeGeneral},
		"@Gemini please resume the plan",
	)
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("send message: %v", err)
	}

	msgs, err := h.GetMessages(chName, 10)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	var stored *protocol.Message
	for _, m := range msgs {
		if m.ID == msg.ID {
			stored = m
			break
		}
	}
	if stored == nil {
		t.Fatal("expected message stored in collaboration channel")
	}
	if got := stored.GetCollaborationID(); got != collab.ID {
		t.Fatalf("expected inherited collaboration id %q, got %q", collab.ID, got)
	}
	if got := stored.GetCollaborationPhase(); got != string(collaboration.PhasePlanning) {
		t.Fatalf("expected inherited phase %q, got %q", collaboration.PhasePlanning, got)
	}
}

func TestAgentMentionDoesNotAutoAddParticipantByDefault(t *testing.T) {
	h := NewHub()
	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "a3", Name: "Claude", Type: protocol.AgentTypeCLI, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	_ = h.RegisterAgent(a3)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("plan", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	chName := "collab-" + collab.ID
	h.CreateChannelWithType(chName, "collab", "general", protocol.ChannelTypeCollaboration, "tester")
	if err := cm.BindCollaborationChannel(collab.ID, chName); err != nil {
		t.Fatal(err)
	}
	_ = h.AddAgentToChannel("a1", chName)
	_ = h.AddAgentToChannel("a2", chName)

	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a1, "@Claude should review this too")
	msg.SetCollaborationID(collab.ID)
	msg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if len(snap.Agents) != 2 {
		t.Fatalf("expected no auto-add, got agents: %+v", snap.Agents)
	}
	if len(snap.PendingParticipantRequests) != 0 {
		t.Fatalf("expected no pending requests by default, got %+v", snap.PendingParticipantRequests)
	}
}

func TestAgentMentionCreatesParticipantRequestWhenEnabled(t *testing.T) {
	h := NewHub()
	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "a3", Name: "Claude", Type: protocol.AgentTypeCLI, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	_ = h.RegisterAgent(a3)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"a1", "a2"},
		"general",
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{AllowAgentParticipantRequests: true},
	)
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	chName := "collab-" + collab.ID
	h.CreateChannelWithType(chName, "collab", "general", protocol.ChannelTypeCollaboration, "tester")
	if err := cm.BindCollaborationChannel(collab.ID, chName); err != nil {
		t.Fatal(err)
	}
	_ = h.AddAgentToChannel("a1", chName)
	_ = h.AddAgentToChannel("a2", chName)

	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a1, "@Claude should review this too")
	msg.SetCollaborationID(collab.ID)
	msg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if len(snap.Agents) != 2 {
		t.Fatalf("expected request without auto-add, got agents: %+v", snap.Agents)
	}
	if len(snap.PendingParticipantRequests) != 1 {
		t.Fatalf("expected one pending request, got %+v", snap.PendingParticipantRequests)
	}
	if snap.PendingParticipantRequests[0].AgentID != "a3" {
		t.Fatalf("expected pending Claude request, got %+v", snap.PendingParticipantRequests[0])
	}

	approved, err := h.ApproveCollaborationParticipantRequest(collab.ID, "a3")
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	if len(approved.PendingParticipantRequests) != 0 {
		t.Fatalf("expected pending request cleared, got %+v", approved.PendingParticipantRequests)
	}
	if !cm.IsParticipant(collab.ID, "a3") {
		t.Fatal("expected approved agent to become participant")
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
