package agent

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
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

func TestCollabGenerationContext_collabTaskDefault(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "ch", protocol.AgentInfo{Name: "A"}, "Write collabs/x/a.md")
	ctx, cancel := collabGenerationContext(context.Background(), msg)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected default file-task deadline")
	}
	want := time.Duration(collaboration.DefaultCollabFileExecutionTimeoutSeconds) * time.Second
	if d := time.Until(deadline); d < want-2*time.Second || d > want+2*time.Second {
		t.Fatalf("deadline %v want ~%v", d, want)
	}
}

func TestCollabGenerationContext_collabRecap(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabRecap, "ch", protocol.AgentInfo{Name: "A"}, "recap")
	ctx, cancel := collabGenerationContext(context.Background(), msg)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected recap deadline")
	}
	want := time.Duration(defaultCollabRecapTimeoutSeconds) * time.Second
	if d := time.Until(deadline); d < want-2*time.Second || d > want+2*time.Second {
		t.Fatalf("deadline %v want ~%v", d, want)
	}
}

func TestCollabGenerationContext_collabDiscussionPlanning(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch", protocol.AgentInfo{Name: "A"}, "discuss")
	msg.SetCollaborationPhase("planning")
	ctx, cancel := collabGenerationContext(context.Background(), msg)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected planning discussion deadline")
	}
	want := time.Duration(defaultCollabDiscussionTimeoutSeconds) * time.Second
	if d := time.Until(deadline); d < want-2*time.Second || d > want+2*time.Second {
		t.Fatalf("deadline %v want ~%v", d, want)
	}
}

func TestCollabGenerationContext_skipsNonCollabChat(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{Name: "A"}, "hi")
	msg.Metadata = map[string]interface{}{"execution_timeout_seconds": 30}
	ctx, cancel := collabGenerationContext(context.Background(), msg)
	defer cancel()
	if _, ok := ctx.Deadline(); ok {
		t.Fatal("chat should not get collab deadline")
	}
}
