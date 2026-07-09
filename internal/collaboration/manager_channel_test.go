package collaboration

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestGetByChannel_prefersActiveOverCancelled(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)

	cancelled, err := cm.CreateCollaboration(
		"old",
		[]string{"arch-id", "be-id"},
		"collab-scenarios",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.CancelCollaboration(cancelled.ID); err != nil {
		t.Fatal(err)
	}
	cm.mu.Lock()
	cancelled.UpdatedAt = time.Now().Add(time.Minute)
	cm.mu.Unlock()

	active, err := cm.CreateCollaboration(
		"new",
		[]string{"arch-id", "be-id"},
		"collab-scenarios",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	got := cm.GetByChannel("collab-scenarios")
	if got == nil {
		t.Fatal("expected collaboration on channel")
	}
	if got.ID != active.ID {
		t.Fatalf("GetByChannel = %s (phase=%s), want active %s; cancelled=%s",
			got.ID, got.Phase, active.ID, cancelled.ID)
	}
}
