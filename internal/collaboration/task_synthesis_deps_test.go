package collaboration

import "testing"

func TestInferSynthesisTaskDependencies_compileFindings(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "t1", Title: "Identify schema files", Status: TaskPending},
		{ID: "t2", Title: "Analyze schema definitions", Status: TaskPending},
		{ID: "t3", Title: "Review security", Status: TaskPending},
		{ID: "t4", Title: "Compile findings from the above tasks into a markdown document", Status: TaskPending},
	}
	if !InferSynthesisTaskDependencies(tasks) {
		t.Fatal("expected dependencies inferred")
	}
	if len(tasks[3].Dependencies) != 3 {
		t.Fatalf("task 4 deps: %#v", tasks[3].Dependencies)
	}
	if tasks[3].Dependencies[0] != "t1" || tasks[3].Dependencies[2] != "t3" {
		t.Fatalf("unexpected dep order: %#v", tasks[3].Dependencies)
	}
}

func TestInferSynthesisTaskDependencies_skipsExplicitDeps(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "t1", Title: "One", Status: TaskPending},
		{ID: "t2", Title: "Compile findings from the above tasks", Status: TaskPending, Dependencies: []string{"t1"}},
	}
	if InferSynthesisTaskDependencies(tasks) {
		t.Fatal("expected no change when deps already set")
	}
	if len(tasks[1].Dependencies) != 1 {
		t.Fatalf("deps mutated: %#v", tasks[1].Dependencies)
	}
}

func TestInferSynthesisTaskDependencies_noMatch(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "t1", Title: "Identify files", Status: TaskPending},
		{ID: "t2", Title: "Write unit tests", Status: TaskPending},
	}
	if InferSynthesisTaskDependencies(tasks) {
		t.Fatal("expected no synthesis deps for unrelated tasks")
	}
}
