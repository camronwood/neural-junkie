package collaboration

import "testing"

func TestExecutionTimeoutSeconds_fileVsChat(t *testing.T) {
	fileTask := CollaborationTask{Description: "Write collabs/x/findings.md"}
	if got := ExecutionTimeoutSeconds(fileTask, 0); got != DefaultCollabFileExecutionTimeoutSeconds {
		t.Fatalf("file task default: got %d want %d", got, DefaultCollabFileExecutionTimeoutSeconds)
	}
	if got := ExecutionTimeoutSeconds(fileTask, 240); got != 240 {
		t.Fatalf("file task override: got %d want 240", got)
	}
	chatTask := CollaborationTask{Description: "Summarize risks in chat"}
	if got := ExecutionTimeoutSeconds(chatTask, 240); got != DefaultCollabExecutionTimeoutSeconds {
		t.Fatalf("chat task ignores file override: got %d want %d", got, DefaultCollabExecutionTimeoutSeconds)
	}
}
