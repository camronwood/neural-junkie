package test

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func registerTwoCollabAgents(t *testing.T, h *hub.Hub) {
	t.Helper()
	a1 := &protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SecurityExpert", Type: protocol.AgentTypeSecurity, Status: "active"}
	if err := h.RegisterAgent(a1); err != nil {
		t.Fatal(err)
	}
	if err := h.RegisterAgent(a2); err != nil {
		t.Fatal(err)
	}
}

func humanTester() protocol.AgentInfo {
	return protocol.AgentInfo{ID: "u1", Name: "Tester", Type: "human"}
}

// ── Hub slash commands: collaboration lifecycle (CommandHandler + SendMessage) ──

func TestSlashCollaborateSeedOmitsWorkspaceByDefault(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate @RustExpert @SecurityExpert who is the better programmer",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "Neural Junkie",
			"workspace_path": "/proj",
			"file_tree":      "internal/\n",
			"open_files":     []interface{}{},
		},
		"context_scope": "full",
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	cm := h.GetCollaborationManager()
	active := cm.ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 collaboration, got %d", len(active))
	}
	ch := active[0].Channel
	msgs, err := h.GetMessages(ch, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range msgs {
		if m == nil || m.Type != protocol.MessageTypeCollabDiscussion {
			continue
		}
		if m.Metadata != nil {
			if _, ok := m.Metadata["workspace_context"]; ok {
				t.Fatalf("seed/turn should not inherit workspace_context without --workspace, msg: %.80s", m.Content)
			}
		}
	}
}

func TestSlashCollaborateWithWorkspaceFlag(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate --workspace @RustExpert @SecurityExpert review this repo layout",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "Neural Junkie",
			"workspace_path": "/proj",
			"file_tree":      "internal/\n",
			"open_files": []interface{}{
				map[string]interface{}{"path": "main.go", "language": "go", "content": "package main"},
			},
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	cm := h.GetCollaborationManager()
	ch := cm.ListActive()[0].Channel
	msgs, err := h.GetMessages(ch, 20)
	if err != nil {
		t.Fatal(err)
	}
	foundOutline := false
	for _, m := range msgs {
		if m == nil || m.Metadata == nil {
			continue
		}
		if m.Metadata["context_scope"] == "outline" {
			foundOutline = true
			if _, ok := m.Metadata["workspace_context"]; !ok {
				t.Fatal("expected workspace_context with outline scope")
			}
			ws := m.Metadata["workspace_context"].(map[string]interface{})
			if _, hasOpen := ws["open_files"]; hasOpen {
				t.Fatal("collab --workspace should not copy open_files bodies")
			}
		}
	}
	if !foundOutline {
		t.Fatal("expected context_scope outline on collab message with --workspace")
	}
}

func TestSlashCollaboratePassesDiscussionFlags(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate --rounds 7 --messages 35 @RustExpert @SecurityExpert encrypt config files",
	)
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	cm := h.GetCollaborationManager()
	active := cm.ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 active collaboration, got %d", len(active))
	}
	c := active[0]
	if c.Discussion == nil {
		t.Fatal("expected discussion")
	}
	if c.Discussion.MaxRounds != 7 {
		t.Fatalf("expected MaxRounds 7 from flags, got %d", c.Discussion.MaxRounds)
	}
	if c.Discussion.MaxTotalMessages != 35 {
		t.Fatalf("expected MaxTotalMessages 35 from flags, got %d", c.Discussion.MaxTotalMessages)
	}
}

func TestSlashApprovePlanStartsExecution(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	if err := h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate @RustExpert @SecurityExpert build a secure CLI",
	)); err != nil {
		t.Fatalf("collaborate: %v", err)
	}

	cm := h.GetCollaborationManager()
	active := cm.ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 collaboration, got %d", len(active))
	}
	id, ch := active[0].ID, active[0].Channel
	if _, err := cm.TransitionToReviewing(id); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}

	approve := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		ch,
		humanTester(),
		"/approve-plan "+id[:8],
	)
	if err := h.SendMessage(approve); err != nil {
		t.Fatalf("approve-plan: %v", err)
	}

	got, err := cm.GetCollaboration(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected executing after /approve-plan, got %s", got.Phase)
	}

	msgs, err := h.GetMessages(ch, 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range msgs {
		if m != nil && strings.Contains(m.Content, "Plan Approved") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a channel message containing Plan Approved")
	}
}

