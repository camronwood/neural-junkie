package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/pipeline"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestDefaultTurnPipelineStepOrder(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "Test", ID: "a1"}}
	st := &turnState{
		agent:   a,
		msg:     protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "hello"),
		outcome: turnContinue,
	}
	steps := a.defaultTurnPipeline(st)
	var names []string
	for _, s := range steps {
		names = append(names, s.Name())
	}
	want := []string{
		"prepare_turn", "intent_classify", "knowledge_plan", "knowledge_execute",
		"governance_record", "provider_route", "generate", "post_process",
		"stamp_metadata", "deliver_response",
	}
	if len(names) != len(want) {
		t.Fatalf("steps=%v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("step[%d]=%q want %q", i, names[i], want[i])
		}
	}
}

func TestPipelineRunWithHooks(t *testing.T) {
	var ran []string
	steps := []pipeline.Step{
		pipeline.FuncStep{StepName: "intent_classify", Fn: func(ctx context.Context) error {
			ran = append(ran, "intent_classify")
			return nil
		}},
	}
	if err := pipeline.RunWithHooks(context.Background(), steps, func(n string) { ran = append(ran, "before:"+n) }, func(n string) { ran = append(ran, "after:"+n) }); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 3 {
		t.Fatalf("ran=%v", ran)
	}
}
