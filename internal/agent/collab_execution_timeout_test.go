package agent

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollabGenerationContext_appliesDeadline(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "ch", protocol.AgentInfo{Name: "A"}, "task")
	msg.Metadata = map[string]interface{}{"execution_timeout_seconds": 1}
	ctx, cancel := collabGenerationContext(context.Background(), msg)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline on collab task context")
	}
	if time.Until(deadline) > 1500*time.Millisecond {
		t.Fatalf("deadline too far: %v", deadline)
	}
}

func TestCollabGenerationContext_skipsNonCollabTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{Name: "A"}, "hi")
	msg.Metadata = map[string]interface{}{"execution_timeout_seconds": 30}
	_, cancel := collabGenerationContext(context.Background(), msg)
	if cancel == nil {
		t.Fatal("expected noop cancel")
	}
	cancel()
}
