package collaboration

import "testing"

func TestInferTaskStatusFromAgentReplyTASK_STATUSLine(t *testing.T) {
	got := InferTaskStatusFromAgentReply("Done.\nTASK_STATUS: completed\n")
	if got != TaskCompleted {
		t.Fatalf("expected completed, got %q", got)
	}
}

func TestInferTaskStatusFromAgentReplyBlocked(t *testing.T) {
	got := InferTaskStatusFromAgentReply("TASK_STATUS: blocked\nNeed credentials.")
	if got != TaskBlocked {
		t.Fatalf("expected blocked, got %q", got)
	}
}
