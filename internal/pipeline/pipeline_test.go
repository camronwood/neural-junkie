package pipeline

import (
	"context"
	"testing"
)

func TestRunOrder(t *testing.T) {
	var order []string
	steps := []Step{
		FuncStep{StepName: "a", Fn: func(ctx context.Context) error { order = append(order, "a"); return nil }},
		FuncStep{StepName: "b", Fn: func(ctx context.Context) error { order = append(order, "b"); return nil }},
	}
	if err := Run(context.Background(), steps); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("order=%v", order)
	}
}

func TestRunWithHooks(t *testing.T) {
	var names []string
	steps := []Step{
		FuncStep{StepName: "step1", Fn: func(ctx context.Context) error { return nil }},
	}
	err := RunWithHooks(context.Background(), steps,
		func(name string) { names = append(names, "before:"+name) },
		func(name string) { names = append(names, "after:"+name) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("names=%v", names)
	}
}
