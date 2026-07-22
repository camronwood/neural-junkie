package hub

import (
	"testing"
	"time"
)

func TestRestoredToolApprovalCoalescesAndResumesAllWaiters(t *testing.T) {
	manager := NewToolApprovalManager(nil)
	defer manager.Stop()
	restored := &ToolApproval{
		ID: "approval", SessionID: "session", ToolName: "write_file",
		Channel: "general", ToolInput: map[string]any{"path": "x"},
		Status: ToolApprovalPending, CreatedAt: time.Now(),
	}
	manager.RestorePending(restored)
	if got := manager.CreateApproval(
		"agent", "Agent", "session", "write_file", "general", map[string]any{"path": "x"},
	); got.ID != restored.ID {
		t.Fatalf("created duplicate approval %s", got.ID)
	}

	results := make(chan ToolApprovalStatus, 2)
	for i := 0; i < 2; i++ {
		go func() {
			status, _ := manager.WaitForDecision(restored.ID, time.Second)
			results <- status
		}()
	}
	deadline := time.Now().Add(time.Second)
	for {
		manager.mu.Lock()
		count := len(manager.waiters[restored.ID])
		manager.mu.Unlock()
		if count == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("waiters did not register")
		}
		time.Sleep(time.Millisecond)
	}
	if err := manager.Approve(restored.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if status := <-results; status != ToolApprovalApproved {
			t.Fatalf("waiter status=%s", status)
		}
	}
}
