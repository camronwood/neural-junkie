package hub

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSubmitPlanMovesPlanningToReviewing(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatalf("command handler: %v", err)
	}
	_ = h.CreateChannel("submit-plan", "submit plan", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("manual review", []string{"a1", "a2"}, "submit-plan", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if collab.Phase != collaboration.PhasePlanning {
		t.Fatalf("phase=%s want planning", collab.Phase)
	}

	if err := cm.EndDiscussion(collab.ID, collaboration.DiscussionBudgetExhausted); err != nil {
		t.Fatalf("end discussion: %v", err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"submit-plan",
		protocol.AgentInfo{ID: "human", Name: "tester", Type: protocol.AgentTypeGeneral},
		"/submit-plan "+collab.ID[:8],
	)
	out, err := ch.handleSubmitPlan(context.Background(), msg, strings.Fields(msg.Content))
	if err != nil {
		t.Fatalf("handleSubmitPlan: %v", err)
	}
	if out == nil {
		t.Fatal("expected system response")
	}

	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Phase != collaboration.PhaseReviewing {
		t.Fatalf("phase=%s want reviewing", snap.Phase)
	}
	if snap.Discussion == nil {
		t.Fatal("expected discussion snapshot")
	}
	if snap.Discussion.Status == collaboration.DiscussionActive {
		t.Fatalf("discussion status=%v want finished", snap.Discussion.Status)
	}
}

func TestSubmitPlanRejectsActivePlanningDiscussion(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatalf("command handler: %v", err)
	}
	_ = h.CreateChannel("submit-plan-active", "submit plan", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("still planning", []string{"a1", "a2"}, "submit-plan-active", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"submit-plan-active",
		protocol.AgentInfo{ID: "human", Name: "tester", Type: protocol.AgentTypeGeneral},
		"/submit-plan "+collab.ID[:8],
	)
	out, err := ch.handleSubmitPlan(context.Background(), msg, strings.Fields(msg.Content))
	if err != nil {
		t.Fatalf("handleSubmitPlan: %v", err)
	}
	if out == nil || !strings.Contains(out.Content, "Planning still in progress") {
		t.Fatalf("expected in-progress response, got: %v", out)
	}

	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Phase != collaboration.PhasePlanning {
		t.Fatalf("phase=%s want planning", snap.Phase)
	}
}
