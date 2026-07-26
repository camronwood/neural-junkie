package hub

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
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
		PlanningRecapStatus:  collaboration.RecapStatusPending,
		PlanningRecapAgentID: "a1",
		SessionRecapStatus:   collaboration.RecapStatusPending,
		SessionRecapAgentID:  "a2",
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

	// Seed discussion so last speaker is a2 (round-robin order).
	first := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a1, "opening planning thought")
	first.SetCollaborationID(collab.ID)
	if err := cm.RecordMessage(collab.ID, first); err != nil {
		t.Fatalf("record a1: %v", err)
	}
	discMsg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a2, "final planning thought")
	discMsg.SetCollaborationID(collab.ID)
	if err := cm.RecordMessage(collab.ID, discMsg); err != nil {
		t.Fatalf("record a2: %v", err)
	}

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

func TestSessionRestoreDispatchesPendingPlanningRecapWithoutAssignee(t *testing.T) {
	h := NewHub()
	chName := "recap-pending-unassigned"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	cm.SetOnEnterReviewing(func(string) {})
	collab, err := cm.CreateCollaboration("pending without assignee", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	first := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a1, "opening planning thought")
	first.SetCollaborationID(collab.ID)
	if err := cm.RecordMessage(collab.ID, first); err != nil {
		t.Fatalf("record a1: %v", err)
	}
	discMsg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a2, "final planning thought")
	discMsg.SetCollaborationID(collab.ID)
	if err := cm.RecordMessage(collab.ID, discMsg); err != nil {
		t.Fatalf("record discussion: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}

	before, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot before recap: %v", err)
	}
	if before.PlanningRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("planning_recap_status=%s", before.PlanningRecapStatus)
	}
	if before.PlanningRecapAgentID != "" {
		t.Fatalf("expected no recap assignee before dispatch, got %q", before.PlanningRecapAgentID)
	}

	h.RedispatchOpenCollaborationTasksAfterSessionRestore()

	after, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot after recap: %v", err)
	}
	if after.PlanningRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("planning_recap_status after dispatch=%s", after.PlanningRecapStatus)
	}
	if after.PlanningRecapAgentID != "a2" {
		t.Fatalf("planning_recap_agent_id=%q want a2", after.PlanningRecapAgentID)
	}

	msgs, _ := h.GetMessages(chName, 50)
	for _, m := range msgs {
		if m != nil && m.Type == protocol.MessageTypeCollabRecap {
			if assignee, ok := m.Metadata["recap_assignee"].(string); !ok || assignee != "a2" {
				t.Fatalf("recap_assignee=%v want a2", m.Metadata["recap_assignee"])
			}
			return
		}
	}
	t.Fatal("expected collaboration_recap message for pending unassigned recap")
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

func TestRequestFinalRecap_ForceSkipsPendingSessionRecap(t *testing.T) {
	h := NewHub()
	chName := "recap-force-skip"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("force skip recap", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{ID: "t1", Title: "Only task", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskCompleted},
	})

	// Start a normal finalize (recap pending).
	h.requestFinalRecapAndFinalize(collab.ID, chName, "All tasks are done.", collaboration.FinalizeOptions{})
	before, _ := cm.GetCollaborationSnapshot(collab.ID)
	if before.SessionRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("session_recap_status=%s want pending", before.SessionRecapStatus)
	}
	if before.Phase != collaboration.PhaseExecuting {
		t.Fatalf("phase=%s want executing", before.Phase)
	}

	// Force close must not wait for the facilitator reply.
	h.requestFinalRecapAndFinalize(collab.ID, chName, "Closed by user.", collaboration.FinalizeOptions{
		SkipSessionRecap: true,
	})

	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected completed after force skip, got %s", after.Phase)
	}
	if after.AwaitingFinalize {
		t.Fatal("expected awaiting_finalize cleared")
	}
	if after.SessionRecapStatus != collaboration.RecapStatusSkipped {
		t.Fatalf("session_recap_status=%s want skipped", after.SessionRecapStatus)
	}
}

func TestRequestFinalRecap_PendingEnsuresAwaitingFinalize(t *testing.T) {
	h := NewHub()
	chName := "recap-pending-await"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("pending await", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{ID: "t1", Title: "Only task", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskCompleted},
	})

	h.requestFinalRecapAndFinalize(collab.ID, chName, "All tasks are done.", collaboration.FinalizeOptions{})
	cm.ClearAwaitingFinalize(collab.ID)

	mid, _ := cm.GetCollaborationSnapshot(collab.ID)
	if mid.SessionRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("session_recap_status=%s want pending", mid.SessionRecapStatus)
	}
	if mid.AwaitingFinalize {
		t.Fatal("expected awaiting cleared for setup")
	}

	// Second non-force close must re-arm awaiting so the pending recap still finalizes.
	h.requestFinalRecapAndFinalize(collab.ID, chName, "Marked complete by user.", collaboration.FinalizeOptions{})
	armed, _ := cm.GetCollaborationSnapshot(collab.ID)
	if !armed.AwaitingFinalize {
		t.Fatal("expected awaiting_finalize re-armed while session recap pending")
	}
	if armed.Phase != collaboration.PhaseExecuting {
		t.Fatalf("phase=%s want executing", armed.Phase)
	}
}

