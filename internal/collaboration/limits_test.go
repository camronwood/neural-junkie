package collaboration

import (
	"fmt"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEmptyDraftRunbookDoesNotCountAsActive(t *testing.T) {
	c := &Collaboration{
		Phase:  PhaseDraft,
		Source: SourceRunbook,
		Tasks:  nil,
	}
	if collaborationCountsAsActive(c) {
		t.Fatal("empty draft runbook should not count toward concurrent limit")
	}
	c.Tasks = []CollaborationTask{{ID: "t1", Status: TaskPending}}
	if !collaborationCountsAsActive(c) {
		t.Fatal("draft runbook with tasks should count")
	}
}

func TestMaxConcurrentIgnoresEmptyDraftRunbooks(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	h.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(h)

	for i := 0; i < MaxConcurrentCollaborations+2; i++ {
		_, err := cm.CreateRunbook("draft", []string{"a1"}, "general", "user", DiscussionConfig{}, CreateOptions{})
		if err != nil {
			t.Fatalf("empty draft runbook %d should not hit concurrent cap: %v", i, err)
		}
	}

	// Real collaborations still hit the cap
	for i := 0; i < MaxConcurrentCollaborations; i++ {
		_, err := cm.CreateCollaboration("real", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
		if err != nil {
			t.Fatalf("collaboration %d: %v", i, err)
		}
	}
	_, err := cm.CreateCollaboration("one too many", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if err == nil {
		t.Fatal("expected concurrent limit for non-empty-phase collabs")
	}
}

func TestCreateRunbookReusesEmptyDraft(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)

	first, err := cm.CreateRunbook("first", []string{"a1"}, "general", "camron", DiscussionConfig{}, CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := cm.CreateRunbook("second", []string{"a1"}, "general", "camron", DiscussionConfig{}, CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected reuse of %s, got new %s", first.ID[:8], second.ID[:8])
	}
	if second.Description != "second" {
		t.Fatalf("description = %q", second.Description)
	}
}

func TestUpdateRunbookCapsTaskCount(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	c, err := cm.CreateRunbook("cap", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	over := make([]CollaborationTask, HardMaxTasksPerCollaboration+5)
	for i := range over {
		over[i] = CollaborationTask{
			ID: fmt.Sprintf("t%d", i), Title: "T", Status: TaskPending, CreatedAt: now, UpdatedAt: now,
		}
	}
	snap, err := cm.UpdateRunbook(c.ID, RunbookUpdatePayload{Tasks: over})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Tasks) != HardMaxTasksPerCollaboration {
		t.Fatalf("tasks len = %d, want %d", len(snap.Tasks), HardMaxTasksPerCollaboration)
	}
}
