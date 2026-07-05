package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	unified "github.com/camronwood/neural-junkie/internal/routing"
)

// ImplementationRoutingPlan describes provider selection for an implementation session.
type ImplementationRoutingPlan struct {
	ProviderID string
	ToolModel  string
	Reason     string
}

// ImplementationRouting optionally overrides the AI provider for IDE implementation sessions.
type ImplementationRouting interface {
	Plan(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message) (ImplementationRoutingPlan, ai.AIProvider)
}

var globalImplementationRouting ImplementationRouting

// SetGlobalImplementationRouting registers the server implementation.
func SetGlobalImplementationRouting(r ImplementationRouting) {
	globalImplementationRouting = r
}

// GlobalImplementationRouting returns the registered implementation routing.
func GlobalImplementationRouting() ImplementationRouting {
	return globalImplementationRouting
}

// EffectiveImplementationProvider returns the provider for an implementation session.
func (a *Agent) EffectiveImplementationProvider(ctx context.Context, msg *protocol.Message) ai.AIProvider {
	if a == nil {
		return nil
	}
	base := a.GetAIProvider()
	if globalImplementationRouting == nil {
		return base
	}
	plan, eff := globalImplementationRouting.Plan(ctx, base, a.Info, msg)
	if eff == nil {
		return base
	}
	snap := RoutingSnapshot{
		ProviderID: plan.ProviderID,
		ToolModel:  plan.ToolModel,
		Reason:     plan.Reason,
		Source:     "rules",
	}
	if m := eff.GetModel(); m != "" {
		snap.ChatModel = m
	}
	if plan.Reason == "reliable_repair_tier" {
		snap.CostTier = unified.CostPremium
	}
	a.RecordRoutingSnapshot(snap)
	a.broadcastRoutingTelemetry(msg)
	return eff
}
