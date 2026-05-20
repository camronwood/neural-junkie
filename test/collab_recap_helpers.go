package test

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const testPlanningRecapBody = "Test planning session recap for approval."

// ensurePlanningRecapComplete marks the pre-approval recap done so /approve-plan can proceed.
// Required when tests use hub.NewHub() (recap callback sets planning_recap_status pending).
func ensurePlanningRecapComplete(t *testing.T, cm *collaboration.CollaborationManager, collabID string) {
	t.Helper()
	snap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil {
		t.Fatalf("GetCollaborationSnapshot: %v", err)
	}
	if snap.PlanningRecapStatus == collaboration.RecapStatusComplete {
		return
	}
	agentID := snap.PlanningRecapAgentID
	if agentID == "" {
		agentID = "a1"
	}
	if err := cm.CompletePlanningRecap(collabID, agentID, testPlanningRecapBody); err != nil {
		t.Fatalf("CompletePlanningRecap: %v", err)
	}
}

// submitSessionRecapReply simulates the facilitator posting the final session summary.
func submitSessionRecapReply(t *testing.T, h *hub.Hub, collabID, channel, agentID, agentName string, body string) {
	t.Helper()
	if agentName == "" {
		agentName = "Agent"
	}
	reply := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		channel,
		protocol.AgentInfo{ID: agentID, Name: agentName, Type: protocol.AgentTypeRust},
		body,
	)
	reply.SetCollaborationID(collabID)
	reply.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	if err := h.SendMessage(reply); err != nil {
		t.Fatalf("SendMessage session recap: %v", err)
	}
}

// finishExecutingCollabViaRecap posts the final recap if the collaboration is awaiting it.
func finishExecutingCollabViaRecap(t *testing.T, h *hub.Hub, collabID string) {
	t.Helper()
	cm := h.GetCollaborationManager()
	snap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil {
		t.Fatalf("GetCollaborationSnapshot: %v", err)
	}
	if snap.SessionRecapStatus == collaboration.RecapStatusComplete {
		return
	}
	agentID := snap.SessionRecapAgentID
	agentName := "RustExpert"
	if agentID == "" {
		agentID = "a1"
	}
	for _, a := range snap.Agents {
		if a.AgentID == agentID {
			agentName = a.AgentName
			break
		}
	}
	submitSessionRecapReply(t, h, collabID, snap.Channel, agentID, agentName,
		"### Final session summary\n\n- Research and implementation complete.\n- No open blockers.")
}
