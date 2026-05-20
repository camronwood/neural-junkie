package collaboration

import "testing"

func TestEvaluateEdgeConditionOnOutput(t *testing.T) {
	up := CollaborationTask{Status: TaskCompleted, Output: `{"severity":"critical"}`}
	cond := &EdgeCondition{Mode: "on_output", Contains: "critical"}
	ok, err := EvaluateEdgeCondition(cond, up)
	if err != nil || !ok {
		t.Fatalf("want match, ok=%v err=%v", ok, err)
	}
}

func TestIsTaskReadySkipBranchOnBlocked(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Status: TaskBlocked},
		{ID: "b", Status: TaskPending, Dependencies: []string{"a"}},
	}
	c := &Collaboration{
		Tasks: tasks,
		ExecutionPolicy: ExecutionPolicy{BlockedUpstreamPolicy: BlockedPolicyBlock},
	}
	if IsTaskReadyForCollab(tasks[1], c) {
		t.Fatal("block policy should wait on blocked upstream")
	}
	c.ExecutionPolicy.BlockedUpstreamPolicy = BlockedPolicySkipBranch
	if !IsTaskReadyForCollab(tasks[1], c) {
		t.Fatal("skip_branch should treat blocked upstream as satisfied")
	}
}

func TestEvaluateEdgeConditionOnStatus(t *testing.T) {
	up := CollaborationTask{Status: TaskBlocked}
	cond := &EdgeCondition{Mode: "on_status", Status: "blocked"}
	ok, err := EvaluateEdgeCondition(cond, up)
	if err != nil || !ok {
		t.Fatalf("want blocked match, ok=%v err=%v", ok, err)
	}
}

func TestEvaluateEdgeConditionRegex(t *testing.T) {
	up := CollaborationTask{Status: TaskCompleted, Output: "severity: CRITICAL alert"}
	cond := &EdgeCondition{Mode: "on_output", Regex: `(?i)critical`}
	ok, err := EvaluateEdgeCondition(cond, up)
	if err != nil || !ok {
		t.Fatalf("regex should match, ok=%v err=%v", ok, err)
	}
}

func TestIsTaskReadyWithDependencyEdgeCondition(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Status: TaskCompleted, Output: "ok"},
		{ID: "b", Status: TaskPending, Dependencies: []string{"a"}, DependencyEdges: []DependencyEdge{
			{FromTaskID: "a", Condition: &EdgeCondition{Mode: "on_output", Contains: "fail"}},
		}},
		{ID: "c", Status: TaskPending, Dependencies: []string{"a"}, DependencyEdges: []DependencyEdge{
			{FromTaskID: "a", Condition: &EdgeCondition{Mode: "on_output", Contains: "ok"}},
		}},
	}
	c := &Collaboration{Tasks: tasks}
	if IsTaskReadyForCollab(tasks[1], c) {
		t.Fatal("b should not be ready when output contains fail")
	}
	if !IsTaskReadyForCollab(tasks[2], c) {
		t.Fatal("c should be ready when output contains ok")
	}
}

func TestEvaluateEdgeConditionAlwaysRequiresCompleted(t *testing.T) {
	up := CollaborationTask{Status: TaskInProgress, Output: "partial"}
	cond := &EdgeCondition{Mode: "always"}
	ok, err := EvaluateEdgeCondition(cond, up)
	if err != nil || ok {
		t.Fatalf("always should require completed upstream, ok=%v err=%v", ok, err)
	}
}

func TestEvaluateEdgeConditionInvalidRegex(t *testing.T) {
	_, err := EvaluateEdgeCondition(&EdgeCondition{Mode: "on_output", Regex: "["}, CollaborationTask{Output: "x"})
	if err == nil {
		t.Fatal("expected regex compile error")
	}
}

func TestDependencyGroupAny(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "a", Status: TaskCompleted},
		{ID: "b", Status: TaskPending},
		{ID: "c", Status: TaskPending, DependencyGroups: []DependencyGroup{{Mode: "any", TaskIDs: []string{"a", "b"}}}},
	}
	c := &Collaboration{Tasks: tasks}
	if !IsTaskReadyForCollab(tasks[2], c) {
		t.Fatal("any group should pass when a is completed")
	}
}
