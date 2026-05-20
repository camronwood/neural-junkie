package collaboration

import "testing"

func TestInferTaskStatusFromAgentReplyTASK_STATUSLine(t *testing.T) {
	got := InferTaskStatusFromAgentReply("Done.\nTASK_STATUS: completed\n", false)
	if got != TaskCompleted {
		t.Fatalf("expected completed, got %q", got)
	}
}

func TestInferTaskStatusFromAgentReplyBlocked(t *testing.T) {
	got := InferTaskStatusFromAgentReply("TASK_STATUS: blocked\nNeed credentials.", false)
	if got != TaskBlocked {
		t.Fatalf("expected blocked, got %q", got)
	}
}

func TestInferTaskStatusStrictIgnoresFuzzyDone(t *testing.T) {
	got := InferTaskStatusFromAgentReply("All done and finished implementing.", true)
	if got != "" {
		t.Fatalf("strict mode should ignore fuzzy completion, got %q", got)
	}
}

func TestInferTaskStatusNonStrictFuzzyDone(t *testing.T) {
	got := InferTaskStatusFromAgentReply("All done and finished implementing.", false)
	if got != TaskCompleted {
		t.Fatalf("non-strict should infer completed, got %q", got)
	}
}
