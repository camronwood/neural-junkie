package hub

import (
	"testing"
	"time"
)

func TestApproveScopedAlwaysSetsReason(t *testing.T) {
	manager := NewToolApprovalManager(nil)
	defer manager.Stop()
	approval := manager.CreateApproval("a1", "Frontend", "", "run_command", "dm-test", map[string]interface{}{
		"command": "make start-all",
		"reason":  "not_allowlisted",
	})
	done := make(chan ToolApprovalStatus, 1)
	go func() {
		status, _ := manager.WaitForDecision(approval.ID, time.Second)
		done <- status
	}()
	time.Sleep(20 * time.Millisecond)
	if err := manager.ApproveScoped(approval.ID, "always"); err != nil {
		t.Fatal(err)
	}
	if status := <-done; status != ToolApprovalApproved {
		t.Fatalf("status=%s", status)
	}
	got := manager.GetApproval(approval.ID)
	if got == nil || got.Reason != "always" {
		t.Fatalf("approval=%+v", got)
	}
}