func TestSlashApprovePlan_NoTaskMessagesUntilWorkspaceAck(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	if err := h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate @RustExpert @SecurityExpert build a secure CLI",
	)); err != nil {
		t.Fatalf("collaborate: %v", err)
	}

	cm := h.GetCollaborationManager()
	active := cm.ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 collaboration, got %d", len(active))
	}
	id, ch := active[0].ID, active[0].Channel
	if _, err := cm.TransitionToReviewing(id); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}

	approve := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		ch,
		humanTester(),
		"/approve-plan "+id[:8],
	)
	if err := h.SendMessage(approve); err != nil {
		t.Fatalf("approve-plan: %v", err)
	}

	msgs, err := h.GetMessages(ch, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range msgs {
		if m != nil && m.Type == protocol.MessageTypeCollabTask {
			t.Fatalf("did not expect collaboration_task before workspace ack, got one id=%s", m.ID[:8])
		}
	}

	ack := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		ch,
		humanTester(),
		"/ack-collab-workspace "+id[:8],
	)
	if err := h.SendMessage(ack); err != nil {
		t.Fatalf("ack-collab-workspace: %v", err)
	}

	msgs2, err := h.GetMessages(ch, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundTask := false
	for _, m := range msgs2 {
		if m != nil && m.Type == protocol.MessageTypeCollabTask {
			foundTask = true
			break
		}
	}
	if !foundTask {
		t.Fatal("expected at least one collaboration_task after workspace ack")
	}
}

func TestSlashRevisePlanReturnsToPlanning(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	if err := h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate @RustExpert @SecurityExpert audit the auth module",
	)); err != nil {
		t.Fatal(err)
	}

	cm := h.GetCollaborationManager()
	active := cm.ListActive()
	id, ch := active[0].ID, active[0].Channel
	if _, err := cm.TransitionToReviewing(id); err != nil {
		t.Fatal(err)
	}

	rev := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		ch,
		humanTester(),
		"/revise-plan "+id[:8]+" add threat modeling section",
	)
	if err := h.SendMessage(rev); err != nil {
		t.Fatalf("revise-plan: %v", err)
	}

	got, err := cm.GetCollaboration(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != collaboration.PhasePlanning {
		t.Fatalf("expected planning after revise, got %s", got.Phase)
	}
}

func TestSlashCompleteCollabRequiresForceWhenOpenTasks(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	if err := h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate @RustExpert @SecurityExpert ship the landing page",
	)); err != nil {
		t.Fatal(err)
	}

	cm := h.GetCollaborationManager()
	active := cm.ListActive()
	id, ch := active[0].ID, active[0].Channel

	_ = cm.SetTasks(id, []collaboration.CollaborationTask{
		{ID: "t1", Title: "HTML", AssignedTo: "a1", AssignedName: "RustExpert", Status: collaboration.TaskPending},
	})
	_, _ = cm.TransitionToReviewing(id)
	_, _ = cm.ApprovePlan(id)
	_, _ = cm.TransitionToExecuting(id)

	attempt := protocol.NewMessage(protocol.MessageTypeQuestion, ch, humanTester(), "/complete-collab "+id[:8])
	if err := h.SendMessage(attempt); err != nil {
		t.Fatalf("complete-collab: %v", err)
	}
	got, _ := cm.GetCollaboration(id)
	if got.Phase == collaboration.PhaseCompleted {
		t.Fatal("expected not completed without --force")
	}

	force := protocol.NewMessage(protocol.MessageTypeQuestion, ch, humanTester(), "/complete-collab "+id[:8]+" --force")
	if err := h.SendMessage(force); err != nil {
		t.Fatalf("complete-collab --force: %v", err)
	}
	got, _ = cm.GetCollaboration(id)
	if got.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected completed, got %s", got.Phase)
	}
}

func TestSlashCollabTaskDoneCompletesCollaboration(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("done test", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{ID: "t1", Title: "Only", AssignedTo: "a1", AssignedName: "RustExpert", Status: collaboration.TaskPending},
	})
	_, _ = cm.TransitionToReviewing(collab.ID)
	_, _ = cm.ApprovePlan(collab.ID)
	_, _ = cm.TransitionToExecuting(collab.ID)

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", humanTester(), "/collab-task-done "+collab.ID[:8]+" 1")
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("collab-task-done: %v", err)
	}
	got, _ := cm.GetCollaboration(collab.ID)
	if got.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected completed, got %s", got.Phase)
	}
}

