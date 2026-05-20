package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// completePlanningRecapForHubTest marks planning recap complete so ApprovePlan succeeds in hub tests.
func completePlanningRecapForHubTest(t *testing.T, cm *collaboration.CollaborationManager, collabID string) {
	t.Helper()
	snap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.PlanningRecapStatus == collaboration.RecapStatusComplete {
		return
	}
	agentID := "a1"
	if snap.PlanningRecapAgentID != "" {
		agentID = snap.PlanningRecapAgentID
	}
	if err := cm.CompletePlanningRecap(collabID, agentID, "test planning recap"); err != nil {
		t.Fatalf("CompletePlanningRecap: %v", err)
	}
}

// approveAndExecuteCollabForTest runs reviewing → recap → approve → executing for hub integration tests.
func approveAndExecuteCollabForTest(t *testing.T, cm *collaboration.CollaborationManager, collabID string) {
	t.Helper()
	if _, err := cm.TransitionToReviewing(collabID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	completePlanningRecapForHubTest(t, cm, collabID)
	if _, err := cm.ApprovePlan(collabID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collabID); err != nil {
		t.Fatalf("TransitionToExecuting: %v", err)
	}
}
