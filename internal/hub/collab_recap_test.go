package hub

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestNewHubWiresCollaborationRecapsBeforeApprove(t *testing.T) {
	h := NewHub()
	chName := "recap-wire"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("wire", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.PlanningRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("expected planning recap pending after hub-wired reviewing, got %s", snap.PlanningRecapStatus)
	}
	if _, err := cm.ApprovePlan(collab.ID); err == nil {
		t.Fatal("expected ApprovePlan blocked while planning recap pending")
	}
}

func TestBuildRecapPrompt_PreApproval(t *testing.T) {
	p := buildRecapPrompt(collaboration.RecapKindPreApproval, "RustExpert", "# Goal\nDo the thing")
	if !strings.Contains(p, "@RustExpert") {
		t.Fatalf("expected mention: %s", p)
	}
	if !strings.Contains(p, "/approve-plan") {
		t.Fatal("expected approve-plan hint")
	}
	if !strings.Contains(p, "Do the thing") {
		t.Fatal("expected context appended")
	}
}

func TestPendingRecapAssignee(t *testing.T) {
	snap := &collaboration.Collaboration{
		PlanningRecapStatus: collaboration.RecapStatusPending,
		PlanningRecapAgentID:  "a1",
		SessionRecapStatus:    collaboration.RecapStatusPending,
		SessionRecapAgentID:   "a2",
	}
	kind, id := pendingRecapAssignee(snap, "a1")
	if kind != collaboration.RecapKindPreApproval || id != "a1" {
		t.Fatalf("planning: kind=%q id=%q", kind, id)
	}
	kind, id = pendingRecapAssignee(snap, "a2")
	if kind != collaboration.RecapKindFinal || id != "a2" {
		t.Fatalf("final: kind=%q id=%q", kind, id)
	}
	if k, _ := pendingRecapAssignee(snap, "other"); k != "" {
		t.Fatal("unexpected match for unrelated agent")
	}
}

func TestDeterministicRecapFallback(t *testing.T) {
	snap := &collaboration.Collaboration{
		Description: "Ship feature X",
		Tasks: []collaboration.CollaborationTask{
			{Title: "Design", Status: collaboration.TaskCompleted},
		},
	}
	out := deterministicRecapFallback(snap, collaboration.RecapKindPreApproval)
	if !strings.Contains(out, "Ship feature X") || !strings.Contains(out, "/approve-plan") {
		t.Fatalf("unexpected fallback: %s", out)
	}
}

func TestMaybeProcessRecapReply_PlanningRecap(t *testing.T) {
	h := NewHub()
	chName := "recap-reply-planning"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("recap", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	_ = cm.MarkPlanningRecapDispatched(collab.ID, "a2")

	reply := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		chName,
		*a2,
		"## Planning recap\n\nWe agreed on the approach.",
	)
	reply.SetCollaborationID(collab.ID)

	if !h.maybeProcessRecapReply(reply) {
		t.Fatal("expected recap reply to be processed")
	}
	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.PlanningRecapStatus != collaboration.RecapStatusComplete {
		t.Fatalf("status=%s", snap.PlanningRecapStatus)
	}
	if !strings.Contains(snap.PlanningRecap, "agreed on the approach") {
		t.Fatalf("stored recap: %q", snap.PlanningRecap)
	}
}

func TestOnRecapTimeout_PlanningFallback(t *testing.T) {
	h := NewHub()
	chName := "recap-timeout-planning"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("timeout recap", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	_ = cm.MarkPlanningRecapDispatched(collab.ID, "a1")

	h.onRecapTimeout(collab.ID, collaboration.RecapKindPreApproval)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.PlanningRecapStatus != collaboration.RecapStatusComplete {
		t.Fatalf("expected complete after timeout fallback, got %s", snap.PlanningRecapStatus)
	}
	if strings.TrimSpace(snap.PlanningRecap) == "" {
		t.Fatal("expected non-empty planning recap from fallback")
	}
}

func TestTransitionToReviewing_DispatchesCollabRecap(t *testing.T) {
	h := NewHub()
	chName := "recap-dispatch"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("dispatch recap test", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Seed discussion so last speaker is a2
	discMsg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a2, "final planning thought")
	discMsg.SetCollaborationID(collab.ID)
	_ = cm.RecordMessage(collab.ID, discMsg)

	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	h.dispatchCollaborationRecap(snap, collaboration.RecapKindPreApproval)

	msgs, _ := h.GetMessages(chName, 50)
	var sawRecap bool
	for _, m := range msgs {
		if m != nil && m.Type == protocol.MessageTypeCollabRecap {
			sawRecap = true
			if assignee, ok := m.Metadata["recap_assignee"].(string); !ok || assignee != "a2" {
				t.Fatalf("recap_assignee=%v want a2", m.Metadata["recap_assignee"])
			}
		}
	}
	if !sawRecap {
		t.Fatal("expected collaboration_recap message after entering reviewing")
	}
	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.PlanningRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("planning_recap_status=%s", after.PlanningRecapStatus)
	}
}

func TestRequestFinalRecap_DefersFinalizeUntilReply(t *testing.T) {
	h := NewHub()
	chName := "recap-final-defer"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("final recap", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{ID: "t1", Title: "Only task", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskCompleted},
	})

	h.requestFinalRecapAndFinalize(collab.ID, chName, "All tasks are done.", collaboration.FinalizeOptions{})

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected still executing, got %s", snap.Phase)
	}
	if snap.SessionRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("session_recap_status=%s", snap.SessionRecapStatus)
	}

	reply := protocol.NewMessage(protocol.MessageTypeAnswer, chName, *a1, "Final: shipped it.")
	reply.SetCollaborationID(collab.ID)
	if !h.maybeProcessRecapReply(reply) {
		t.Fatal("expected final recap processing")
	}

	snap, _ = cm.GetCollaborationSnapshot(collab.ID)
	if snap.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected completed after final recap, got %s", snap.Phase)
	}
	if !strings.Contains(snap.SessionRecap, "shipped it") {
		t.Fatalf("session_recap=%q", snap.SessionRecap)
	}
}