func TestSlashCancelPlanFromPlanning(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	if err := h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate @RustExpert @SecurityExpert document the API",
	)); err != nil {
		t.Fatal(err)
	}

	cm := h.GetCollaborationManager()
	active := cm.ListActive()
	id, ch := active[0].ID, active[0].Channel

	cancel := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		ch,
		humanTester(),
		"/cancel-plan "+id[:8],
	)
	if err := h.SendMessage(cancel); err != nil {
		t.Fatalf("cancel-plan: %v", err)
	}

	got, err := cm.GetCollaboration(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != collaboration.PhaseCancelled {
		t.Fatalf("expected cancelled, got %s", got.Phase)
	}
}

func TestSlashCollabExtendAfterBudgetExhausted(t *testing.T) {
	h := hub.NewHub()
	_ = h.CreateChannel("extend-ch", "extend", "p")
	registerTwoCollabAgents(t, h)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"tight budget",
		[]string{"a1", "a2"},
		"extend-ch",
		"tester",
		collaboration.DiscussionConfig{
			MaxRounds:        1,
			TurnBudget:       1,
			MaxTotalMessages: 1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"extend-ch",
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"only message",
	)
	if err := cm.RecordMessage(collab.ID, msg); err != nil {
		t.Fatal(err)
	}
	got, _ := cm.GetCollaboration(collab.ID)
	if got.Discussion == nil || got.Discussion.Status != collaboration.DiscussionBudgetExhausted {
		t.Fatalf("expected budget exhausted, got %+v", got.Discussion)
	}

	ext := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"extend-ch",
		humanTester(),
		"/collab-extend "+collab.ID[:8]+" --rounds 1 --messages 4",
	)
	if err := h.SendMessage(ext); err != nil {
		t.Fatalf("collab-extend: %v", err)
	}

	after, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Discussion == nil || after.Discussion.Status != collaboration.DiscussionActive {
		t.Fatalf("expected discussion active after extend, got %+v", after.Discussion)
	}
}

func TestSlashCollabStatusListAndDetail(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	if err := h.SendMessage(protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collaborate @RustExpert @SecurityExpert ship metrics export",
	)); err != nil {
		t.Fatal(err)
	}

	cm := h.GetCollaborationManager()
	id := cm.ListActive()[0].ID

	listMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collab-status",
	)
	if err := h.SendMessage(listMsg); err != nil {
		t.Fatal(err)
	}
	msgs, err := h.GetMessages("general", 30)
	if err != nil {
		t.Fatal(err)
	}
	var lastList string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i] != nil && strings.Contains(msgs[i].Content, "Active Collaborations") {
			lastList = msgs[i].Content
			break
		}
	}
	if lastList == "" || !strings.Contains(lastList, id[:8]) {
		t.Fatalf("expected /collab-status list to mention collab prefix, got tail: %q", lastList)
	}

	detail := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/collab-status "+id[:8],
	)
	if err := h.SendMessage(detail); err != nil {
		t.Fatal(err)
	}
	msgs2, err := h.GetMessages("general", 30)
	if err != nil {
		t.Fatal(err)
	}
	foundDetail := false
	for i := len(msgs2) - 1; i >= 0; i-- {
		m := msgs2[i]
		if m == nil {
			continue
		}
		if strings.Contains(m.Content, "**Phase:**") && strings.Contains(m.Content, id[:8]) {
			foundDetail = true
			break
		}
	}
	if !foundDetail {
		t.Fatal("expected /collab-status <id> detail output with Phase line")
	}
}

func TestSlashApprovePlanUnknownID(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	m := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		humanTester(),
		"/approve-plan deadbeef",
	)
	if err := h.SendMessage(m); err != nil {
		t.Fatal(err)
	}
	msgs, err := h.GetMessages("general", 20)
	if err != nil {
		t.Fatal(err)
	}
	var last string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i] != nil && strings.Contains(msgs[i].Content, "Collaboration not found") {
			last = msgs[i].Content
			break
		}
	}
	if last == "" {
		t.Fatal("expected not-found system response for bogus collab id")
	}
}
