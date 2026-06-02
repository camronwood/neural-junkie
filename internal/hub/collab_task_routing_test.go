package hub

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubCollabRouting struct{}

func (stubCollabRouting) EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, collab agent.CollaborationInfo, msg *protocol.Message) ai.AIProvider {
	return base
}

func (stubCollabRouting) PlanTask(ctx context.Context, assignee protocol.AgentInfo, taskText string, overrides agent.TaskRoutingOverrides) agent.TaskRoutingPlan {
	return agent.TaskRoutingPlan{
		ProviderID: "ollama-local",
		Model:      "qwen2.5:3b",
		Reason:     "light_local_model",
	}
}

func TestAnnotateCollaborationTaskRouting(t *testing.T) {
	prev := agent.GlobalCollabRouting()
	agent.SetGlobalCollabRouting(stubCollabRouting{})
	t.Cleanup(func() { agent.SetGlobalCollabRouting(prev) })

	h := &Hub{}
	snap := &collaboration.Collaboration{
		Agents: []collaboration.CollaborationAgent{
			{AgentID: "a1", AgentName: "Scout", AgentType: protocol.AgentTypeBackend},
		},
		Tasks: []collaboration.CollaborationTask{
			{
				ID:           "t1",
				Title:        "Identify schema files",
				AssignedTo:   "a1",
				AssignedName: "Scout",
			},
		},
	}
	h.annotateCollaborationTaskRouting(snap)
	if snap.Tasks[0].Options == nil {
		t.Fatal("expected task options to be populated")
	}
	if snap.Tasks[0].Options.ExpectedModel != "qwen2.5:3b" {
		t.Fatalf("expected_model = %q", snap.Tasks[0].Options.ExpectedModel)
	}
	if snap.Tasks[0].Options.RoutingReason != "light_local_model" {
		t.Fatalf("routing_reason = %q", snap.Tasks[0].Options.RoutingReason)
	}
}
