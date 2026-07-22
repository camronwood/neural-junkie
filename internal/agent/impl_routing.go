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
	msgID := ""
	if msg != nil {
		msgID = msg.ID
	}
	prior := a.LastRoutingSnapshotFor(msgID)
	snap.Attempts = append([]protocol.RoutingAttempt(nil), prior.Attempts...)
	hints := ImplementationRoutingHintsFromContext(ctx)
	if hints.RepairAttempts > 0 && len(snap.Attempts) > 0 {
		last := len(snap.Attempts) - 1
		if snap.Attempts[last].FailureReason == "" {
			snap.Attempts[last].FailureReason = "implementation_repair"
		}
		snap.FailureEvidence = appendUniqueString(prior.FailureEvidence, "implementation_repair")
	}
	if !routingAttemptContains(snap.Attempts, plan.ProviderID, snap.ChatModel) {
		snap.Attempts = append(snap.Attempts, protocol.RoutingAttempt{
			ProviderID: plan.ProviderID,
			Model:      snap.ChatModel,
			Tier:       implementationAttemptTier(plan.Reason),
			Reason:     plan.Reason,
		})
	}
	if plan.Reason == "reliable_local_repair_tier" || plan.Reason == "frontier_after_local_exhaustion" {
		snap.CostTier = unified.CostPremium
	}
	a.RecordRoutingSnapshotFor(msgID, snap)
	a.broadcastRoutingTelemetry(msg)
	return eff
}

func implementationAttemptTier(reason string) string {
	switch reason {
	case "reliable_local_repair_tier":
		return string(ConversationTierReliable)
	case "frontier_after_local_exhaustion":
		return "frontier"
	default:
		return string(ConversationTierStandard)
	}
}
