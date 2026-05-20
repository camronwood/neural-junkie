package collaboration

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestNormalizeDependenciesByIndex(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "id1", Title: "A", Status: TaskPending},
		{ID: "id2", Title: "B", Status: TaskPending, Dependencies: []string{"1"}},
		{ID: "id3", Title: "C", Status: TaskPending, Dependencies: []string{"1", "2"}},
	}
	NormalizeDependencies(tasks)
	if len(tasks[1].Dependencies) != 1 || tasks[1].Dependencies[0] != "id1" {
		t.Fatalf("task2 deps: %#v", tasks[1].Dependencies)
	}
	if len(tasks[2].Dependencies) != 2 {
		t.Fatalf("task3 deps: %#v", tasks[2].Dependencies)
	}
}

func TestValidateDAGCycle(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Dependencies: []string{"b"}},
		{ID: "b", Dependencies: []string{"a"}},
	}
	if err := ValidateDAG(tasks); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestReadyTasksWaitsOnDeps(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Status: TaskPending},
		{ID: "b", Status: TaskPending, Dependencies: []string{"a"}},
	}
	ready := ReadyTasks(tasks)
	if len(ready) != 1 || ready[0].ID != "a" {
		t.Fatalf("expected only a ready, got %#v", ready)
	}
	tasks[0].Status = TaskCompleted
	ready = ReadyTasks(tasks)
	if len(ready) != 1 || ready[0].ID != "b" {
		t.Fatalf("expected b ready after a done, got %#v", ready)
	}
}

func TestReadyTasksSkipsDispatched(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Status: TaskPending, PromptDispatched: true},
	}
	if len(ReadyTasks(tasks)) != 0 {
		t.Fatal("dispatched task should not be ready for dispatch again")
	}
}

func TestParseDependencyRefs(t *testing.T) {
	refs := ParseDependencyRefs("depends: 1, 2")
	if len(refs) != 2 || refs[0] != "1" || refs[1] != "2" {
		t.Fatalf("refs: %#v", refs)
	}
}

func TestValidateDAGUnknownDependency(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Dependencies: []string{"missing-id"}},
	}
	if err := ValidateDAG(tasks); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected unknown dependency error, got %v", err)
	}
}

func TestFormatDependencyHandoffIncludesOutput(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Title: "Scaffold", AssignedName: "Rust", Status: TaskCompleted, Output: "created main.rs"},
		{ID: "b", Title: "Integrate", Dependencies: []string{"a"}, Status: TaskPending},
	}
	handoff := FormatDependencyHandoff(tasks[1], tasks)
	if !strings.Contains(handoff, "Scaffold") || !strings.Contains(handoff, "created main.rs") {
		t.Fatalf("handoff missing upstream context: %q", handoff)
	}
}

func TestFormatDependencyHandoffWithLimitTruncates(t *testing.T) {
	long := strings.Repeat("x", 500)
	tasks := []CollaborationTask{
		{ID: "a", Title: "Upstream", AssignedName: "A", Status: TaskCompleted, Output: long},
		{ID: "b", Title: "Down", Dependencies: []string{"a"}, Status: TaskPending},
	}
	handoff := FormatDependencyHandoffWithLimit(tasks[1], tasks, 80)
	if !strings.Contains(handoff, "truncated") {
		t.Fatalf("expected truncation marker, got %q", handoff)
	}
	if strings.Contains(handoff, strings.Repeat("x", 200)) {
		t.Fatal("handoff should not include full upstream output")
	}
}

func TestSetTasksRejectsInvalidDAG(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	h.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateCollaboration("c", []string{"a1", "a2"}, "general", "u", DiscussionConfig{})

	now := time.Now()
	err := cm.SetTasks(c.ID, []CollaborationTask{
		{ID: "x", Title: "X", Status: TaskPending, Dependencies: []string{"y"}, CreatedAt: now, UpdatedAt: now},
		{ID: "y", Title: "Y", Status: TaskPending, Dependencies: []string{"x"}, CreatedAt: now, UpdatedAt: now},
	})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("SetTasks should reject cycle, got %v", err)
	}
}

func TestBlockedBy(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Title: "Scaffold", Status: TaskInProgress},
		{ID: "b", Title: "Integrate", Status: TaskPending, Dependencies: []string{"a"}},
	}
	blocked := BlockedBy(tasks[1], tasks)
	if len(blocked) != 1 || blocked[0] != "Scaffold" {
		t.Fatalf("blocked: %#v", blocked)
	}
}
