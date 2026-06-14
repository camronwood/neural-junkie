package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TaskRoutingPlan is the predicted provider/model for a collaboration task.
type TaskRoutingPlan struct {
	ProviderID string
	Model      string
	Reason     string
	Source     string
	Domain     string
	CostTier   string
}

// TaskRoutingOverrides carries explicit per-task routing metadata when set.
type TaskRoutingOverrides struct {
	ProviderID  string
	OllamaModel string
}

// CollabRouting optionally overrides the AI provider for collaboration execution tasks.
type CollabRouting interface {
	EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, collab CollaborationInfo, msg *protocol.Message) ai.AIProvider
	PlanTask(ctx context.Context, assignee protocol.AgentInfo, taskText string, overrides TaskRoutingOverrides) TaskRoutingPlan
}

var globalCollabRouting CollabRouting

// SetGlobalCollabRouting registers the server implementation (e.g. from cmd/server).
func SetGlobalCollabRouting(r CollabRouting) {
	globalCollabRouting = r
}

// GlobalCollabRouting returns the registered collaboration routing implementation.
func GlobalCollabRouting() CollabRouting {
	return globalCollabRouting
}

// EffectiveAIProvider returns the provider to use for this message (collab routing or base).
func (a *Agent) EffectiveAIProvider(ctx context.Context, msg *protocol.Message) ai.AIProvider {
	if a == nil {
		return nil
	}
	base := a.GetAIProvider()
	if globalCollabRouting == nil {
		return base
	}
	eff := globalCollabRouting.EffectiveAI(ctx, base, a.Info, a.getCollaborationContext(msg), msg)
	if msg != nil && msg.Type == protocol.MessageTypeCollabTask {
		overrides := TaskRoutingOverrides{}
		if msg.Metadata != nil {
			if pid, ok := msg.Metadata["task_provider_id"].(string); ok {
				overrides.ProviderID = pid
			}
			if m, ok := msg.Metadata["task_ollama_model"].(string); ok {
				overrides.OllamaModel = m
			}
		}
		plan := globalCollabRouting.PlanTask(ctx, a.Info, msg.Content, overrides)
		source := plan.Source
		if source == "" {
			source = "rules"
		}
		snap := RoutingSnapshot{
			ProviderID: plan.ProviderID,
			ChatModel:  plan.Model,
			Reason:     plan.Reason,
			Source:     source,
			Domain:     plan.Domain,
			CostTier:   plan.CostTier,
		}
		if eff != nil {
			if m := eff.GetModel(); snap.ChatModel == "" {
				snap.ChatModel = m
			}
		}
		a.RecordRoutingSnapshot(snap)
	}
	return eff
}
