package test

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/collaboration/actions"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func TestRunbookScenario_health_check_branch(t *testing.T) {
	def := runbooklibrary.RunbookDefinition{
		ID:    "health-check-alert",
		Title: "Health check",
		Inputs: []runbooklibrary.RunInputSpec{
			{Key: "health_url", Default: "https://httpbin.org/status/500"},
		},
		Tasks: []collaboration.CollaborationTask{
			{
				ID: "health-check", Title: "Health check", Kind: collaboration.TaskKindAction,
				Action: &collaboration.TaskActionSpec{
					Type: "http_get",
					Config: map[string]interface{}{"url": "{{inputs.health_url}}"},
				},
			},
			{
				ID: "notify-fail", Title: "Notify", Kind: collaboration.TaskKindAction,
				Dependencies: []string{"health-check"},
				DependencyEdges: []collaboration.DependencyEdge{{
					FromTaskID: "health-check",
					Condition:  &collaboration.EdgeCondition{Mode: "on_output", Contains: "500"},
				}},
				Action: &collaboration.TaskActionSpec{Type: "webhook", Config: map[string]interface{}{"url": "https://httpbin.org/post"}},
			},
		},
	}
	inputs := runbooklibrary.MergeInputDefaults(&def, map[string]string{"health_url": "https://httpbin.org/status/500"})
	tasks := runbooklibrary.ApplyInputsToTasks(def.Tasks, nil, inputs)
	if err := collaboration.ValidateDAG(tasks); err != nil {
		t.Fatal(err)
	}
	collab := &collaboration.Collaboration{Phase: collaboration.PhaseExecuting, Tasks: tasks, RunInputs: inputs}
	for i := range collab.Tasks {
		collab.Tasks[i].Status = collaboration.TaskPending
	}
	ready := collaboration.ReadyTasksForCollab(collab)
	if len(ready) != 1 || ready[0].ID != "health-check" {
		t.Fatalf("expected health-check ready, got %v", ready)
	}
	runner := actions.NewRunner(actions.Config{})
	out, err := runner.Execute(t.Context(), collab, ready[0])
	if err != nil {
		t.Fatal(err)
	}
	collab.Tasks[0].Status = collaboration.TaskCompleted
	collab.Tasks[0].Output = out
	ready = collaboration.ReadyTasksForCollab(collab)
	foundNotify := false
	for _, r := range ready {
		if r.ID == "notify-fail" {
			foundNotify = true
		}
	}
	if !foundNotify {
		t.Fatal("expected notify-fail branch after 500 response")
	}
}