func TestCancelCollaboration_ClearsFinalizeAndSessionRecap(t *testing.T) {
	h := NewHub()
	chName := "recap-cancel-cleanup"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("cancel cleanup", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)

	h.requestFinalRecapAndFinalize(collab.ID, chName, "All tasks are done.", collaboration.FinalizeOptions{})

	before, _ := cm.GetCollaborationSnapshot(collab.ID)
	if !before.AwaitingFinalize {
		t.Fatal("expected awaiting_finalize before cancel")
	}
	if before.SessionRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("session_recap_status=%s want pending", before.SessionRecapStatus)
	}

	if _, err := cm.CancelCollaboration(collab.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	h.cancelCollaborationRecaps(collab.ID)

	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.AwaitingFinalize {
		t.Fatal("expected awaiting_finalize cleared after cancel")
	}
	if after.SessionRecapStatus != collaboration.RecapStatusSkipped {
		t.Fatalf("session_recap_status=%s want skipped", after.SessionRecapStatus)
	}
	if after.Phase != collaboration.PhaseCancelled {
		t.Fatalf("phase=%s want cancelled", after.Phase)
	}
}

func TestDispatchCollaborationRecap_AbortsPeerGenerations(t *testing.T) {
	h := NewHub()
	chName := "recap-abort-peers"
	_ = h.CreateChannel(chName, "collab", "test")

	handler, ok := h.GetCommandHandler().(*CommandHandler)
	if !ok || handler == nil {
		t.Fatal("expected *CommandHandler")
	}

	peer := agent.NewAgent(
		protocol.AgentTypeBackend,
		"PeerPlanner",
		nil,
		ai.NewMockProvider(),
		h,
	)
	facilitator := agent.NewAgent(
		protocol.AgentTypeArchitecture,
		"Facilitator",
		nil,
		ai.NewMockProvider(),
		h,
	)
	handler.RegisterRuntimeAgent(peer)
	handler.RegisterRuntimeAgent(facilitator)
	_ = h.RegisterAgent(&peer.Info)
	_ = h.RegisterAgent(&facilitator.Info)

	peerCtx, peerCancel := context.WithCancel(context.Background())
	defer peerCancel()
	agent.RegisterGenCancelForTest(peer, chName, peerCancel)

	facCtx, facCancel := context.WithCancel(context.Background())
	defer facCancel()
	agent.RegisterGenCancelForTest(facilitator, chName, facCancel)

	peerDone := make(chan struct{})
	go func() {
		<-peerCtx.Done()
		close(peerDone)
	}()
	facDone := make(chan struct{})
	go func() {
		<-facCtx.Done()
		close(facDone)
	}()

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"abort peers before recap",
		[]string{peer.Info.ID, facilitator.Info.ID},
		chName,
		"tester",
		collaboration.DiscussionConfig{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Seed last speaker = facilitator so SelectRecapFacilitator picks them.
	for _, a := range []*agent.Agent{peer, facilitator} {
		msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, a.Info, "planning note")
		msg.SetCollaborationID(collab.ID)
		if err := cm.RecordMessage(collab.ID, msg); err != nil {
			t.Fatalf("record %s: %v", a.Info.Name, err)
		}
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	// Transition may already have dispatched via onEnterReviewing; force another
	// dispatch path only if needed for coverage of abortCollabChannelGensExcept.
	if snap.PlanningRecapAgentID == "" {
		h.dispatchCollaborationRecap(snap, collaboration.RecapKindPreApproval)
	}

	select {
	case <-peerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("expected peer planning generation aborted before/during recap dispatch")
	}
	select {
	case <-facDone:
	case <-time.After(2 * time.Second):
		t.Fatal("expected facilitator stale generation aborted before fresh recap turn")
	}
	if agent.ActiveGenCountForTest(peer, chName) != 0 {
		t.Fatalf("peer still has active gens")
	}
	if agent.ActiveGenCountForTest(facilitator, chName) != 0 {
		t.Fatalf("facilitator still has active gens after abort-all before dispatch")
	}
}
